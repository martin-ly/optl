package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GRPCMiddleware 提供 gRPC 服务端和客户端的自动插桩
type GRPCMiddleware struct {
	tracer trace.Tracer
}

// NewGRPCMiddleware 创建 gRPC 中间件
func NewGRPCMiddleware(serviceName string) *GRPCMiddleware {
	return &GRPCMiddleware{
		tracer: otel.Tracer(serviceName),
	}
}

// UnaryServerInterceptor 返回 gRPC 服务端一元调用拦截器
func (g *GRPCMiddleware) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return otelgrpc.UnaryServerInterceptor(
		otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
		otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
	)
}

// StreamServerInterceptor 返回 gRPC 服务端流式调用拦截器
func (g *GRPCMiddleware) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return otelgrpc.StreamServerInterceptor(
		otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
		otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
	)
}

// UnaryClientInterceptor 返回 gRPC 客户端一元调用拦截器
func (g *GRPCMiddleware) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return otelgrpc.UnaryClientInterceptor(
		otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
		otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
	)
}

// StreamClientInterceptor 返回 gRPC 客户端流式调用拦截器
func (g *GRPCMiddleware) StreamClientInterceptor() grpc.StreamClientInterceptor {
	return otelgrpc.StreamClientInterceptor(
		otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
		otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
	)
}

// DialOption 返回配置了追踪的 gRPC 客户端连接选项
func (g *GRPCMiddleware) DialOption() grpc.DialOption {
	return grpc.WithUnaryInterceptor(g.UnaryClientInterceptor())
}

// ServerOptions 返回配置了追踪的 gRPC 服务端选项
func (g *GRPCMiddleware) ServerOptions() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.UnaryInterceptor(g.UnaryServerInterceptor()),
		grpc.StreamInterceptor(g.StreamServerInterceptor()),
	}
}

// WrapUnaryHandler 包装一元 gRPC 处理器，添加自定义属性
func (g *GRPCMiddleware) WrapUnaryHandler(operationName string, handler grpc.UnaryHandler) grpc.UnaryHandler {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		ctx, span := g.tracer.Start(ctx, operationName)
		defer span.End()

		// 添加请求属性
		span.SetAttributes(
			attribute.String("rpc.system", "grpc"),
			attribute.String("rpc.service", operationName),
		)

		// 从元数据中提取信息
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if userAgent := md.Get("user-agent"); len(userAgent) > 0 {
				span.SetAttributes(attribute.String("rpc.user_agent", userAgent[0]))
			}
		}

		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		// 设置响应属性
		span.SetAttributes(attribute.Int64("rpc.duration_ms", duration.Milliseconds()))

		if err != nil {
			// 设置错误状态
			if st, ok := status.FromError(err); ok {
				span.SetAttributes(
					attribute.String("rpc.grpc.status_code", st.Code().String()),
					attribute.Int("rpc.grpc.status_code_int", int(st.Code())),
				)
				span.SetStatus(codes.Error, st.Message())
			} else {
				span.SetStatus(codes.Error, err.Error())
			}
		} else {
			span.SetAttributes(attribute.String("rpc.grpc.status_code", "OK"))
			span.SetStatus(codes.Ok, "")
		}

		return resp, err
	}
}

// WrapStreamHandler 包装流式 gRPC 处理器
func (g *GRPCMiddleware) WrapStreamHandler(operationName string, handler grpc.StreamHandler) grpc.StreamHandler {
	return func(srv interface{}, stream grpc.ServerStream) error {
		ctx, span := g.tracer.Start(stream.Context(), operationName)
		defer span.End()

		// 添加请求属性
		span.SetAttributes(
			attribute.String("rpc.system", "grpc"),
			attribute.String("rpc.service", operationName),
			attribute.String("rpc.method", "stream"),
		)

		// 从元数据中提取信息
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if userAgent := md.Get("user-agent"); len(userAgent) > 0 {
				span.SetAttributes(attribute.String("rpc.user_agent", userAgent[0]))
			}
		}

		start := time.Now()
		err := handler(srv, stream)
		duration := time.Since(start)

		// 设置响应属性
		span.SetAttributes(attribute.Int64("rpc.duration_ms", duration.Milliseconds()))

		if err != nil {
			if st, ok := status.FromError(err); ok {
				span.SetAttributes(
					attribute.String("rpc.grpc.status_code", st.Code().String()),
					attribute.Int("rpc.grpc.status_code_int", int(st.Code())),
				)
				span.SetStatus(codes.Error, st.Message())
			} else {
				span.SetStatus(codes.Error, err.Error())
			}
		} else {
			span.SetAttributes(attribute.String("rpc.grpc.status_code", "OK"))
			span.SetStatus(codes.Ok, "")
		}

		return err
	}
}

// PropagateContext 在 gRPC 调用中传播追踪上下文
func (g *GRPCMiddleware) PropagateContext(ctx context.Context) context.Context {
	// 创建元数据并注入上下文
	md := metadata.New(nil)
	otel.GetTextMapPropagator().Inject(ctx, &metadataCarrier{md})
	return metadata.NewOutgoingContext(ctx, md)
}

// ExtractContext 从 gRPC 上下文提取追踪上下文
func (g *GRPCMiddleware) ExtractContext(ctx context.Context) context.Context {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		return otel.GetTextMapPropagator().Extract(ctx, &metadataCarrier{md})
	}
	return ctx
}

// metadataCarrier 实现 propagation.TextMapCarrier 接口
type metadataCarrier struct {
	metadata.MD
}

func (mc *metadataCarrier) Get(key string) string {
	values := mc.MD.Get(key)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

func (mc *metadataCarrier) Set(key, value string) {
	mc.MD.Set(key, value)
}

func (mc *metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(mc.MD))
	for k := range mc.MD {
		keys = append(keys, k)
	}
	return keys
}
