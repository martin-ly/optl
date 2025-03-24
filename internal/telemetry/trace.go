package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TraceProvider 封装 trace provider 和 cleanup 函数
type TraceProvider struct {
	provider *sdktrace.TracerProvider
	cleanup  func() error
}

// SetupTracing 配置追踪功能
func SetupTracing(cfg Config) (*TraceProvider, error) {
	// 创建资源属性
	res, err := createResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 配置导出器
	var (
		exporter sdktrace.SpanExporter
		cleanup  func() error
	)

	if cfg.EnableConsoleExporter {
		consoleExporter, err := stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout exporter: %w", err)
		}

		if exporter == nil {
			exporter = consoleExporter
			cleanup = func() error {
				return consoleExporter.Shutdown(context.Background())
			}
		} else {
			// 多导出器组合
			multiExporter := newMultiSpanExporter(exporter, consoleExporter)
			//bsp := sdktrace.NewBatchSpanProcessor(multiExporter)
			exporter = multiExporter
			oldCleanup := cleanup
			cleanup = func() error {
				err1 := oldCleanup()
				err2 := consoleExporter.Shutdown(context.Background())
				if err1 != nil {
					return err1
				}
				return err2
			}
		}
	}

	// 添加 OTLP 导出器
	if cfg.OTLPEndpoint != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(ctx, cfg.OTLPEndpoint,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to OTLP endpoint: %w", err)
		}

		otlpExporter, err := otlptrace.New(
			context.Background(),
			otlptracegrpc.NewClient(
				otlptracegrpc.WithGRPCConn(conn),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}

		if exporter == nil {
			exporter = otlpExporter
			cleanup = func() error {
				return otlpExporter.Shutdown(context.Background())
			}
		} else {
			// 多导出器组合
			multiExporter := newMultiSpanExporter(exporter, otlpExporter)
			oldCleanup := cleanup
			cleanup = func() error {
				err1 := oldCleanup()
				err2 := otlpExporter.Shutdown(context.Background())
				if err1 != nil {
					return err1
				}
				return err2
			}
			exporter = multiExporter
		}
	}

	// 配置采样器
	var sampler sdktrace.Sampler
	if cfg.SamplingRatio >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.SamplingRatio <= 0.0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(cfg.SamplingRatio)
	}

	// 配置处理器
	bsp := sdktrace.NewBatchSpanProcessor(
		exporter,
		sdktrace.WithBatchTimeout(cfg.BatchTimeout),
		sdktrace.WithMaxExportBatchSize(cfg.MaxExportBatchSize),
	)

	// 创建 provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
		sdktrace.WithSpanProcessor(bsp),
	)

	// 设置全局 provider
	otel.SetTracerProvider(tp)

	// 设置全局传播器
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &TraceProvider{
		provider: tp,
		cleanup:  cleanup,
	}, nil
}

// Shutdown 关闭 trace provider
func (tp *TraceProvider) Shutdown(ctx context.Context) error {
	err := tp.provider.Shutdown(ctx)
	if err != nil {
		return err
	}
	if tp.cleanup != nil {
		return tp.cleanup()
	}
	return nil
}

// Tracer 通过全局 provider 获取 tracer
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// multiSpanExporter 实现多导出器组合
type multiSpanExporter []sdktrace.SpanExporter

func newMultiSpanExporter(exporters ...sdktrace.SpanExporter) sdktrace.SpanExporter {
	return multiSpanExporter(exporters)
}

func (e multiSpanExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, exporter := range e {
		if err := exporter.ExportSpans(ctx, spans); err != nil {
			return err
		}
	}
	return nil
}

func (e multiSpanExporter) Shutdown(ctx context.Context) error {
	var errs []error
	for _, exporter := range e {
		if err := exporter.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("multiple errors during shutdown: %v", errs)
	}
	return nil
}

// createResource 创建并配置资源信息
func createResource(cfg Config) (*resource.Resource, error) {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
			//semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	// 添加额外的资源属性
	if len(cfg.ResourceAttributes) > 0 {
		attrs := make([]attribute.KeyValue, 0, len(cfg.ResourceAttributes))
		for k, v := range cfg.ResourceAttributes {
			attrs = append(attrs, attribute.String(k, v))
		}
		extraAttrs := resource.NewWithAttributes(semconv.SchemaURL, attrs...)
		r, err = resource.Merge(r, extraAttrs)
		if err != nil {
			return nil, err
		}
	}

	return r, nil
}
