# 课程化材料

## 课程大纲

### 第一部分：理论基础（4 周）

**第 1 周：OpenTelemetry 概述与三信号**

- 可观测性三大支柱：Traces、Metrics、Logs
- OpenTelemetry 数据模型：Resource、Scope、Instrument
- 语义约定（SemConv）与标准化
- 实验：搭建基础遥测环境

**第 2 周：分布式追踪与上下文传播**

- W3C TraceContext 规范
- Baggage 传播机制
- 采样策略：Head-based vs Tail-based
- 实验：实现跨服务追踪链路

**第 3 周：指标系统与聚合**

- 指标类型：Counter、Gauge、Histogram
- 聚合策略：Sum、LastValue、Explicit/Exponential Histogram
- Temporality：Cumulative vs Delta
- 实验：配置自定义指标与 View

**第 4 周：日志系统与关联**

- 结构化日志与 OTel Logs
- 日志-Trace 关联机制
- 采样与过滤策略
- 实验：实现日志与追踪的关联

### 第二部分：工程实践（4 周）

**第 5 周：Collector 流水线设计**

- 接收器、处理器、导出器配置
- 批处理与队列管理
- 错误处理与重试机制
- 实验：设计多环境 Collector 配置

**第 6 周：安全与合规**

- TLS/mTLS 配置
- PII 过滤与数据脱敏
- 访问控制与审计
- 实验：实现安全遥测管道

**第 7 周：性能优化与监控**

- 观测开销评估
- 背压处理与限流
- 自观测与健康检查
- 实验：性能基准测试

**第 8 周：生产部署与运维**

- 容器化与 Kubernetes 部署
- 监控告警与 SLO
- 故障排查与调试
- 实验：端到端生产环境搭建

## 作业设计

### 作业 1：自定义采样器实现（第 2 周）

**目标**：实现一个基于业务逻辑的自定义采样器

**要求**：

```go
// 实现接口
type BusinessSampler struct {
    // 根据用户等级、错误率、延迟等条件采样
}

func (s *BusinessSampler) ShouldSample(parameters SamplingParameters) SamplingResult {
    // 实现采样逻辑
}
```

**评估标准**：

- 采样逻辑正确性（40%）
- 性能影响评估（30%）
- 测试覆盖率（30%）

### 作业 2：指标聚合配置（第 3 周）

**目标**：配置不同场景下的指标聚合策略

**要求**：

- 为 HTTP 请求延迟配置指数直方图
- 为业务计数器配置 Delta Temporality
- 为系统指标配置 Cumulative Temporality
- 验证聚合误差在可接受范围内

**评估标准**：

- 配置正确性（50%）
- 误差分析报告（30%）
- 性能对比（20%）

### 作业 3：端到端链路实现（第 5 周）

**目标**：为新的中间件实现自动插桩

**要求**：

- 选择一种中间件（Redis、Kafka、gRPC 等）
- 实现客户端和服务端插桩
- 确保上下文正确传播
- 提供完整的测试用例

**评估标准**：

- 插桩完整性（40%）
- 上下文传播正确性（30%）
- 测试质量（30%）

### 作业 4：生产环境设计（第 8 周）

**目标**：设计完整的生产级遥测系统

**要求**：

- 设计多环境配置（dev/staging/prod）
- 实现安全与合规要求
- 配置监控告警
- 提供故障排查手册

**评估标准**：

- 架构设计合理性（40%）
- 安全合规性（30%）
- 运维友好性（30%）

## 评分 Rubric

### 正确性（40%）

- **优秀（36-40 分）**：代码完全正确，通过所有测试，无逻辑错误
- **良好（32-35 分）**：代码基本正确，通过大部分测试，有轻微问题
- **及格（28-31 分）**：代码基本可用，通过主要测试，有明显问题
- **不及格（<28 分）**：代码有严重错误，无法通过基本测试

### 工程化与运维（30%）

- **优秀（27-30 分）**：代码结构清晰，文档完整，易于维护
- **良好（24-26 分）**：代码结构合理，文档基本完整
- **及格（21-23 分）**：代码结构一般，文档不够完整
- **不及格（<21 分）**：代码结构混乱，缺乏文档

### 分析与报告（30%）

- **优秀（27-30 分）**：分析深入，报告专业，有独到见解
- **良好（24-26 分）**：分析合理，报告清晰，有一定见解
- **及格（21-23 分）**：分析基本正确，报告基本清晰
- **不及格（<21 分）**：分析有误，报告不清晰

## 实验环境

### 开发环境

```yaml
# 本地开发栈
services:
  - OpenTelemetry Collector (dev 配置)
  - Tempo (分布式追踪)
  - Prometheus (指标存储)
  - Loki (日志存储)
  - Grafana (可视化)
```

### 实验脚本

```bash
# 一键启动实验环境
./scripts/setup-lab.sh

# 运行实验
./scripts/run-experiment.sh sampling-bias
./scripts/run-experiment.sh backpressure
./scripts/run-experiment.sh overhead
```

## 课程资源

### 参考书籍

- 《Distributed Systems Observability》- Cindy Sridharan
- 《Observability Engineering》- Charity Majors
- 《Site Reliability Engineering》- Google

### 在线资源

- [OpenTelemetry 官方文档](https://opentelemetry.io/docs/)
- [CNCF 可观测性白皮书](https://github.com/cncf/tag-observability)
- [Prometheus 最佳实践](https://prometheus.io/docs/practices/)

### 工具链

- **开发**：Go、Docker、Kubernetes
- **监控**：Prometheus、Grafana、Jaeger
- **测试**：testcontainers、k6、Artillery
- **分析**：pprof、trace、flamegraph

## 考核方式

### 平时成绩（40%）

- 实验报告（20%）
- 课堂参与（10%）
- 作业提交（10%）

### 期末考试（60%）

- 理论考试（30%）：概念理解、原理分析
- 实践考试（30%）：现场编程、问题排查

### 加分项目

- 开源贡献（5%）
- 技术分享（5%）
- 创新实验（5%）

## 学习成果

### 知识目标

- 掌握 OpenTelemetry 核心概念与架构
- 理解分布式追踪、指标、日志的设计原理
- 熟悉生产级遥测系统的部署与运维

### 技能目标

- 能够设计和实现完整的遥测系统
- 具备性能优化和故障排查能力
- 掌握安全与合规最佳实践

### 素养目标

- 培养系统性思维和工程化意识
- 提升问题分析和解决能力
- 增强团队协作和沟通能力
