package telemetry

import (
	"context"
	"fmt"
)

// Provider 整合所有遥测功能的提供者
type Provider struct {
	config         Config
	traceProvider  *TraceProvider
	metricProvider *MetricProvider
	logProvider    *LogProvider
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
		return fmt.Errorf("errors during shutdown: %v", errs)
	}
	return nil
}

// 提供对配置的访问
func (p *Provider) Config() Config {
	return p.config
}
