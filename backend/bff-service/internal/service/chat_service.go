package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tensor-talks/bff-service/internal/client"
	"github.com/tensor-talks/bff-service/internal/kafka"
	"go.uber.org/zap"
)

// ChatService управляет чатами и сессиями.
type ChatService struct {
	sessionClient *client.SessionClient
	kafkaProducer *kafka.Producer
	logger        *zap.Logger
	// Хранилище активных сессий
	sessions sync.Map // map[string]*Session
}

// Session представляет активную сессию чата.
type Session struct {
	SessionID string
	UserID    string
	// Очередь новых вопросов от модели
	Questions chan QuestionUpdate
	// Результаты чата (если завершён)
	Results *ChatResults
	mu      sync.RWMutex
}

// QuestionUpdate представляет обновление с вопросом от модели.
type QuestionUpdate struct {
	Question   string
	QuestionID string
	Timestamp  time.Time
}

// ChatResults представляет результаты завершенного чата.
type ChatResults struct {
	Score           int
	Feedback        string
	Recommendations []string
	CompletedAt     time.Time
}

// NewChatService создаёт новый сервис для работы с чатами.
func NewChatService(sessionClient *client.SessionClient, kafkaProducer *kafka.Producer, logger *zap.Logger) *ChatService {
	return &ChatService{
		sessionClient: sessionClient,
		kafkaProducer: kafkaProducer,
		logger:        logger,
	}
}

// StartChat создаёт новую сессию и отправляет событие начала чата в Kafka.
func (s *ChatService) StartChat(ctx context.Context, userID string) (string, error) {
	// Создаём сессию через session-service
	sessionResp, err := s.sessionClient.CreateSession(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to create session",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return "", fmt.Errorf("create session: %w", err)
	}

	sessionID := sessionResp.SessionID

	// Сохраняем сессию
	session := &Session{
		SessionID: sessionID,
		UserID:    userID,
		Questions: make(chan QuestionUpdate, 10), // Буферизованный канал для вопросов
		Results:   nil,
	}
	s.sessions.Store(sessionID, session)

	// Отправляем событие начала чата в Kafka
	requestID := uuid.New().String()
	if err := s.kafkaProducer.SendChatStarted(sessionID, userID, requestID); err != nil {
		s.logger.Error("Failed to send chat started event",
			zap.String("session_id", sessionID),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		// Не возвращаем ошибку, так как сессия уже создана
	}

	s.logger.Info("Chat started",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
	)

	return sessionID, nil
}

// SendMessage отправляет сообщение пользователя в Kafka.
func (s *ChatService) SendMessage(ctx context.Context, sessionID, userID, content string) error {
	// Проверяем, что сессия существует
	if _, ok := s.sessions.Load(sessionID); !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	messageID := uuid.New().String()
	requestID := uuid.New().String()

	if err := s.kafkaProducer.SendUserMessage(sessionID, userID, content, messageID, requestID); err != nil {
		s.logger.Error("Failed to send user message",
			zap.String("session_id", sessionID),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return fmt.Errorf("send message: %w", err)
	}

	s.logger.Info("User message sent",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.String("message_id", messageID),
	)

	return nil
}

// HandleModelQuestion обрабатывает вопрос от модели (реализует kafka.EventHandler).
func (s *ChatService) HandleModelQuestion(ctx context.Context, sessionID, userID, question, questionID string) error {
	s.logger.Info("Received model question",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.String("question_id", questionID),
	)

	// Находим сессию и добавляем вопрос в очередь
	if state, ok := s.sessions.Load(sessionID); ok {
		session := state.(*Session)
		update := QuestionUpdate{
			Question:   question,
			QuestionID: questionID,
			Timestamp:  time.Now(),
		}

		// Неблокирующая отправка в канал
		select {
		case session.Questions <- update:
			s.logger.Info("Question added to session queue",
				zap.String("session_id", sessionID),
			)
		default:
			s.logger.Warn("Session questions channel full, dropping question",
				zap.String("session_id", sessionID),
			)
		}
	} else {
		s.logger.Warn("Session not found for question",
			zap.String("session_id", sessionID),
		)
	}

	return nil
}

// HandleChatCompleted обрабатывает завершение чата (реализует kafka.EventHandler).
func (s *ChatService) HandleChatCompleted(ctx context.Context, sessionID, userID string, results kafka.ChatResults) error {
	s.logger.Info("Chat completed",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.Int("score", results.Score),
	)

	// Сохраняем результаты в сессии
	if state, ok := s.sessions.Load(sessionID); ok {
		session := state.(*Session)
		session.mu.Lock()
		session.Results = &ChatResults{
			Score:           results.Score,
			Feedback:        results.Feedback,
			Recommendations: results.Recommendations,
			CompletedAt:     time.Now(),
		}
		session.mu.Unlock()

		// Закрываем канал вопросов
		close(session.Questions)
	} else {
		s.logger.Warn("Session not found for completion",
			zap.String("session_id", sessionID),
		)
	}

	return nil
}

// GetNextQuestion получает следующий вопрос для сессии (для polling).
func (s *ChatService) GetNextQuestion(sessionID string) (*QuestionUpdate, bool) {
	if state, ok := s.sessions.Load(sessionID); ok {
		session := state.(*Session)
		select {
		case question := <-session.Questions:
			return &question, true
		default:
			return nil, false
		}
	}
	return nil, false
}

// GetResults получает результаты чата (если завершён).
func (s *ChatService) GetResults(sessionID string) (*ChatResults, bool) {
	if state, ok := s.sessions.Load(sessionID); ok {
		session := state.(*Session)
		session.mu.RLock()
		defer session.mu.RUnlock()
		if session.Results != nil {
			return session.Results, true
		}
	}
	return nil, false
}
