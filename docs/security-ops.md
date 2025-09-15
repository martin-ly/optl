# 安全与运维最佳实践（TLS/mTLS、重试、限流、尾采样）

## 连接安全

### TLS/mTLS 配置

```go
// 生产环境 OTLP 连接配置
func createSecureOTLPConnection(endpoint string) (*grpc.ClientConn, error) {
    // 加载客户端证书
    cert, err := tls.LoadX509KeyPair("client.crt", "client.key")
    if err != nil {
        return nil, fmt.Errorf("failed to load client cert: %w", err)
    }
    
    // 加载 CA 证书
    caCert, err := os.ReadFile("ca.crt")
    if err != nil {
        return nil, fmt.Errorf("failed to read CA cert: %w", err)
    }
    
    caCertPool := x509.NewCertPool()
    if !caCertPool.AppendCertsFromPEM(caCert) {
        return nil, fmt.Errorf("failed to parse CA cert")
    }
    
    // 配置 TLS
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      caCertPool,
        MinVersion:   tls.VersionTLS12,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
        },
    }
    
    return grpc.DialContext(ctx, endpoint,
        grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
        grpc.WithBlock(),
    )
}
```

### 证书轮换策略

- 使用 Kubernetes Secrets 或 HashiCorp Vault 管理证书
- 实现证书自动轮换，避免服务中断
- 监控证书过期时间，提前告警

## 稳定性与背压

### SDK 批处理配置

```go
// 根据流量特征调整批处理参数
bsp := sdktrace.NewBatchSpanProcessor(
    exporter,
    sdktrace.WithBatchTimeout(5*time.Second),        // 批超时
    sdktrace.WithMaxExportBatchSize(512),            // 最大批量
    sdktrace.WithMaxQueueSize(2048),                 // 队列容量
    sdktrace.WithExportTimeout(30*time.Second),      // 导出超时
)
```

### 重试与退避策略

```go
// 指数退避重试配置
retryConfig := otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{
    Enabled:         true,
    InitialInterval: 1 * time.Second,
    MaxInterval:     5 * time.Minute,
    MaxElapsedTime:  30 * time.Minute,
    Multiplier:      1.5,
    RandomizationFactor: 0.5,
})
```

### 背压保护

- 监控队列使用率，超过 80% 时告警
- 实现降级策略：高负载时仅保留错误和关键路径 Span
- 设置内存使用上限，防止 OOM

## 采样策略

### Head-based 采样（开发环境）

```go
// 开发环境：固定比例采样
sampler := sdktrace.TraceIDRatioBased(0.1) // 10% 采样率
```

### Tail-based 采样（生产环境）

```yaml
# Collector 配置：错误和慢请求优先
processors:
  tail_sampling:
    decision_wait: 10s
    num_traces: 50000
    expected_new_traces_per_sec: 10
    policies:
      - name: error-priority
        type: status_code
        status_code:
          status_codes: [ERROR]
      - name: slow-request
        type: latency
        latency:
          threshold_ms: 1000
      - name: random-sampling
        type: probabilistic
        probabilistic:
          sampling_percentage: 5
```

### 采样率校正

```go
// 在分析端进行无偏估计
func correctSamplingBias(observedCount int, samplingRate float64) float64 {
    return float64(observedCount) / samplingRate
}
```

## PII 与合规

### 属性过滤配置

```yaml
# Collector 属性处理器
processors:
  attributes:
    actions:
      - key: user.email
        action: delete
      - key: user.phone
        action: delete
      - key: request.body
        action: hash
        from_attribute: request.body
        to_attribute: request.body_hash
```

### 应用侧字段分类

```go
// 定义字段分类
const (
    FieldCategoryPublic    = "public"
    FieldCategoryInternal  = "internal"
    FieldCategorySensitive = "sensitive"
)

// 根据分类决定是否记录
func addSpanAttribute(key, value, category string) {
    if category == FieldCategorySensitive {
        // 仅记录哈希值
        hashed := sha256.Sum256([]byte(value))
        span.SetAttributes(attribute.String(key+"_hash", hex.EncodeToString(hashed[:])))
    } else {
        span.SetAttributes(attribute.String(key, value))
    }
}
```

## 运维与监控

### 健康指标暴露

```go
// 为遥测系统本身暴露指标
func (p *Provider) exposeHealthMetrics() {
    meter := otel.Meter("telemetry.health")
    
    queueDepth, _ := meter.Int64Gauge("telemetry_queue_depth")
    exportErrors, _ := meter.Int64Counter("telemetry_export_errors")
    exportLatency, _ := meter.Float64Histogram("telemetry_export_latency")
    
    // 定期更新指标
    go func() {
        ticker := time.NewTicker(10 * time.Second)
        for range ticker.C {
            queueDepth.Record(context.Background(), p.getQueueDepth())
            exportErrors.Add(context.Background(), p.getExportErrorCount())
            exportLatency.Record(context.Background(), p.getExportLatency())
        }
    }()
}
```

### SLO 与告警配置

```yaml
# Prometheus 告警规则
groups:
  - name: telemetry
    rules:
      - alert: TelemetryQueueHigh
        expr: telemetry_queue_depth > 1000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "遥测队列深度过高"
          
      - alert: TelemetryExportErrors
        expr: rate(telemetry_export_errors[5m]) > 0.1
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "遥测导出错误率过高"
```

### 仪表板配置

- RED 指标：请求速率、错误率、延迟分布
- Golden Signals：延迟、流量、错误、饱和度
- 遥测系统健康：队列深度、导出成功率、采样率

## 安全审计清单

- [ ] 所有 OTLP 连接使用 TLS/mTLS
- [ ] 证书自动轮换机制就位
- [ ] 敏感数据过滤规则配置
- [ ] 访问日志记录和监控
- [ ] 定期安全扫描和漏洞评估
- [ ] 备份和恢复策略测试
