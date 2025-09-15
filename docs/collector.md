# Collector 流水线与配置矩阵

## 目标

- 收敛三信号，提供开发/预发/生产三套配置基线。
- 支持一键部署与可观测性栈集成。

## 开发环境（dev）

### 完整配置

```yaml
# otelcol-dev.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 1s
    send_batch_size: 1024
  memory_limiter:
    limit_mib: 512
  resource:
    attributes:
      - key: environment
        value: development
        action: insert

exporters:
  logging:
    loglevel: debug
  otlp/tempo:
    endpoint: tempo:4317
    tls:
      insecure: true
  prometheus:
    endpoint: "0.0.0.0:8889"
  loki:
    endpoint: http://loki:3100/loki/api/v1/push

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch, resource]
      exporters: [logging, otlp/tempo]
    metrics:
      receivers: [otlp]
      processors: [memory_limiter, batch, resource]
      exporters: [logging, prometheus]
    logs:
      receivers: [otlp]
      processors: [memory_limiter, batch, resource]
      exporters: [logging, loki]
```

### Docker Compose 集成

```yaml
# docker-compose.dev.yml
version: '3.8'
services:
  otelcol:
    image: otel/opentelemetry-collector-contrib:latest
    command: ["--config=/etc/otelcol-dev.yaml"]
    volumes:
      - ./configs/otelcol-dev.yaml:/etc/otelcol-dev.yaml
    ports:
      - "4317:4317"
      - "4318:4318"
      - "8889:8889"
    depends_on:
      - tempo
      - prometheus
      - loki

  tempo:
    image: grafana/tempo:latest
    command: [ "-config.file=/etc/tempo.yaml" ]
    volumes:
      - ./configs/tempo.yaml:/etc/tempo.yaml
    ports:
      - "3200:3200"

  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./configs/prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"

  loki:
    image: grafana/loki:latest
    ports:
      - "3100:3100"

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
```

## 预发环境（staging）

### 配置要点

```yaml
# otelcol-staging.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
        tls:
          cert_file: /etc/ssl/certs/server.crt
          key_file: /etc/ssl/private/server.key

processors:
  batch:
    timeout: 5s
    send_batch_size: 2048
  memory_limiter:
    limit_mib: 2048
  resource:
    attributes:
      - key: environment
        value: staging
        action: insert
  attributes:
    actions:
      - key: user.email
        action: delete
      - key: user.phone
        action: delete
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
          sampling_percentage: 10

exporters:
  otlp/tempo:
    endpoint: tempo-staging:4317
    tls:
      cert_file: /etc/ssl/certs/client.crt
      key_file: /etc/ssl/private/client.key
  prometheus:
    endpoint: "0.0.0.0:8889"
  loki:
    endpoint: http://loki-staging:3100/loki/api/v1/push
    headers:
      X-Scope-OrgID: staging
```

## 生产环境（prod）

### 高可用配置

```yaml
# otelcol-prod.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
        tls:
          cert_file: /etc/ssl/certs/server.crt
          key_file: /etc/ssl/private/server.key
          client_ca_file: /etc/ssl/certs/ca.crt
        auth:
          authenticator: basicauth/client

processors:
  batch:
    timeout: 10s
    send_batch_size: 4096
  memory_limiter:
    limit_mib: 4096
  resource:
    attributes:
      - key: environment
        value: production
        action: insert
      - key: cluster
        from_attribute: k8s.cluster.name
        action: insert
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
  tail_sampling:
    decision_wait: 30s
    num_traces: 100000
    expected_new_traces_per_sec: 50
    policies:
      - name: error-priority
        type: status_code
        status_code:
          status_codes: [ERROR]
      - name: slow-request
        type: latency
        latency:
          threshold_ms: 2000
      - name: high-value-customers
        type: string_attribute
        string_attribute:
          key: customer.tier
          values: [premium, enterprise]
      - name: random-sampling
        type: probabilistic
        probabilistic:
          sampling_percentage: 5

exporters:
  otlp/tempo:
    endpoint: tempo-prod:4317
    tls:
      cert_file: /etc/ssl/certs/client.crt
      key_file: /etc/ssl/private/client.key
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
      max_elapsed_time: 300s
    sending_queue:
      enabled: true
      num_consumers: 10
      queue_size: 10000
  prometheus:
    endpoint: "0.0.0.0:8889"
  loki:
    endpoint: http://loki-prod:3100/loki/api/v1/push
    headers:
      X-Scope-OrgID: production
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s

extensions:
  health_check:
    endpoint: 0.0.0.0:13133
  pprof:
    endpoint: 0.0.0.0:1777
  zpages:
    endpoint: 0.0.0.0:55679

service:
  extensions: [health_check, pprof, zpages]
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, resource, attributes, tail_sampling, batch]
      exporters: [otlp/tempo]
    metrics:
      receivers: [otlp]
      processors: [memory_limiter, resource, batch]
      exporters: [prometheus]
    logs:
      receivers: [otlp]
      processors: [memory_limiter, resource, attributes, batch]
      exporters: [loki]
```

## Kubernetes 部署

### ConfigMap

```yaml
# k8s/otelcol-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: otelcol-config
  namespace: observability
data:
  otelcol.yaml: |
    # 生产配置内容
```

### Deployment

```yaml
# k8s/otelcol-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: otelcol
  namespace: observability
spec:
  replicas: 3
  selector:
    matchLabels:
      app: otelcol
  template:
    metadata:
      labels:
        app: otelcol
    spec:
      containers:
      - name: otelcol
        image: otel/opentelemetry-collector-contrib:latest
        args: ["--config=/etc/otelcol/otelcol.yaml"]
        volumeMounts:
        - name: config
          mountPath: /etc/otelcol
        - name: certs
          mountPath: /etc/ssl/certs
          readOnly: true
        ports:
        - containerPort: 4317
          name: otlp-grpc
        - containerPort: 4318
          name: otlp-http
        - containerPort: 8889
          name: prometheus
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /
            port: 13133
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /
            port: 13133
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: otelcol-config
      - name: certs
        secret:
          secretName: otelcol-certs
```

## 监控与告警

### Collector 自身指标

```yaml
# 监控 Collector 健康状态
- alert: OtelColDown
  expr: up{job="otelcol"} == 0
  for: 1m
  labels:
    severity: critical
  annotations:
    summary: "OpenTelemetry Collector 实例下线"

- alert: OtelColHighMemory
  expr: process_resident_memory_bytes{job="otelcol"} > 2e9
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Collector 内存使用过高"
```

## 配置管理最佳实践

### 环境变量注入

```yaml
# 使用环境变量动态配置
exporters:
  otlp/tempo:
    endpoint: ${TEMPO_ENDPOINT}
    tls:
      cert_file: ${TLS_CERT_FILE}
      key_file: ${TLS_KEY_FILE}
```

### 配置验证

```bash
# 验证配置文件语法
otelcol --config=otelcol.yaml --dry-run

# 检查配置完整性
otelcol --config=otelcol.yaml --check-config
```

## 性能调优指南

### 资源分配

- **CPU**: 根据处理量分配，建议 0.5-2 cores
- **内存**: 根据队列大小和批处理配置，建议 1-4GB
- **网络**: 确保足够的带宽用于数据导出

### 批处理优化

- 根据延迟要求调整 `timeout`
- 根据内存限制调整 `send_batch_size`
- 监控队列使用率，避免背压

### 采样策略

- 生产环境建议使用 Tail-based 采样
- 根据错误率和延迟要求调整采样策略
- 定期评估采样效果和成本
