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
    "go.opentelemetry.io/otel/sdk/metric/reader"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
    "google.golang.org/grpc/credentials/insecure"
)

// MetricProvider 封装 metric provider 和 cleanup 函数（新 API）
type MetricProvider struct {
    meterProvider *metric.MeterProvider
    cleanup       func() error
}

// SetupMetrics 配置指标监控功能（基于新 reader/view 架构）
func SetupMetrics(cfg Config) (*MetricProvider, error) {
    if !cfg.EnableMetrics {
        return nil, nil
    }

    // 创建资源属性
    res, err := createResource(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create resource: %w", err)
    }

    // 构造 readers（每个导出器一个 reader）与清理函数链
    var (
        readers []metric.Reader
        cleanup func() error
    )

    // 控制台导出器
    if cfg.EnableConsoleExporter {
        consoleExporter, err := stdoutmetric.New(
            stdoutmetric.WithPrettyPrint(),
        )
        if err != nil {
            return nil, fmt.Errorf("failed to create stdout metric exporter: %w", err)
        }
        readers = append(readers, reader.NewPeriodic(
            consoleExporter,
            reader.WithInterval(cfg.MetricCollectionInterval),
        ))
        prev := cleanup
        cleanup = func() error {
            if prev != nil {
                if err := prev(); err != nil {
                    return err
                }
            }
            return consoleExporter.Shutdown(context.Background())
        }
    }

    // OTLP 导出器
    if cfg.OTLPEndpoint != "" {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        // 配置 gRPC 连接选项
        var grpcOpts []grpc.DialOption
        
        // 配置 TLS 凭据
        if cfg.TLSConfig.Enabled {
            tlsConfig, err := createTLSConfig(cfg.TLSConfig)
            if err != nil {
                return nil, fmt.Errorf("failed to create TLS config: %w", err)
            }
            grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
        } else {
            grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
        }
        
        grpcOpts = append(grpcOpts, grpc.WithBlock())

        conn, err := grpc.DialContext(ctx, cfg.OTLPEndpoint, grpcOpts...)
        if err != nil {
            return nil, fmt.Errorf("failed to connect to OTLP endpoint: %w", err)
        }

        // 配置 OTLP 客户端选项
        var clientOpts []otlpmetricgrpc.Option
        clientOpts = append(clientOpts, otlpmetricgrpc.WithGRPCConn(conn))
        
        // 配置重试选项
        if cfg.RetryConfig.Enabled {
            clientOpts = append(clientOpts, otlpmetricgrpc.WithRetry(otlpmetricgrpc.RetryConfig{
                Enabled:         true,
                InitialInterval: cfg.RetryConfig.InitialInterval,
                MaxInterval:     cfg.RetryConfig.MaxInterval,
                MaxElapsedTime:  cfg.RetryConfig.MaxElapsedTime,
                Multiplier:      cfg.RetryConfig.Multiplier,
                RandomizationFactor: cfg.RetryConfig.RandomizationFactor,
            }))
        }

        otlpExporter, err := otlpmetricgrpc.New(
            context.Background(),
            clientOpts...,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
        }
        readers = append(readers, reader.NewPeriodic(
            otlpExporter,
            reader.WithInterval(cfg.MetricCollectionInterval),
        ))
        prev := cleanup
        cleanup = func() error {
            if prev != nil {
                if err := prev(); err != nil {
                    return err
                }
            }
            return otlpExporter.Shutdown(context.Background())
        }
    }

    if len(readers) == 0 {
        // 未启用任何导出器时，不创建 provider
        return &MetricProvider{meterProvider: nil, cleanup: nil}, nil
    }

    // 创建 MeterProvider 并挂载 readers
    opts := []metric.Option{metric.WithResource(res)}
    for _, r := range readers {
        opts = append(opts, metric.WithReader(r))
    }
    mp := metric.NewMeterProvider(opts...)

    // 设置全局 provider
    otel.SetMeterProvider(mp)

    // 启用 runtime 指标
    if err := runtime.Start(
        runtime.WithMinimumReadMemStatsInterval(time.Second),
        runtime.WithMeterProvider(mp),
    ); err != nil {
        return nil, fmt.Errorf("failed to start runtime metrics: %w", err)
    }

    return &MetricProvider{
        meterProvider: mp,
        cleanup:       cleanup,
    }, nil
}

// Shutdown 关闭 metric provider
func (mp *MetricProvider) Shutdown(ctx context.Context) error {
    if mp.meterProvider != nil {
        if err := mp.meterProvider.Shutdown(ctx); err != nil {
            return err
        }
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


