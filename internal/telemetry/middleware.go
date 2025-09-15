package telemetry

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// HTTPMiddleware 提供 HTTP 服务端和客户端的自动插桩
type HTTPMiddleware struct {
	tracer trace.Tracer
}

// NewHTTPMiddleware 创建 HTTP 中间件
func NewHTTPMiddleware(serviceName string) *HTTPMiddleware {
	return &HTTPMiddleware{
		tracer: otel.Tracer(serviceName),
	}
}

// Handler 返回 HTTP 服务端中间件
func (h *HTTPMiddleware) Handler(next http.Handler) http.Handler {
	return otelhttp.NewHandler(next, "http-server",
		otelhttp.WithTracerProvider(otel.GetTracerProvider()),
		otelhttp.WithPropagators(otel.GetTextMapPropagator()),
	)
}

// HandlerWithName 返回指定名称的 HTTP 服务端中间件
func (h *HTTPMiddleware) HandlerWithName(operationName string, next http.Handler) http.Handler {
	return otelhttp.NewHandler(next, operationName,
		otelhttp.WithTracerProvider(otel.GetTracerProvider()),
		otelhttp.WithPropagators(otel.GetTextMapPropagator()),
	)
}

// Client 返回配置了追踪的 HTTP 客户端
func (h *HTTPMiddleware) Client() *http.Client {
	return &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport,
			otelhttp.WithTracerProvider(otel.GetTracerProvider()),
			otelhttp.WithPropagators(otel.GetTextMapPropagator()),
		),
		Timeout: 30 * time.Second,
	}
}

// ClientWithTransport 返回使用指定 Transport 的追踪客户端
func (h *HTTPMiddleware) ClientWithTransport(transport http.RoundTripper) *http.Client {
	return &http.Client{
		Transport: otelhttp.NewTransport(transport,
			otelhttp.WithTracerProvider(otel.GetTracerProvider()),
			otelhttp.WithPropagators(otel.GetTextMapPropagator()),
		),
		Timeout: 30 * time.Second,
	}
}

// WrapHandler 包装 HTTP 处理器，添加自定义属性
func (h *HTTPMiddleware) WrapHandler(operationName string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := h.tracer.Start(r.Context(), operationName)
		defer span.End()

		// 添加请求属性
		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.user_agent", r.UserAgent()),
			attribute.String("http.scheme", r.URL.Scheme),
			attribute.String("http.host", r.Host),
		)

		// 创建响应写入器来捕获状态码
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		// 执行处理器
		handler(wrapped, r.WithContext(ctx))

		// 设置响应属性
		span.SetAttributes(attribute.Int("http.status_code", wrapped.statusCode))
		
		// 设置状态码
		if wrapped.statusCode >= 400 {
			span.SetStatus(codes.Error, http.StatusText(wrapped.statusCode))
		}
	}
}

// responseWriter 包装 http.ResponseWriter 以捕获状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// PropagateContext 在 HTTP 请求中传播追踪上下文
func (h *HTTPMiddleware) PropagateContext(ctx context.Context, req *http.Request) *http.Request {
	// 使用全局传播器注入上下文
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
	return req
}

// ExtractContext 从 HTTP 请求中提取追踪上下文
func (h *HTTPMiddleware) ExtractContext(req *http.Request) context.Context {
	return otel.GetTextMapPropagator().Extract(req.Context(), propagation.HeaderCarrier(req.Header))
}
