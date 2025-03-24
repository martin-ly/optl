package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/metric"

	"go.opentelemetry.io/otel/sdk/metric/aggregation"

	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// MetricProvider 封装 metric provider 和 cleanup 函数
type MetricProvider struct {
	controller *controller.Controller
	cleanup    func() error
}

// SetupMetrics 配置指标监控功能
func SetupMetrics(cfg Config) (*MetricProvider, error) {
	if !cfg.EnableMetrics {
		return nil, nil
	}

	// 创建资源属性
	res, err := createResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 创建导出器
	var (
		exporter metric.Exporter
		cleanup  func() error
	)

	// 控制台导出器
	if cfg.EnableConsoleExporter {
		consoleExporter, err := stdoutmetric.New(
			stdoutmetric.WithPrettyPrint(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout metric exporter: %w", err)
		}

		if exporter == nil {
			exporter = consoleExporter
			cleanup = func() error {
				return consoleExporter.Shutdown(context.Background())
			}
		} else {
			multiExporter := newMultiMetricExporter(exporter, consoleExporter)
			oldCleanup := cleanup
			cleanup = func() error {
				err1 := oldCleanup()
				err2 := consoleExporter.Shutdown(context.Background())
				if err1 != nil {
					return err1
				}
				return err2
			}
			exporter = multiExporter
		}
	}

	// OTLP 导出器
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

		otlpExporter, err := otlpmetricgrpc.New(
			context.Background(),
			otlpmetricgrpc.WithGRPCConn(conn),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
		}

		if exporter == nil {
			exporter = otlpExporter
			cleanup = func() error {
				return otlpExporter.Shutdown(context.Background())
			}
		} else {
			multiExporter := newMultiMetricExporter(exporter, otlpExporter)
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

	// 配置处理器和控制器
	selector := simple.NewWithHistogramDistribution(
		simple.WithExplicitBoundaries([]float64{0.5, 1, 2, 5, 10, 20, 50, 100, 200, 500, 1000}),
	)

	proc := processor.NewFactory(
		selector,
		aggregation.CumulativeTemporalitySelector(),
	)

	cont := controller.New(
		proc,
		controller.WithResource(res),
		controller.WithExporter(exporter),
		controller.WithCollectPeriod(cfg.MetricCollectionInterval),
	)

	err = cont.Start(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to start metric controller: %w", err)
	}

	// 设置全局 provider
	otel.SetMeterProvider(cont.MeterProvider())

	// 启用 runtime 指标（可选）
	err = runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second))
	if err != nil {
		return nil, fmt.Errorf("failed to start runtime metrics: %w", err)
	}

	return &MetricProvider{
		controller: cont,
		cleanup:    cleanup,
	}, nil
}

// Shutdown 关闭 metric provider
func (mp *MetricProvider) Shutdown(ctx context.Context) error {
	err := mp.controller.Stop(ctx)
	if err != nil {
		return err
	}
	if mp.cleanup != nil {
		return mp.cleanup()
	}
	return nil
}

// Meter 通过全局 provider 获取 meter
func Meter(name string) metric.Meter {
	return otel.Meter(name)
}

// multiMetricExporter 实现多导出器组合
type multiMetricExporter []metric.Exporter

func newMultiMetricExporter(exporters ...metric.Exporter) metric.Exporter {
	return multiMetricExporter(exporters)
}

func (e multiMetricExporter) Export(ctx context.Context, checkpointSet metric.ExportKindSelector) error {
	for _, exporter := range e {
		if err := exporter.Export(ctx, checkpointSet); err != nil {
			return err
		}
	}
	return nil
}

func (e multiMetricExporter) Shutdown(ctx context.Context) error {
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

// 实现 Aggregation 方法
func (e multiMetricExporter) Aggregation() metric.Aggregation {
	// 返回一个合适的聚合类型
	return aggregation.Sum() // 使用 aggregation 包中的 Sum
}
