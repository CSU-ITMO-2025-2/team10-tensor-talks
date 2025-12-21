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
	sessionCRUDCl *client.SessionCRUDClient
	chatCRUDCl    *client.ChatCRUDClient
	resultsCRUDCl *client.ResultsCRUDClient
	kafkaProducer *kafka.Producer
	logger        *zap.Logger
	// Хранилище активных сессий (только для активных чатов)
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
func NewChatService(
	sessionClient *client.SessionClient,
	sessionCRUDCl *client.SessionCRUDClient,
	chatCRUDCl *client.ChatCRUDClient,
	resultsCRUDCl *client.ResultsCRUDClient,
	kafkaProducer *kafka.Producer,
	logger *zap.Logger,
) *ChatService {
	return &ChatService{
		sessionClient: sessionClient,
		sessionCRUDCl: sessionCRUDCl,
		chatCRUDCl:    chatCRUDCl,
		resultsCRUDCl: resultsCRUDCl,
		kafkaProducer: kafkaProducer,
		logger:        logger,
	}
}

// StartChat создаёт новую сессию с параметрами интервью и отправляет событие начала чата в Kafka.
func (s *ChatService) StartChat(ctx context.Context, userID uuid.UUID, params client.SessionParams) (uuid.UUID, error) {
	// Создаём сессию через session-manager-service с параметрами
	sessionResp, err := s.sessionClient.CreateSession(ctx, userID, params)
	if err != nil {
		s.logger.Error("Failed to create session",
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
		return uuid.Nil, fmt.Errorf("create session: %w", err)
	}

	sessionID := sessionResp.SessionID
	sessionIDStr := sessionID.String()
	userIDStr := userID.String()

	// Сохраняем сессию в локальное хранилище (для активных чатов)
	session := &Session{
		SessionID: sessionIDStr,
		UserID:    userIDStr,
		Questions: make(chan QuestionUpdate, 10), // Буферизованный канал для вопросов
		Results:   nil,
	}
	s.sessions.Store(sessionIDStr, session)

	// Отправляем событие начала чата в Kafka
	requestID := uuid.New().String()
	if err := s.kafkaProducer.SendChatStarted(sessionIDStr, userIDStr, requestID); err != nil {
		s.logger.Error("Failed to send chat started event",
			zap.String("session_id", sessionIDStr),
			zap.String("user_id", userIDStr),
			zap.Error(err),
		)
		// Не возвращаем ошибку, так как сессия уже создана
	}

	s.logger.Info("Chat started",
		zap.String("session_id", sessionIDStr),
		zap.String("user_id", userIDStr),
	)

	return sessionID, nil
}

// SendMessage отправляет сообщение пользователя в Kafka.
func (s *ChatService) SendMessage(ctx context.Context, sessionID uuid.UUID, userID uuid.UUID, content string) error {
	sessionIDStr := sessionID.String()
	// Проверяем, что сессия существует (для активных чатов)
	if _, ok := s.sessions.Load(sessionIDStr); !ok {
		// Для завершенных чатов это нормально, продолжаем
		s.logger.Info("Session not in active sessions, continuing",
			zap.String("session_id", sessionIDStr),
		)
	}

	messageID := uuid.New().String()
	requestID := uuid.New().String()

	if err := s.kafkaProducer.SendUserMessage(sessionIDStr, userID.String(), content, messageID, requestID); err != nil {
		s.logger.Error("Failed to send user message",
			zap.String("session_id", sessionIDStr),
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("send message: %w", err)
	}

	s.logger.Info("User message sent",
		zap.String("session_id", sessionIDStr),
		zap.String("user_id", userID.String()),
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

// GetInterviews получает список всех интервью пользователя (из session-crud и results-crud).
func (s *ChatService) GetInterviews(ctx context.Context, userID uuid.UUID) ([]InterviewInfo, error) {
	// Получаем все сессии пользователя
	sessions, err := s.sessionCRUDCl.GetSessionsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get sessions: %w", err)
	}

	// Получаем session IDs для запроса результатов
	sessionIDs := make([]uuid.UUID, len(sessions))
	for i, session := range sessions {
		sessionIDs[i] = session.SessionID
	}

	// Получаем результаты для всех сессий
	results, err := s.resultsCRUDCl.GetResults(ctx, sessionIDs)
	if err != nil {
		s.logger.Warn("Failed to get results, continuing without them",
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
		results = []client.Result{} // Продолжаем без результатов
	}

	// Создаём map для быстрого поиска результатов по session_id
	resultsMap := make(map[uuid.UUID]client.Result)
	for _, result := range results {
		resultsMap[result.SessionID] = result
	}

	// Формируем ответ
	interviews := make([]InterviewInfo, len(sessions))
	for i, session := range sessions {
		result, hasResult := resultsMap[session.SessionID]
		interviews[i] = InterviewInfo{
			SessionID:  session.SessionID,
			StartTime:  session.StartTime,
			EndTime:    session.EndTime,
			Params:     session.Params,
			HasResults: hasResult,
			Score:      result.Score,
			Feedback:   result.Feedback,
		}
	}

	return interviews, nil
}

// InterviewInfo представляет информацию об интервью для списка.
type InterviewInfo struct {
	SessionID  uuid.UUID            `json:"session_id"`
	StartTime  time.Time            `json:"start_time"`
	EndTime    *time.Time           `json:"end_time,omitempty"`
	Params     client.SessionParams `json:"params"`
	HasResults bool                 `json:"has_results"`
	Score      int                  `json:"score,omitempty"`
	Feedback   string               `json:"feedback,omitempty"`
}

// GetChatHistory получает историю чата по session_id (из chat-crud).
func (s *ChatService) GetChatHistory(ctx context.Context, sessionID uuid.UUID) ([]client.ChatMessage, error) {
	messages, err := s.chatCRUDCl.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	return messages, nil
}

// GetChatResult получает результат интервью по session_id (из results-crud).
func (s *ChatService) GetChatResult(ctx context.Context, sessionID uuid.UUID) (*client.Result, error) {
	result, err := s.resultsCRUDCl.GetResult(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get result: %w", err)
	}
	return result, nil
}
