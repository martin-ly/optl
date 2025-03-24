package services

import (
	"context"
	"fmt"
	"time"

	"optl/internal/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Processor 处理数据的服务
type Processor struct {
	name     string
	storage  *Storage
	analyzer *Analyzer
	logger   *zap.Logger
}

// NewProcessor 创建新的处理器
func NewProcessor(name string, storage *Storage, analyzer *Analyzer) *Processor {
	return &Processor{
		name:     name,
		storage:  storage,
		analyzer: analyzer,
		logger:   telemetry.Logger(),
	}
}

// ProcessData 处理数据并跟踪整个过程
func (p *Processor) ProcessData(ctx context.Context, dataID string, data []byte) ([]byte, error) {
	// 创建一个处理数据的 span
	ctx, span := telemetry.ContextWithSpan(ctx, "processor.process_data",
		trace.WithAttributes(
			attribute.String("processor.name", p.name),
			attribute.String("data.id", dataID),
			attribute.Int("data.size", len(data)),
		),
	)
	defer span.End()

	// 记录处理开始的事件
	telemetry.AddSpanEvent(ctx, "processing_started",
		attribute.String("data.id", dataID),
		attribute.Int("data.size", len(data)),
	)

	// 获取带有 trace 上下文的日志记录器
	logger := telemetry.LoggerWithContext(ctx)
	logger.Info("Processing data started",
		zap.String("processor", p.name),
		zap.String("data_id", dataID),
		zap.Int("data_size", len(data)),
	)

	// 验证数据
	err := p.validateData(ctx, data)
	if err != nil {
		span.RecordError(err)
		logger.Error("Data validation failed",
			zap.String("data_id", dataID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 转换数据 - 跨组件调用，传递上下文
	transformedData, err := p.transformData(ctx, data)
	if err != nil {
		span.RecordError(err)
		logger.Error("Data transformation failed",
			zap.String("data_id", dataID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("transformation failed: %w", err)
	}

	// 分析数据 - 跨服务调用，传递上下文
	analysisResult, err := p.analyzer.AnalyzeData(ctx, dataID, transformedData)
	if err != nil {
		span.RecordError(err)
		logger.Error("Data analysis failed",
			zap.String("data_id", dataID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	// 存储数据 - 跨服务调用，传递上下文
	err = p.storage.StoreData(ctx, dataID, analysisResult)
	if err != nil {
		span.RecordError(err)
		logger.Error("Data storage failed",
			zap.String("data_id", dataID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("storage failed: %w", err)
	}

	// 记录处理完成的事件
	telemetry.AddSpanEvent(ctx, "processing_completed",
		attribute.String("data.id", dataID),
		attribute.Int("result.size", len(analysisResult)),
	)

	logger.Info("Processing data completed",
		zap.String("processor", p.name),
		zap.String("data_id", dataID),
		zap.Int("result_size", len(analysisResult)),
	)

	return analysisResult, nil
}

// 验证数据
func (p *Processor) validateData(ctx context.Context, data []byte) error {
	return telemetry.WithSpan(ctx, "processor.validate_data", func(ctx context.Context) error {
		logger := telemetry.LoggerWithContext(ctx)
		logger.Debug("Validating data")

		// 模拟验证逻辑
		if len(data) == 0 {
			return fmt.Errorf("empty data")
		}

		// 添加延迟以模拟处理
		time.Sleep(20 * time.Millisecond)

		return nil
	})
}

// 转换数据
func (p *Processor) transformData(ctx context.Context, data []byte) ([]byte, error) {
	var result []byte

	// 使用 WithSpan 包装函数
	err := telemetry.WithSpan(ctx, "processor.transform_data", func(ctx context.Context) error {
		logger := telemetry.LoggerWithContext(ctx)
		logger.Debug("Transforming data")

		// 模拟转换逻辑
		result = make([]byte, len(data))
		copy(result, data)

		// 添加延迟以模拟处理
		time.Sleep(50 * time.Millisecond)

		// 模拟数据转换：反转数据
		for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
			result[i], result[j] = result[j], result[i]
		}

		return nil
	})

	return result, err
}
