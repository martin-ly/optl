# Metrics 现代化迁移方案（Go 1.25 / 2025）

## 目标

- 使用 `sdk/metric` 新架构：`MeterProvider + reader.Periodic + view`。
- 明确 Temporality（Cumulative/Delta）与 Aggregation（Explicit/Exponential Histogram、Sum、LastValue）。

## 当前问题分析

- 现有代码使用 `controller/basic`、`processor/basic`、`selector/simple` 等已弃用接口。
- 缺少明确的 View 配置，无法精确控制聚合策略。
- 未充分利用新 SDK 的 Temporality 选择能力。

## 迁移步骤详解

### 1. 移除旧依赖

```go
// 移除这些导入
// "go.opentelemetry.io/otel/sdk/metric/controller/basic"
// "go.opentelemetry.io/otel/sdk/metric/processor/basic"
// "go.opentelemetry.io/otel/sdk/metric/selector/simple"
```

### 2. 新架构实现

```go
import (
    "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/sdk/metric/reader"
    "go.opentelemetry.io/otel/sdk/resource"
)

func SetupMetrics(cfg Config) (*MetricProvider, error) {
    // 创建资源
    res, err := createResource(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create resource: %w", err)
    }

    // 创建导出器
    exporter, cleanup, err := createMetricExporter(cfg)
    if err != nil {
        return nil, err
    }

    // 创建 Reader
    periodicReader := reader.NewPeriodic(
        exporter,
        reader.WithInterval(cfg.MetricCollectionInterval),
    )

    // 创建 MeterProvider 并配置 View
    mp := metric.NewMeterProvider(
        metric.WithResource(res),
        metric.WithReader(periodicReader),
        metric.WithView(metric.NewView(
            metric.Instrument{Name: "http_request_duration"},
            metric.Stream{
                Aggregation: metric.AggregationExplicitBucketHistogram{
                    Boundaries: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
                },
                Temporality: metric.DeltaTemporality,
            },
        )),
        metric.WithView(metric.NewView(
            metric.Instrument{Name: "runtime_*"},
            metric.Stream{
                Temporality: metric.CumulativeTemporality,
            },
        )),
    )

    // 设置全局 provider
    otel.SetMeterProvider(mp)

    return &MetricProvider{
        meterProvider: mp,
        cleanup:       cleanup,
    }, nil
}
```

### 3. 运行时指标更新

```go
import "go.opentelemetry.io/contrib/instrumentation/runtime"

// 使用最新 API
err := runtime.Start(
    runtime.WithMinimumReadMemStatsInterval(time.Second),
    runtime.WithMeterProvider(mp), // 传入自定义 provider
)
```

## 配置建议

### Temporality 选择策略

- **Cumulative**：适用于 Prometheus 兼容后端，累积计数器
- **Delta**：适用于流式处理，减少存储压力
- **建议**：服务指标用 Delta，系统指标用 Cumulative

### 直方图边界配置

```go
// 延迟分布边界（毫秒）
latencyBoundaries := []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000}

// 大小分布边界（字节）
sizeBoundaries := []float64{1024, 4096, 16384, 65536, 262144, 1048576, 4194304}

// 指数直方图（推荐用于动态范围大的指标）
exponentialHistogram := metric.AggregationBase2ExponentialHistogram{
    MaxSize: 160, // 最大桶数
}
```

### View 配置最佳实践

```go
views := []metric.View{
    // HTTP 请求延迟
    metric.NewView(
        metric.Instrument{Name: "http_request_duration_seconds"},
        metric.Stream{
            Aggregation: metric.AggregationExplicitBucketHistogram{
                Boundaries: latencyBoundaries,
            },
            Temporality: metric.DeltaTemporality,
        },
    ),
    // 业务计数器
    metric.NewView(
        metric.Instrument{Name: "business_*"},
        metric.Stream{
            Aggregation: metric.AggregationSum{},
            Temporality: metric.DeltaTemporality,
        },
    ),
    // 系统指标
    metric.NewView(
        metric.Instrument{Name: "runtime_*"},
        metric.Stream{
            Temporality: metric.CumulativeTemporality,
        },
    ),
}
```

## 兼容性策略

### 版本锁定

```go
// go.mod 中固定版本
require (
    go.opentelemetry.io/otel v1.30.0
    go.opentelemetry.io/otel/sdk v1.30.0
    go.opentelemetry.io/contrib/instrumentation/runtime v0.60.0
)
```

### 升级检查清单

- [ ] 验证所有 View 配置正确应用
- [ ] 确认 Temporality 与后端兼容
- [ ] 测试运行时指标正常收集
- [ ] 验证自定义指标聚合正确
- [ ] 检查导出器配置兼容性

## 验证方法

### 单元测试

```go
func TestMetricsSetup(t *testing.T) {
    cfg := Config{
        EnableMetrics: true,
        MetricCollectionInterval: 5 * time.Second,
    }
    
    provider, err := SetupMetrics(cfg)
    require.NoError(t, err)
    defer provider.Shutdown(context.Background())
    
    // 验证 MeterProvider 创建成功
    assert.NotNil(t, provider.meterProvider)
    
    // 验证全局 provider 设置
    assert.Equal(t, provider.meterProvider, otel.GetMeterProvider())
}
```

### 集成测试

```go
func TestMetricsExport(t *testing.T) {
    // 使用 testcontainers 启动 Collector
    // 验证指标正确导出和聚合
}
```

## 性能考虑

- 新架构减少了内存分配，提升性能
- View 配置在初始化时完成，运行时开销更低
- 建议在生产环境进行性能基准测试

## 迁移时间线

1. **第1周**：完成代码迁移，编写单元测试
2. **第2周**：集成测试，性能验证
3. **第3周**：生产环境灰度部署
4. **第4周**：全量部署，监控稳定性
