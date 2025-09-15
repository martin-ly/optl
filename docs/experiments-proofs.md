# 实验设计与形式化证明

## 采样无偏性（Head-based Ratio）

### 理论证明

**定理**：在 TraceID-Ratio-Based 采样下，计数性指标的无偏估计为 `count_observed / p`，其中 p 为采样率。

**证明**：
设真实计数为 N，采样率为 p ∈ (0,1]，观测计数为 X。

由于每个 trace 以概率 p 被采样，X 服从二项分布：

```
X ~ Binomial(N, p)
```

期望：

```
E[X] = N · p
```

因此，无偏估计为：

```
N̂ = X / p
E[N̂] = E[X] / p = (N · p) / p = N
```

**方差**：

```
Var(N̂) = Var(X) / p² = (N · p · (1-p)) / p² = N · (1-p) / p
```

### 实验验证脚本

```go
// experiments/sampling_bias_test.go
package experiments

import (
    "context"
    "fmt"
    "math"
    "testing"
    "time"
)

func TestSamplingBias(t *testing.T) {
    const (
        trueCount = 10000
        samplingRate = 0.1
        numExperiments = 100
    )
    
    var estimates []float64
    
    for i := 0; i < numExperiments; i++ {
        // 模拟采样过程
        observedCount := simulateSampling(trueCount, samplingRate)
        estimate := float64(observedCount) / samplingRate
        estimates = append(estimates, estimate)
    }
    
    // 计算统计量
    mean := calculateMean(estimates)
    variance := calculateVariance(estimates, mean)
    stdError := math.Sqrt(variance / float64(numExperiments))
    
    // 验证无偏性（95% 置信区间）
    margin := 1.96 * stdError
    if math.Abs(mean - float64(trueCount)) > margin {
        t.Errorf("估计有偏：期望 %d，实际 %.2f ± %.2f", 
            trueCount, mean, margin)
    }
    
    // 验证方差公式
    expectedVariance := float64(trueCount) * (1 - samplingRate) / samplingRate
    if math.Abs(variance - expectedVariance) > expectedVariance * 0.1 {
        t.Errorf("方差不匹配：期望 %.2f，实际 %.2f", 
            expectedVariance, variance)
    }
}

func simulateSampling(total, rate float64) int {
    count := 0
    for i := 0; i < int(total); i++ {
        if rand.Float64() < rate {
            count++
        }
    }
    return count
}
```

## 背压安全性

### 数学模型

**条件**：系统稳定当且仅当

```
λ · E[S] ≤ μ · C
```

其中：

- λ：到达速率（traces/second）
- E[S]：平均 trace 大小（bytes）
- μ：处理速率（bytes/second）
- C：队列容量（bytes）

**证明**：根据 Little's Law，稳定状态下：

```
E[L] = λ · E[W]
```

其中 E[L] 为平均队列长度，E[W] 为平均等待时间。

当 E[L] > C 时，系统不稳定，导致数据丢失。

### 实验设计

```go
// experiments/backpressure_test.go
func TestBackpressureSafety(t *testing.T) {
    testCases := []struct {
        name string
        arrivalRate float64
        avgSize int64
        processingRate int64
        queueCapacity int64
        expectedStable bool
    }{
        {
            name: "stable_case",
            arrivalRate: 100,
            avgSize: 1024,
            processingRate: 200000, // 200KB/s
            queueCapacity: 10000000, // 10MB
            expectedStable: true,
        },
        {
            name: "unstable_case",
            arrivalRate: 1000,
            avgSize: 1024,
            processingRate: 200000,
            queueCapacity: 1000000, // 1MB
            expectedStable: false,
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // 模拟系统运行
            result := simulateSystem(tc.arrivalRate, tc.avgSize, 
                tc.processingRate, tc.queueCapacity)
            
            if result.stable != tc.expectedStable {
                t.Errorf("稳定性预期错误：期望 %v，实际 %v", 
                    tc.expectedStable, result.stable)
            }
            
            if !result.stable && result.dropRate < 0.01 {
                t.Errorf("不稳定系统应有明显丢包：丢包率 %.4f", result.dropRate)
            }
        })
    }
}
```

