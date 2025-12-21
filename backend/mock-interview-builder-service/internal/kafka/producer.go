package kafka

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/tensor-talks/mock-interview-builder-service/internal/model"
	"go.uber.org/zap"
)

// Producer отправляет события в Kafka (interview.build.response).
type Producer struct {
	producer    sarama.SyncProducer
	topic       string
	serviceName string
	version     string
	logger      *zap.Logger
}

// NewProducer создаёт новый Kafka producer.
func NewProducer(brokers []string, topic, serviceName, version string, logger *zap.Logger) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	return &Producer{
		producer:    producer,
		topic:       topic,
		serviceName: serviceName,
		version:     version,
		logger:      logger,
	}, nil
}

// SendInterviewProgram отправляет готовую программу интервью.
func (p *Producer) SendInterviewProgram(sessionID string, program model.InterviewProgram) error {
	// Конвертируем программу в JSON-совместимую структуру
	questions := make([]map[string]interface{}, len(program.Questions))
	for i, q := range program.Questions {
		questions[i] = map[string]interface{}{
			"question": q.Question,
			"theory":   q.Theory,
			"order":    q.Order,
		}
	}

	programPayload := map[string]interface{}{
		"questions": questions,
	}

	event := InterviewBuilderEvent{
		EventID:   uuid.New().String(),
		EventType: "interview.build.response",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Service:   p.serviceName,
		Version:   p.version,
		Payload: map[string]interface{}{
			"session_id": sessionID,
			"program":    programPayload,
		},
	}

	return p.sendEvent(event, sessionID)
}

func (p *Producer) sendEvent(event InterviewBuilderEvent, sessionID string) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Value: sarama.StringEncoder(data),
		Key:   sarama.StringEncoder(sessionID), // Используем session_id как ключ для партиционирования
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		p.logger.Error("Failed to send Kafka message",
			zap.String("event_type", event.EventType),
			zap.String("session_id", sessionID),
			zap.Error(err),
		)
		return fmt.Errorf("send message: %w", err)
	}

	p.logger.Info("Kafka message sent",
		zap.String("event_type", event.EventType),
		zap.String("session_id", sessionID),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset),
	)

	return nil
}

// Close закрывает producer.
func (p *Producer) Close() error {
	return p.producer.Close()
}
