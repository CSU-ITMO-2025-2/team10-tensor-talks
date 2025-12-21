package service

import (
	"context"
	"fmt"

	"github.com/tensor-talks/mock-interview-builder-service/internal/kafka"
	"github.com/tensor-talks/mock-interview-builder-service/internal/model"
	"go.uber.org/zap"
)

// InterviewBuilderService обрабатывает запросы на создание программы интервью.
type InterviewBuilderService struct {
	producer *kafka.Producer
	logger   *zap.Logger
}

// NewInterviewBuilderService создаёт новый сервис для создания программы интервью.
func NewInterviewBuilderService(producer *kafka.Producer, logger *zap.Logger) *InterviewBuilderService {
	return &InterviewBuilderService{
		producer: producer,
		logger:   logger,
	}
}

// HandleInterviewBuildRequest обрабатывает запрос на создание программы интервью.
func (s *InterviewBuilderService) HandleInterviewBuildRequest(ctx context.Context, sessionID string, params map[string]interface{}) error {
	s.logger.Info("Building interview program",
		zap.String("session_id", sessionID),
		zap.Any("params", params),
	)

	// Пока возвращаем статичную программу
	// В будущем здесь будет логика генерации программы на основе params (topics, level, type)
	program := model.GetStaticInterviewProgram()

	// Отправляем готовую программу в Kafka
	if err := s.producer.SendInterviewProgram(sessionID, program); err != nil {
		return fmt.Errorf("send interview program: %w", err)
	}

	s.logger.Info("Interview program sent",
		zap.String("session_id", sessionID),
		zap.Int("questions_count", len(program.Questions)),
	)

	return nil
}
