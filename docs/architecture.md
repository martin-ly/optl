# 架构与信号流说明

## 目标与范围

- 统一三信号（Traces/Metrics/Logs）到 Collector/后端，支持本地与生产两种模式。
- 正确的上下文传播：进程内、跨 goroutine、跨进程（HTTP/gRPC）。
- 生产可用的导出与资源语义约定，且可回放与复现实验。

## 组件视图

- 应用：`internal/services/*` 示例服务，调用 `internal/telemetry/*` 建立三信号。
- 遥测提供者：`Provider` 负责 Tracer/Meter/Logger 初始化、生命周期与 Shutdown。
- 导出层：stdout + OTLP（gRPC）。推荐在生产使用 OTLP→Collector→后端（Tempo/Jaeger、Prometheus/Mimir、Loki）。

## 信号流

1. 应用接入 `context.Context`，在入口（HTTP/gRPC）通过中间件抽取 W3C TraceContext 与 Baggage。
2. 生成/继续 Span；在关键路径记录事件、异常与属性；在并发 goroutine 通过上下文传播保持链路一致。
3. 指标在关键操作处记录（计数、直方图），必要时附带 Exemplars 与关联 TraceID。
4. 日志通过结构化记录保持 `trace_id`/`span_id` 字段一致，可经 Collector 汇聚。
5. 批处理与导出由 SDK/Collector 完成；失败时退避重试，确保背压安全。

## 入口与中间件建议（Go）

- HTTP：服务端与客户端分别使用 OTel 官方中间件（`net/http` RoundTripper 与 Handler 包装）。
- gRPC：拦截器（Unary/Stream）双向注入与提取 TraceContext/Baggage。
- 数据层/消息队列：优先采用 `contrib` 中的 instrumentation 以减少手工埋点。

## 上下文传播陷阱与对策

- goroutine 泄漏上下文：新建 goroutine 前显式捕获并传递 `ctx`，避免使用 `context.Background()`。
- 超时与取消：入口统一设置 `timeout/deadline`，在下游尊重取消，避免“僵尸 span”。
- 重试与幂等：重试应保留相同 `trace_id`，必要时在属性中标注 `retry_attempt`。

## 语义约定与资源

- 最小集：`service.name`、`service.version`、`deployment.environment`、`service.instance.id`。
- 扩展建议：主机/容器/K8s/Cloud Provider 属性由 Agent/Collector 注入，避免应用关心环境细节。
- 字段一致性：日志与 Span 属性使用统一命名，确保可关联查询。

## 开发与生产运行模式

- 开发：启用 stdout 导出、低采样门槛、较短批超时，Collector 简化流水线（本地后端）。
- 生产：关闭 insecure、启用 TLS/mTLS；启用重试与限流；Tail-based 采样；完善 PII 过滤。

## 样例调用链（建议图示）

- Client → API Gateway → Service A → Service B → DB/Cache。
- 每段都通过中间件传播 TraceContext，形成单一 `trace_id` 的因果链。
- [占位] 在 `docs/_images/trace-seq.png` 放置时序图与组件交互图。

## 可靠性要点

- 批处理参数（超时、最大批量）与队列容量需与峰值流量匹配。
- OTLP 连接应使用 TLS/mTLS；失败重试、退避与限流必须启用。
- 优雅停机：Metrics→Traces→Logs 顺序 drain，设置明确超时。

## 迁移与兼容

- Metrics 使用新版 `sdk/metric` 的 Reader/View；避免 legacy controller/processor。
- 固定 semconv 版本，逐步升级并记录破坏性变更与映射策略。

## 开发与验证路径

- 本地：stdout + Collector(dev)；压测：生成固定速率 Span/Metric，验证丢弃率≈0。
- 生产演示：docker-compose 或 K8s 清单，含 Tempo/Jaeger、Grafana、Loki。
