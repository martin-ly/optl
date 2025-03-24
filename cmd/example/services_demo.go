package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"

	"optl/internal/services"
	"optl/internal/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func runServicesDemo() {
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
	logger.Info("Starting services demo")

	// 创建服务
	storage := services.NewStorage("main-storage")
	analyzer := services.NewAnalyzer("data-analyzer")
	processor := services.NewProcessor("main-processor", storage, analyzer)

	// 生成测试数据
	testData := generateTestData(5)

	// 创建等待组
	var wg sync.WaitGroup
	wg.Add(len(testData))

	// 并行处理数据
	for i, data := range testData {
		go func(idx int, inputData []byte) {
			defer wg.Done()

			// 为每个处理任务创建新的 span
			dataID := fmt.Sprintf("data-%d", idx+1)
			ctx, span := telemetry.ContextWithSpan(ctx, "process_data_task",
				trace.WithAttributes(
					attribute.Int("task.id", idx+1),
					attribute.String("data.id", dataID),
					attribute.Int("data.size", len(inputData)),
				),
			)
			defer span.End()

			// 从上下文中获取带有跟踪信息的日志记录器
			logger := telemetry.LoggerWithContext(ctx)
			logger.Info("Processing data task started",
				zap.Int("task_id", idx+1),
				zap.String("data_id", dataID),
				zap.Int("data_size", len(inputData)),
			)

			// 处理数据
			result, err := processor.ProcessData(ctx, dataID, inputData)
			if err != nil {
				logger.Error("Data processing failed",
					zap.Int("task_id", idx+1),
					zap.String("data_id", dataID),
					zap.Error(err),
				)
				return
			}

			logger.Info("Data processing completed",
				zap.Int("task_id", idx+1),
				zap.String("data_id", dataID),
				zap.Int("result_size", len(result)),
				zap.String("result_preview", previewData(result)),
			)
		}(i, data)
	}

	// 等待所有处理完成
	wg.Wait()
	logger.Info("All data processing tasks completed")
}

// 生成测试数据
func generateTestData(count int) [][]byte {
	testData := make([][]byte, count)
	for i := 0; i < count; i++ {
		// 生成随机大小的数据
		size := 100 + rand.Intn(900)
		data := make([]byte, size)

		// 填充随机数据
		for j := range data {
			data[j] = byte(rand.Intn(256))
		}

		testData[i] = data
	}
	return testData
}

// 生成数据预览
func previewData(data []byte) string {
	// 返回前 10 个字节的十六进制表示
	previewSize := 10
	if len(data) < previewSize {
		previewSize = len(data)
	}
	return hex.EncodeToString(data[:previewSize]) + "..."
}
