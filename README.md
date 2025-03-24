# OpenTelemetry 分布式控制流日志示例

这个项目展示了使用 Golang 和 OpenTelemetry 实现的分布式控制流日志系统，包含以下特性：

## 功能特性

- **日志（Logs）**：使用结构化日志记录，与 traces 和 metrics 关联
- **链路追踪（Traces）**：分布式事务追踪，包括跨服务调用的上下文传播
- **指标监控（Metrics）**：记录应用和业务指标，支持计数器、测量器和直方图
- **控制流组合**：展示如何在复杂控制流中正确传播上下文
- **Goroutine 组合**：在并发 goroutine 中保持追踪上下文
- **资源和属性**：为遥测数据添加丰富的元数据
- **导出器**：支持多种导出格式，包括 OTLP、标准输出等
- **采样**：实现各种采样策略
- **批处理和缓冲**：优化遥测数据发送

## 项目结构

```text
/
├── cmd/                    # 命令行应用
│   └── example/            # 主要示例应用
├── internal/               # 内部包
│   ├── telemetry/          # 遥测相关功能
│   │   ├── config.go       # 配置管理
│   │   ├── trace.go        # 追踪功能
│   │   ├── metric.go       # 指标监控
│   │   ├── log.go          # 日志记录
│   │   ├── context.go      # 上下文传播
│   │   └── provider.go     # 整合提供者
│   └── services/           # 模拟的微服务
│       ├── processor.go    # 数据处理服务
│       ├── analyzer.go     # 数据分析服务
│       └── storage.go      # 数据存储服务
└── go.mod                  # Go 模块定义
```

## 示例说明

该项目提供两种示例：

1. **基本示例**：展示基本的日志、追踪和指标收集，以及在并发 goroutine 中的上下文传播。
2. **服务示例**：展示在多个微服务之间的遥测数据传播，包括跨服务调用、上下文传递等。

## 使用方法

### 前提条件

1. 确保已安装 Go 1.21 或更高版本
2. 克隆此仓库
3. 下载依赖：`go mod tidy`

### 运行基本示例

```bash
go run ./cmd/example
```

### 运行服务示例

```bash
go run ./cmd/example services
```

## 集成 OpenTelemetry Collector

如果需要将遥测数据发送到 OpenTelemetry Collector，可以通过环境变量配置：

```bash
# 配置 OTLP 导出端点
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317

# 运行示例
go run ./cmd/example
```

## 配置选项

可以通过环境变量配置遥测系统的行为：

- `OTEL_SERVICE_NAME`: 服务名称（默认: "optl-service"）
- `OTEL_SERVICE_VERSION`: 服务版本（默认: "v0.1.0"）
- `OTEL_ENVIRONMENT`: 环境类型，如 development, staging, production（默认: "development"）
- `OTEL_RESOURCE_ATTRIBUTES`: 资源属性，格式为 "key1=value1,key2=value2"
- `OTEL_EXPORTER_OTLP_ENDPOINT`: OTLP 导出器端点（默认: "localhost:4317"）
- `OTEL_ENABLE_CONSOLE_EXPORTER`: 是否启用控制台导出（默认: true）
- `OTEL_BATCH_TIMEOUT`: 批处理超时时间（默认: 5s）
- `OTEL_MAX_EXPORT_BATCH_SIZE`: 批处理的最大导出大小（默认: 512）
- `OTEL_SAMPLING_RATIO`: 采样率，0-1（默认: 1.0，全采样）
- `OTEL_ENABLE_METRICS`: 是否启用指标收集（默认: true）
- `OTEL_ENABLE_LOGS`: 是否启用日志收集（默认: true）
- `OTEL_METRIC_COLLECTION_INTERVAL`: 指标收集间隔（默认: 10s）

## 关键功能展示

### 1. 跨服务追踪

服务示例展示了如何在多个服务之间传递追踪上下文，形成完整的请求链路。

### 2. 上下文传播

这个项目展示了如何在不同的控制流（顺序、条件、循环）和并发 goroutine 中传播上下文。

### 3. 自定义属性和事件

示例展示了如何向 span 添加自定义属性和事件，丰富追踪信息。

### 4. 异常跟踪

演示如何跟踪错误和异常，包括捕获、记录和上报。

### 5. 自定义指标

展示如何创建和记录自定义指标，包括计数器、测量器和直方图。

## 相关资源

- [OpenTelemetry Go 文档](https://opentelemetry.io/docs/instrumentation/go/)
- [OpenTelemetry 规范](https://github.com/open-telemetry/opentelemetry-specification)
