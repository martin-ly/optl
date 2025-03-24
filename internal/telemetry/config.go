package telemetry

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds the configuration for telemetry setup
type Config struct {
	// 服务名称
	ServiceName string
	// 服务版本
	ServiceVersion string
	// 环境（dev, staging, prod, etc.）
	Environment string
	// 额外的资源属性
	ResourceAttributes map[string]string
	// OTLP 导出器端点
	OTLPEndpoint string
	// 是否启用控制台导出器
	EnableConsoleExporter bool
	// 批处理的时间间隔
	BatchTimeout time.Duration
	// 批处理的最大导出大小
	MaxExportBatchSize int
	// 采样率 (0.0-1.0)
	SamplingRatio float64
	// 是否启用 metric 导出
	EnableMetrics bool
	// 是否启用 log 导出
	EnableLogs bool
	// Metric 收集间隔
	MetricCollectionInterval time.Duration
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		ServiceName:              getEnv("OTEL_SERVICE_NAME", "optl-service"),
		ServiceVersion:           getEnv("OTEL_SERVICE_VERSION", "v0.1.0"),
		Environment:              getEnv("OTEL_ENVIRONMENT", "development"),
		ResourceAttributes:       parseResourceAttributes(getEnv("OTEL_RESOURCE_ATTRIBUTES", "")),
		OTLPEndpoint:             getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		EnableConsoleExporter:    getEnvBool("OTEL_ENABLE_CONSOLE_EXPORTER", true),
		BatchTimeout:             getEnvDuration("OTEL_BATCH_TIMEOUT", 5*time.Second),
		MaxExportBatchSize:       getEnvInt("OTEL_MAX_EXPORT_BATCH_SIZE", 512),
		SamplingRatio:            getEnvFloat("OTEL_SAMPLING_RATIO", 1.0),
		EnableMetrics:            getEnvBool("OTEL_ENABLE_METRICS", true),
		EnableLogs:               getEnvBool("OTEL_ENABLE_LOGS", true),
		MetricCollectionInterval: getEnvDuration("OTEL_METRIC_COLLECTION_INTERVAL", 10*time.Second),
	}
}

// getEnv 获取环境变量值，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvBool 获取布尔类型的环境变量
func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		return strings.ToLower(value) == "true"
	}
	return defaultValue
}

// getEnvInt 获取整数类型的环境变量
func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := parseIntEnv(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvFloat 获取浮点类型的环境变量
func getEnvFloat(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if floatValue, err := parseFloatEnv(value); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// getEnvDuration 获取时间间隔类型的环境变量
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if durationValue, err := time.ParseDuration(value); err == nil {
			return durationValue
		}
	}
	return defaultValue
}

// parseResourceAttributes 解析资源属性字符串（key1=value1,key2=value2）
func parseResourceAttributes(attributesStr string) map[string]string {
	attributes := make(map[string]string)
	if attributesStr == "" {
		return attributes
	}

	pairs := strings.Split(attributesStr, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			attributes[kv[0]] = kv[1]
		}
	}
	return attributes
}

// 解析整数环境变量
func parseIntEnv(value string) (int, error) {
	var intValue int
	_, err := fmt.Sscanf(value, "%d", &intValue)
	return intValue, err
}

// 解析浮点环境变量
func parseFloatEnv(value string) (float64, error) {
	var floatValue float64
	_, err := fmt.Sscanf(value, "%f", &floatValue)
	return floatValue, err
}
