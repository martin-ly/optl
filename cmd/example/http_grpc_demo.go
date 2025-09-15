package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "time"

    "optl/internal/telemetry"

    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
    "go.uber.org/zap"
)

// runHTTPDemo 启动一个带 OTel 自动插桩的 HTTP 服务，并发起客户端请求演示端到端链路
func runHTTPDemo() {
    // 信号上下文
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
    defer cancel()

    // 初始化遥测
    provider, err := initTelemetry()
    if err != nil {
        fmt.Printf("Failed to initialize telemetry: %v\n", err)
        os.Exit(1)
    }
    defer provider.Shutdown(context.Background())

    logger := telemetry.Logger()
    logger.Info("Starting HTTP demo", zap.String("service", serviceName))

    // HTTP 中间件
    httpmw := telemetry.NewHTTPMiddleware(serviceName)

    // 业务 Handler（包装并添加属性）
    helloHandler := httpmw.WrapHandler("hello_handler", func(w http.ResponseWriter, r *http.Request) {
        // 在当前 span 添加事件与属性
        telemetry.AddSpanEvent(r.Context(), "handle_request",
            attribute.String("path", r.URL.Path),
        )
        // 模拟内部子步骤
        _ = telemetry.WithSpan(r.Context(), "inner_step", func(ctx context.Context) error {
            time.Sleep(50 * time.Millisecond)
            telemetry.AddSpanAttributes(ctx,
                zap.String("step", "inner"),
                zap.Int("work", 1),
            )
            return nil
        })
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("hello, otel http"))
    })

    mux := http.NewServeMux()
    mux.Handle("/hello", helloHandler)

    // 包装服务端 Handler 以自动插桩
    srv := &http.Server{
        Addr:    ":8080",
        Handler: httpmw.Handler(mux),
    }

    // 启动服务器
    go func() {
        logger.Info("HTTP server listening", zap.String("addr", srv.Addr))
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("HTTP server failed", zap.Error(err))
        }
    }()

    // 客户端带自动插桩
    client := httpmw.Client()

    // 发起一条端到端请求
    func() {
        // 构造根 span，模拟上游入口
        rootCtx, span := telemetry.ContextWithSpan(ctx, "client_request",
            trace.WithAttributes(attribute.String("demo", "http")))
        defer span.End()

        req, _ := http.NewRequestWithContext(rootCtx, http.MethodGet, "http://127.0.0.1:8080/hello", nil)
        // 显式注入上下文（可选，Client Transport 已注入）
        req = httpmw.PropagateContext(rootCtx, req)

        resp, err := client.Do(req)
        if err != nil {
            logger.Error("client request failed", zap.Error(err))
            return
        }
        _ = resp.Body.Close()
        logger.Info("client request done", zap.Int("status", resp.StatusCode))
    }()

    // 等待退出信号
    <-ctx.Done()
    _ = srv.Shutdown(context.Background())
    logger.Info("HTTP demo stopped")
}


