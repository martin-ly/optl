# 与 OpenTelemetry 体系/生态的对标与差距

## 规范对齐

- Trace：BatchSpanProcessor、W3C TraceContext、Baggage、采样策略（Always/Never/Ratio）。
- Metrics：采用新 Reader/View/Temporality 与 Aggregation 配置；弃用 legacy controller/processor。
- Logs：建议通过 Collector 接入 OTLP Logs，统一三信号归集。

## 语义约定（SemConv）与版本治理

- 固定 `semconv` 版本（例如 v1.30.x），记录升级策略与不兼容变更。
- 对自定义属性使用私有命名空间前缀（如 `custom.*`），避免与标准冲突。

## 自动插桩清单（建议）

- HTTP：`net/http` 客户端/服务端；
- gRPC：客户端与服务端拦截器；
- 数据库：`database/sql` 包装器（常见驱动）；
- 消息系统：Kafka/NATS/Redis；
- 运行时：Go runtime/GC/协程指标；进程与主机指标（通过 Collector）。

## Collector 功能矩阵

- 处理：队列、重试、限流、尾采样、属性映射、资源检测器、PII 过滤。
- 导出：OTLP/Jaeger/Tempo/Prometheus/Loki/云厂商后端。

## 当前差距

- Metrics 仍基于旧接口；缺 HTTP/gRPC 插桩样例；OTLP 未启用 TLS；缺 Collector 流水线与后端一键化部署与仪表板；Logs 未经 Collector 汇聚。

## 版本锁定策略

- SDK/Contrib/Collector 三者分别锁定小版本；引入 Renovate/Dependabot 自动 PR，配合回归脚本。

## 行动建议（摘要）

1) 迁移 Metrics 至新 API；2) 加入 HTTP/gRPC 中间件；3) OTLP TLS/mTLS；4) Collector 栈与看板；5) Logs 汇聚与字段标准化；6) 建立版本与兼容性策略。