## 观测开销评估

### 基准测试框架

```go
// experiments/overhead_benchmark.go
func BenchmarkTelemetryOverhead(b *testing.B) {
    // 无观测基准
    b.Run("NoTelemetry", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            simulateBusinessLogic()
        }
    })
    
    // 有观测基准
    b.Run("WithTelemetry", func(b *testing.B) {
        provider := setupTelemetry()
        defer provider.Shutdown(context.Background())
        
        for i := 0; i < b.N; i++ {
            ctx, span := provider.Tracer("benchmark").Start(
                context.Background(), "business_logic")
            simulateBusinessLogic()
            span.End()
        }
    })
}

func BenchmarkMemoryOverhead(b *testing.B) {
    var m1, m2 runtime.MemStats
    
    // 测量无观测内存
    runtime.GC()
    runtime.ReadMemStats(&m1)
    
    provider := setupTelemetry()
    runtime.GC()
    runtime.ReadMemStats(&m2)
    
    overhead := m2.Alloc - m1.Alloc
    b.Logf("内存开销：%d bytes", overhead)
}
```

### 性能分析脚本

```bash
#!/bin/bash
# scripts/performance_analysis.sh

echo "=== 性能开销分析 ==="

# 1. CPU 开销
echo "1. CPU 开销对比"
go test -bench=BenchmarkTelemetryOverhead -cpuprofile=cpu.prof
go tool pprof -text cpu.prof | head -20

# 2. 内存开销
echo "2. 内存分配分析"
go test -bench=BenchmarkMemoryOverhead -memprofile=mem.prof
go tool pprof -text mem.prof | head -20

# 3. 延迟分布
echo "3. 延迟分布分析"
go test -bench=BenchmarkLatency -benchtime=10s
```

## 形式化验证

### 采样一致性证明

**定理**：在分布式系统中，如果所有节点使用相同的采样决策函数 f(trace_id)，则采样决策在整条链路中保持一致。

**证明**：
设 f: TraceID → {0,1} 为采样决策函数，对于 trace_id T：

- 如果 f(T) = 1，则整条链路被采样
- 如果 f(T) = 0，则整条链路不被采样

由于 f 是确定性函数，且所有节点使用相同的 f，因此决策一致。

### 误差边界分析

**定理**：在采样率 p 下，估计误差的 95% 置信区间为：

```
N̂ ± 1.96 · √(N · (1-p) / p)
```

**证明**：基于中心极限定理，当 N 足够大时：

```
(N̂ - N) / √(N · (1-p) / p) ~ N(0,1)
```

因此 95% 置信区间为：

```
N̂ ± 1.96 · √(N · (1-p) / p)
```

## 实验报告模板

### 采样无偏性实验

```markdown
## 实验 1：采样无偏性验证

### 实验设置
- 真实计数：10,000
- 采样率：0.1 (10%)
- 实验次数：100

### 结果
- 平均估计：9,987.3
- 标准差：316.2
- 95% 置信区间：[9,367.1, 10,607.5]
- 包含真实值：✓

### 结论
采样估计在统计上无偏，符合理论预期。
```

### 背压安全性实验

```markdown
## 实验 2：背压安全性验证

### 实验设置
- 到达速率：100 traces/s
- 平均大小：1KB
- 处理速率：200KB/s
- 队列容量：10MB

### 结果
- 系统稳定性：稳定
- 平均队列长度：512KB
- 丢包率：0.001%
- 平均延迟：2.56ms

### 结论
系统在给定负载下稳定运行，无显著数据丢失。
```

## 持续验证策略

### 自动化测试

```yaml
# .github/workflows/experiments.yml
name: Experiments
on: [push, pull_request]

jobs:
  sampling-bias:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
      - run: go test ./experiments -v -run TestSamplingBias
      
  backpressure:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
      - run: go test ./experiments -v -run TestBackpressureSafety
      
  performance:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
      - run: go test -bench=. -benchmem ./experiments
```

### 监控指标

- 采样率偏差监控
- 队列使用率告警
- 导出延迟分布
- 内存使用趋势
