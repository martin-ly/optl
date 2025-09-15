package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"optl/internal/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	serviceName = "optl-example"
)

// 模拟的工作项
type workItem struct {
	id     int
	name   string
	delay  time.Duration
	weight int
}

func main() {
	// 设置随机数种子
	rand.Seed(time.Now().UnixNano())

	// 根据命令行参数选择要运行的示例
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "services":
			runServicesDemo()
			return
		case "http":
			runHTTPDemo()
			return
		}
	}
	// 默认运行基本示例
	runBasicExample()
}

// 运行基本示例
func runBasicExample() {
	// 创建上下文，用于处理取消信号
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// 初始化遥测
	provider, err := initTelemetry()
	if err != nil {
		fmt.Printf("Failed to initialize telemetry: %v\n", err)
		os.Exit(1)
	}
	defer provider.Shutdown(context.Background())

	// 获取日志记录器
	logger := telemetry.Logger()
	logger.Info("Starting basic example", zap.String("service", serviceName))

	// 创建指标记录器
	meter := telemetry.Meter(serviceName)

	// 创建请求计数器
	requestCounter, err := meter.Int64Counter(
		"example.requests",
		metric.WithDescription("Number of requests processed"),
	)
	if err != nil {
		logger.Error("Failed to create counter", zap.Error(err))
		os.Exit(1)
	}

	// 创建处理时间记录器
	processingTime, err := meter.Float64Histogram(
		"example.processing_time",
		metric.WithDescription("Time taken to process requests"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		logger.Error("Failed to create histogram", zap.Error(err))
		os.Exit(1)
	}

	// 创建工作项大小记录器
	itemValueRecorder, err := meter.Int64Histogram(
		"example.item_weight",
		metric.WithDescription("Weight of processed items"),
	)
	if err != nil {
		logger.Error("Failed to create recorder", zap.Error(err))
		os.Exit(1)
	}

	// 启动多个并发处理器
	const numWorkers = 3
	workItems := generateWorkItems(20)

	// 使用 GoWithLimitAndSpan 并行处理
	err = telemetry.GoWithLimitAndSpan(ctx, "process_batch", numWorkers, workItems, func(ctx context.Context, item workItem) error {
		return processItem(ctx, item, requestCounter, processingTime, itemValueRecorder)
	})

	if err != nil {
		logger.Error("Processing failed", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Application stopping")
}

// 初始化遥测系统
func initTelemetry() (*telemetry.Provider, error) {
	config := telemetry.DefaultConfig()
	config.ServiceName = serviceName
	config.Environment = "development"
	config.EnableConsoleExporter = true

	return telemetry.NewProvider(config)
}

// 生成工作项
func generateWorkItems(count int) []workItem {
	items := make([]workItem, count)
	for i := 0; i < count; i++ {
		items[i] = workItem{
			id:     i + 1,
			name:   fmt.Sprintf("item-%d", i+1),
			delay:  time.Duration(50+rand.Intn(200)) * time.Millisecond,
			weight: 1 + rand.Intn(100),
		}
	}
	return items
}

// 处理单个工作项
func processItem(
	ctx context.Context,
	item workItem,
	counter metric.Int64Counter,
	histogram metric.Float64Histogram,
	weightHistogram metric.Int64Histogram,
) error {
	// 创建带有工作项属性的子 span
	ctx, span := telemetry.ContextWithSpan(ctx, "process_item",
		trace.WithAttributes(
			attribute.Int("item.id", item.id),
			attribute.String("item.name", item.name),
			attribute.Int("item.weight", item.weight),
		),
	)
	defer span.End()

	// 获取带跟踪上下文的日志记录器
	logger := telemetry.LoggerWithContext(ctx)
	logger.Info("Processing item",
		zap.Int("item_id", item.id),
		zap.String("item_name", item.name),
	)

	// 模拟处理
	startTime := time.Now()

	// 记录事件
	telemetry.AddSpanEvent(ctx, "item_processing_started",
		attribute.Int("item.weight", item.weight),
	)

	// 模拟处理阶段
	err := simulateProcessingStages(ctx, item)
	if err != nil {
		return err
	}

	// 计算处理时间
	duration := time.Since(startTime)
	durationMs := float64(duration.Milliseconds())

	// 记录指标
	counter.Add(ctx, 1, metric.WithAttributes(
		attribute.Int("item.id", item.id),
		attribute.String("item.name", item.name),
	))

	histogram.Record(ctx, durationMs, metric.WithAttributes(
		attribute.Int("item.id", item.id),
		attribute.String("item.name", item.name),
	))

	weightHistogram.Record(ctx, int64(item.weight), metric.WithAttributes(
		attribute.Int("item.id", item.id),
		attribute.String("item.name", item.name),
	))

	logger.Info("Item processed",
		zap.Int("item_id", item.id),
		zap.String("item_name", item.name),
		zap.Duration("processing_time", duration),
		zap.Int("weight", item.weight),
	)

	return nil
}

// 模拟多阶段处理
func simulateProcessingStages(ctx context.Context, item workItem) error {
	stages := []struct {
		name     string
		duration time.Duration
		failRate float64
	}{
		{"validate", item.delay / 4, 0.05},
		{"transform", item.delay / 2, 0.02},
		{"store", item.delay / 4, 0.01},
	}

	for _, stage := range stages {
		// 为每个阶段创建子 span
		err := telemetry.WithSpan(ctx, fmt.Sprintf("stage_%s", stage.name), func(stageCtx context.Context) error {
			// 获取带跟踪上下文的日志记录器
			logger := telemetry.LoggerWithContext(stageCtx)
			logger.Debug("Processing stage",
				zap.String("stage", stage.name),
				zap.Int("item_id", item.id),
			)

			// 模拟阶段处理
			time.Sleep(stage.duration)

			// 模拟随机失败
			if rand.Float64() < stage.failRate {
				err := fmt.Errorf("random failure in stage %s", stage.name)
				logger.Warn("Stage failed",
					zap.String("stage", stage.name),
					zap.Int("item_id", item.id),
					zap.Error(err),
				)
				return err
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}
