package services

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"optl/internal/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Analyzer 用于分析数据的服务
type Analyzer struct {
	name   string
	logger *zap.Logger
}

// NewAnalyzer 创建一个新的分析器服务
func NewAnalyzer(name string) *Analyzer {
	return &Analyzer{
		name:   name,
		logger: telemetry.Logger(),
	}
}

// AnalyzeData 分析数据并跟踪
func (a *Analyzer) AnalyzeData(ctx context.Context, id string, data []byte) ([]byte, error) {
	// 创建一个分析数据的 span
	ctx, span := telemetry.ContextWithSpan(ctx, "analyzer.analyze_data",
		trace.WithAttributes(
			attribute.String("analyzer.name", a.name),
			attribute.String("data.id", id),
			attribute.Int("data.size", len(data)),
		),
	)
	defer span.End()

	// 获取带有 trace 上下文的日志记录器
	logger := telemetry.LoggerWithContext(ctx)
	logger.Info("Analyzing data",
		zap.String("analyzer", a.name),
		zap.String("data_id", id),
		zap.Int("data_size", len(data)),
	)

	// 并行执行多个分析步骤
	analysisTasks := []struct {
		name string
		fn   func(ctx context.Context, data []byte) ([]byte, error)
	}{
		{"preprocess", a.preprocess},
		{"feature_extraction", a.extractFeatures},
		{"pattern_detection", a.detectPatterns},
	}

	// 使用管道模式进行数据处理，每个步骤传递上下文
	var processedData []byte = data
	var err error

	for _, task := range analysisTasks {
		taskData := processedData
		telemetry.AddSpanEvent(ctx, fmt.Sprintf("starting_%s", task.name),
			attribute.Int("input_size", len(taskData)),
		)

		// 使用 WithSpan 包装每个分析步骤
		err = telemetry.WithSpan(ctx, fmt.Sprintf("analyzer.%s", task.name), func(taskCtx context.Context) error {
			var taskErr error
			processedData, taskErr = task.fn(taskCtx, taskData)
			return taskErr
		})

		if err != nil {
			span.RecordError(err)
			logger.Error("Analysis step failed",
				zap.String("analyzer", a.name),
				zap.String("data_id", id),
				zap.String("step", task.name),
				zap.Error(err),
			)
			return nil, fmt.Errorf("analysis step '%s' failed: %w", task.name, err)
		}

		telemetry.AddSpanEvent(ctx, fmt.Sprintf("completed_%s", task.name),
			attribute.Int("output_size", len(processedData)),
		)
	}

	// 记录总结
	span.SetAttributes(attribute.Int("result.size", len(processedData)))

	logger.Info("Data analysis completed",
		zap.String("analyzer", a.name),
		zap.String("data_id", id),
		zap.Int("result_size", len(processedData)),
	)

	return processedData, nil
}

// 预处理数据
func (a *Analyzer) preprocess(ctx context.Context, data []byte) ([]byte, error) {
	logger := telemetry.LoggerWithContext(ctx)
	logger.Debug("Preprocessing data")

	// 模拟随机延迟
	delay := 20 + rand.Intn(30)
	time.Sleep(time.Duration(delay) * time.Millisecond)

	// 数据处理逻辑
	result := make([]byte, len(data))
	copy(result, data)

	// 简单的预处理 - 增加每个字节的值
	for i := range result {
		if result[i] < 255 {
			result[i]++
		}
	}

	return result, nil
}

// 提取特征
func (a *Analyzer) extractFeatures(ctx context.Context, data []byte) ([]byte, error) {
	logger := telemetry.LoggerWithContext(ctx)
	logger.Debug("Extracting features")

	// 模拟复杂的计算过程
	err := telemetry.GoWithSpan(ctx, "parallel_feature_extraction", func(ctx context.Context) error {
		// 模拟随机延迟
		delay := 30 + rand.Intn(40)
		time.Sleep(time.Duration(delay) * time.Millisecond)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 数据处理逻辑
	result := make([]byte, len(data))
	copy(result, data)

	// 随机模拟特征提取
	if len(result) > 0 {
		// 使用随机值替换一些字节，模拟特征提取的结果
		for i := 0; i < len(result)/10; i++ {
			idx := rand.Intn(len(result))
			result[idx] = byte(rand.Intn(256))
		}
	}

	return result, nil
}

// 检测模式
func (a *Analyzer) detectPatterns(ctx context.Context, data []byte) ([]byte, error) {
	logger := telemetry.LoggerWithContext(ctx)
	logger.Debug("Detecting patterns")

	// 模拟随机延迟
	delay := 25 + rand.Intn(35)
	time.Sleep(time.Duration(delay) * time.Millisecond)

	// 模拟错误
	if rand.Float64() < 0.01 {
		return nil, fmt.Errorf("pattern detection algorithm failed")
	}

	// 数据处理逻辑
	result := make([]byte, len(data))
	copy(result, data)

	// 简单的模式检测 - 对偶数索引的字节进行修改
	for i := 0; i < len(result); i += 2 {
		if i+1 < len(result) {
			// 交换相邻字节，模拟发现的模式
			result[i], result[i+1] = result[i+1], result[i]
		}
	}

	return result, nil
}
