package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogProvider 封装日志 provider 和 cleanup 函数
type LogProvider struct {
	logger *zap.Logger
}

// SetupLogging 配置日志功能
func SetupLogging(cfg Config) (*LogProvider, error) {
	// 配置 zap 日志
	zapCfg := zap.NewProductionConfig()

	// 根据环境配置日志级别
	switch cfg.Environment {
	case "development":
		zapCfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		zapCfg.Development = true
		zapCfg.Encoding = "console"
	case "staging":
		zapCfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "production":
		zapCfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	// 添加默认字段
	zapCfg.InitialFields = map[string]interface{}{
		"service": cfg.ServiceName,
		"version": cfg.ServiceVersion,
		"env":     cfg.Environment,
	}

	// 创建日志记录器
	logger, err := zapCfg.Build(
		zap.AddCallerSkip(1),
		zap.WithCaller(true),
	)
	if err != nil {
		return nil, err
	}

	// 替换全局 logger
	zap.ReplaceGlobals(logger)

	return &LogProvider{
		logger: logger,
	}, nil
}

// Shutdown 关闭日志系统
func (lp *LogProvider) Shutdown() error {
	return lp.logger.Sync()
}

// Logger 获取日志记录器
func Logger() *zap.Logger {
	return zap.L()
}

// LoggerWithContext 从上下文中获取日志记录器，如果包含追踪信息则添加
func LoggerWithContext(ctx context.Context) *zap.Logger {
	logger := zap.L()

	// 如果上下文中包含 Span，则提取信息
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		sc := span.SpanContext()
		logger = logger.With(
			zap.String("trace_id", sc.TraceID().String()),
			zap.String("span_id", sc.SpanID().String()),
		)
	}

	return logger
}

// LoggerWithTraceContext 创建带有追踪上下文的日志记录器
func LoggerWithTraceContext(parent *zap.Logger, ctx context.Context) *zap.Logger {
	if parent == nil {
		parent = zap.L()
	}

	// 如果上下文中包含 Span，则提取信息
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		sc := span.SpanContext()
		return parent.With(
			zap.String("trace_id", sc.TraceID().String()),
			zap.String("span_id", sc.SpanID().String()),
		)
	}

	return parent
}

// AddSpanAttributes 为当前 span 添加属性
func AddSpanAttributes(ctx context.Context, fields ...zap.Field) {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}

	attributes := make([]attribute.KeyValue, 0, len(fields))
	for _, field := range fields {
		// 将 zap 字段转换为 OpenTelemetry 属性
		attr := zapFieldToAttribute(field)
		if attr.Key != "" {
			attributes = append(attributes, attr)
		}
	}

	if len(attributes) > 0 {
		span.SetAttributes(attributes...)
	}
}

// zapFieldToAttribute 将 zap 字段转换为 OpenTelemetry 属性
func zapFieldToAttribute(field zap.Field) attribute.KeyValue {
	key := field.Key

	switch field.Type {
	case zapcore.StringType:
		return attribute.String(key, field.String)
	case zapcore.BoolType:
		return attribute.Bool(key, field.Integer == 1)
	case zapcore.Int8Type, zapcore.Int16Type, zapcore.Int32Type, zapcore.Int64Type,
		zapcore.Uint8Type, zapcore.Uint16Type, zapcore.Uint32Type, zapcore.Uint64Type,
		zapcore.UintptrType:
		return attribute.Int64(key, field.Integer)
	case zapcore.Float32Type, zapcore.Float64Type:
		return attribute.Float64(key, float64(field.Integer))
	default:
		// 对于复杂类型，转为字符串
		return attribute.String(key, field.String)
	}
}
