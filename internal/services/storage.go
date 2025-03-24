package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"optl/internal/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Storage 用于存储数据的服务
type Storage struct {
	name   string
	data   map[string][]byte
	mu     sync.RWMutex
	logger *zap.Logger
}

// NewStorage 创建一个新的存储服务
func NewStorage(name string) *Storage {
	return &Storage{
		name:   name,
		data:   make(map[string][]byte),
		logger: telemetry.Logger(),
	}
}

// StoreData 存储数据并跟踪
func (s *Storage) StoreData(ctx context.Context, id string, data []byte) error {
	// 创建一个存储数据的 span
	ctx, span := telemetry.ContextWithSpan(ctx, "storage.store_data",
		trace.WithAttributes(
			attribute.String("storage.name", s.name),
			attribute.String("data.id", id),
			attribute.Int("data.size", len(data)),
		),
	)
	defer span.End()

	// 获取带有 trace 上下文的日志记录器
	logger := telemetry.LoggerWithContext(ctx)
	logger.Info("Storing data",
		zap.String("storage", s.name),
		zap.String("data_id", id),
		zap.Int("data_size", len(data)),
	)

	// 模拟存储操作的延迟
	err := telemetry.WithSpan(ctx, "storage.write_operation", func(ctx context.Context) error {
		// 添加延迟以模拟写入操作
		time.Sleep(30 * time.Millisecond)

		// 写入数据
		s.mu.Lock()
		s.data[id] = data
		s.mu.Unlock()

		// 模拟随机错误
		if len(data) > 1000000 {
			return fmt.Errorf("data too large to store")
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		logger.Error("Failed to store data",
			zap.String("storage", s.name),
			zap.String("data_id", id),
			zap.Error(err),
		)
		return fmt.Errorf("storage operation failed: %w", err)
	}

	logger.Info("Data stored successfully",
		zap.String("storage", s.name),
		zap.String("data_id", id),
	)
	return nil
}

// GetData 获取数据并跟踪
func (s *Storage) GetData(ctx context.Context, id string) ([]byte, error) {
	// 创建一个获取数据的 span
	ctx, span := telemetry.ContextWithSpan(ctx, "storage.get_data",
		trace.WithAttributes(
			attribute.String("storage.name", s.name),
			attribute.String("data.id", id),
		),
	)
	defer span.End()

	// 获取带有 trace 上下文的日志记录器
	logger := telemetry.LoggerWithContext(ctx)
	logger.Info("Retrieving data",
		zap.String("storage", s.name),
		zap.String("data_id", id),
	)

	var data []byte
	var exists bool

	// 模拟读取操作
	err := telemetry.WithSpan(ctx, "storage.read_operation", func(ctx context.Context) error {
		// 添加延迟以模拟读取操作
		time.Sleep(10 * time.Millisecond)

		// 读取数据
		s.mu.RLock()
		data, exists = s.data[id]
		s.mu.RUnlock()

		if !exists {
			return fmt.Errorf("data with id %s not found", id)
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		logger.Error("Failed to retrieve data",
			zap.String("storage", s.name),
			zap.String("data_id", id),
			zap.Error(err),
		)
		return nil, fmt.Errorf("storage operation failed: %w", err)
	}

	// 记录读取到的数据大小
	span.SetAttributes(attribute.Int("data.size", len(data)))

	logger.Info("Data retrieved successfully",
		zap.String("storage", s.name),
		zap.String("data_id", id),
		zap.Int("data_size", len(data)),
	)
	return data, nil
}
