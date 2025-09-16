package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Provider 整合所有遥测功能的提供者
type Provider struct {
	config         Config
	traceProvider  *TraceProvider
	metricProvider *MetricProvider
	logProvider    *LogProvider
	startTime      time.Time
	shutdownErrors metric.Int64Counter
	providerUp     metric.Int64ObservableGauge
}

// NewProvider 创建一个新的遥测功能提供者
func NewProvider(cfg Config) (*Provider, error) {
	provider := &Provider{
		config: cfg,
	}

	// 初始化日志
	logProvider, err := SetupLogging(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to setup logging: %w", err)
	}
	provider.logProvider = logProvider

	// 初始化 trace
	traceProvider, err := SetupTracing(cfg)
	if err != nil {
		logProvider.Shutdown()
		return nil, fmt.Errorf("failed to setup tracing: %w", err)
	}
	provider.traceProvider = traceProvider

	// 初始化 metrics
	if cfg.EnableMetrics {
		metricProvider, err := SetupMetrics(cfg)
		if err != nil {
			logProvider.Shutdown()
			traceProvider.Shutdown(context.Background())
			return nil, fmt.Errorf("failed to setup metrics: %w", err)
		}
		provider.metricProvider = metricProvider
	}

	provider.initHealthMetrics()

	return provider, nil
}

// Shutdown 关闭所有遥测功能
func (p *Provider) Shutdown(ctx context.Context) error {
	var errs []error

	// 关闭 metrics
	if p.metricProvider != nil {
		if err := p.metricProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown metrics: %w", err))
		}
	}

	// 关闭 trace
	if p.traceProvider != nil {
		if err := p.traceProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown tracing: %w", err))
		}
	}

	// 关闭日志
	if p.logProvider != nil {
		if err := p.logProvider.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown logging: %w", err))
		}
	}

	if len(errs) > 0 {
		if p.shutdownErrors != nil {
			p.shutdownErrors.Add(ctx, int64(len(errs)))
		}
		return fmt.Errorf("errors during shutdown: %v", errs)
	}
	return nil
}

// 提供对配置的访问
func (p *Provider) Config() Config {
	return p.config
}

// initHealthMetrics 暴露 Provider 自观测指标
func (p *Provider) initHealthMetrics() {
	p.startTime = time.Now()
	meter := otel.Meter("telemetry.provider")

	up, err := meter.Int64ObservableGauge("telemetry_provider_up",
		metric.WithDescription("Telemetry provider up gauge (1=up)"),
		metric.WithUnit("{state}"),
	)
	if err == nil {
		p.providerUp = up
		_, _ = meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
			o.ObserveInt64(up, 1, metric.WithAttributes(
				attribute.String("service.name", p.config.ServiceName),
				attribute.String("service.version", p.config.ServiceVersion),
				attribute.String("environment", p.config.Environment),
			))
			return nil
		}, up)
	}

	se, err := meter.Int64Counter("telemetry_shutdown_errors",
		metric.WithDescription("Number of errors occurred during provider shutdown"),
	)
	if err == nil {
		p.shutdownErrors = se
	}

	_, _ = meter.Float64ObservableGauge("telemetry_provider_uptime_seconds",
		metric.WithDescription("Telemetry provider uptime in seconds"),
		metric.WithUnit("s"),
		metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
			o.Observe(time.Since(p.startTime).Seconds(), metric.WithAttributes(
				attribute.String("service.name", p.config.ServiceName),
			))
			return nil
		}),
	)
}
