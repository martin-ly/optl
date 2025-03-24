package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// ContextWithSpan 创建带有 span 的上下文
func ContextWithSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Tracer("").Start(ctx, name, opts...)
}

// WithSpan 包装函数，创建一个新的 span
func WithSpan(ctx context.Context, name string, fn func(context.Context) error, opts ...trace.SpanStartOption) error {
	ctx, span := ContextWithSpan(ctx, name, opts...)
	defer span.End()

	// 从上下文中获取带有 trace ID 的日志记录器
	logger := LoggerWithContext(ctx)
	logger.Debug("Starting span", zap.String("span_name", name))

	// 执行函数
	err := fn(ctx)

	// 记录错误
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error("Span error",
			zap.String("span_name", name),
			zap.Error(err),
		)
	} else {
		logger.Debug("Completed span", zap.String("span_name", name))
	}

	return err
}

// SpanFromContext 从上下文中获取当前的 span
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// SpanContext 从上下文中获取 span 上下文
func SpanContext(ctx context.Context) trace.SpanContext {
	return trace.SpanFromContext(ctx).SpanContext()
}

// SpanContextWithTrace 检查上下文中是否有有效的 trace
func SpanContextWithTrace(ctx context.Context) bool {
	return trace.SpanFromContext(ctx).SpanContext().IsValid()
}

// SpanContextWithSampled 检查上下文中的 trace 是否被采样
func SpanContextWithSampled(ctx context.Context) bool {
	return trace.SpanFromContext(ctx).SpanContext().IsSampled()
}

// AddSpanEvent 向 span 添加事件
func AddSpanEvent(ctx context.Context, name string, attributes ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attributes...))
	}
}

// AddSpanEventWithTimestamp 向 span 添加带时间戳的事件
func AddSpanEventWithTimestamp(ctx context.Context, name string, timestamp time.Time, attributes ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attributes...), trace.WithTimestamp(timestamp))
	}
}

// SetSpanAttributes 设置 span 的属性
func SetSpanAttributes(ctx context.Context, attributes ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attributes...)
	}
}

// GoWithContext 在 goroutine 中执行函数并传递上下文
func GoWithContext(ctx context.Context, fn func(context.Context) error) error {
	// 创建 errgroup
	g, gCtx := errgroup.WithContext(ctx)

	// 启动 goroutine
	g.Go(func() error {
		return fn(gCtx)
	})

	// 等待 goroutine 完成
	return g.Wait()
}

// GoWithSpan 在带有 span 的 goroutine 中执行函数
func GoWithSpan(ctx context.Context, name string, fn func(context.Context) error, opts ...trace.SpanStartOption) error {
	return GoWithContext(ctx, func(gCtx context.Context) error {
		return WithSpan(gCtx, name, fn, opts...)
	})
}

// GoForEach 并行执行函数，并传递上下文
func GoForEach[T any](ctx context.Context, items []T, fn func(context.Context, T) error) error {
	g, gCtx := errgroup.WithContext(ctx)

	for _, item := range items {
		item := item // 创建闭包变量副本
		g.Go(func() error {
			return fn(gCtx, item)
		})
	}

	return g.Wait()
}

// GoForEachWithSpan 在带有 span 的 goroutine 中并行执行函数
func GoForEachWithSpan[T any](ctx context.Context, name string, items []T, fn func(context.Context, T) error) error {
	g, gCtx := errgroup.WithContext(ctx)

	for i, item := range items {
		i, item := i, item // 创建闭包变量副本
		g.Go(func() error {
			spanName := fmt.Sprintf("%s-%d", name, i)
			return WithSpan(gCtx, spanName, func(spanCtx context.Context) error {
				return fn(spanCtx, item)
			})
		})
	}

	return g.Wait()
}

// GoWithLimit 限制并行数量并传递上下文
func GoWithLimit[T any](ctx context.Context, concurrency int, items []T, fn func(context.Context, T) error) error {
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	for _, item := range items {
		item := item // 创建闭包变量副本
		g.Go(func() error {
			return fn(gCtx, item)
		})
	}

	return g.Wait()
}

// GoWithLimitAndSpan 在带有 span 的 goroutine 中限制并行数量
func GoWithLimitAndSpan[T any](ctx context.Context, name string, concurrency int, items []T, fn func(context.Context, T) error) error {
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	for i, item := range items {
		i, item := i, item // 创建闭包变量副本
		g.Go(func() error {
			spanName := fmt.Sprintf("%s-%d", name, i)
			return WithSpan(gCtx, spanName, func(spanCtx context.Context) error {
				return fn(spanCtx, item)
			})
		})
	}

	return g.Wait()
}
