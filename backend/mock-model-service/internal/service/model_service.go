package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/tensor-talks/mock-model-service/internal/client"
	"github.com/tensor-talks/mock-model-service/internal/kafka"
	"github.com/tensor-talks/mock-model-service/internal/model"
	"go.uber.org/zap"
)

// ModelService обрабатывает события чата и генерирует ответы модели.
// В будущем это будет marking-service.
type ModelService struct {
	producer         *kafka.Producer
	sessionMgr       *model.SessionManager
	sessionManagerCl *client.SessionManagerClient
	chatCRUDCl       *client.ChatCRUDClient
	resultsCRUDCl    *client.ResultsCRUDClient
	logger           *zap.Logger
	questionDelay    time.Duration
}

// NewModelService создаёт новый сервис модели.
func NewModelService(
	producer *kafka.Producer,
	sessionManagerCl *client.SessionManagerClient,
	chatCRUDCl *client.ChatCRUDClient,
	resultsCRUDCl *client.ResultsCRUDClient,
	questionDelaySeconds int,
	logger *zap.Logger,
) *ModelService {
	return &ModelService{
		producer:         producer,
		sessionMgr:       model.NewSessionManager(),
		sessionManagerCl: sessionManagerCl,
		chatCRUDCl:       chatCRUDCl,
		resultsCRUDCl:    resultsCRUDCl,
		logger:           logger,
		questionDelay:    time.Duration(questionDelaySeconds) * time.Second,
	}
}

// HandleChatStarted обрабатывает начало чата - получает программу интервью и отправляет первый вопрос.
func (s *ModelService) HandleChatStarted(ctx context.Context, sessionID, userID string) error {
	s.logger.Info("Chat started, getting interview program",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
	)

	// Парсим sessionID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session_id: %w", err)
	}

	// Получаем программу интервью от session-manager
	program, err := s.sessionManagerCl.GetInterviewProgram(ctx, sessionUUID)
	if err != nil {
		return fmt.Errorf("get interview program: %w", err)
	}

	s.logger.Info("Interview program received",
		zap.String("session_id", sessionID),
		zap.Int("questions_count", len(program.Questions)),
	)

	// Создаём или получаем состояние сессии и устанавливаем программу
	s.sessionMgr.GetOrCreate(sessionID, userID)
	s.sessionMgr.SetProgram(sessionID, program)

	// Небольшая задержка перед отправкой первого вопроса
	time.Sleep(s.questionDelay)

	// Получаем первый вопрос из программы
	firstQuestion, ok := s.sessionMgr.GetNextQuestion(sessionID)
	if !ok {
		return fmt.Errorf("no questions in program")
	}

	// Сначала сохраняем сообщение в chat-crud
	systemMsg := fmt.Sprintf("Вопрос: %s", firstQuestion)
	if err := s.chatCRUDCl.SaveMessage(ctx, sessionUUID, client.MessageTypeSystem, systemMsg); err != nil {
		s.logger.Warn("Failed to save system message to chat-crud, continuing",
			zap.String("session_id", sessionID),
			zap.Error(err),
		)
		// Продолжаем, даже если не удалось сохранить
	} else {
		s.logger.Info("System message saved to chat-crud",
			zap.String("session_id", sessionID),
		)
	}

	// Затем отправляем вопрос в Kafka
	if err := s.producer.SendModelQuestion(sessionID, userID, firstQuestion); err != nil {
		return fmt.Errorf("send first question: %w", err)
	}

	// Увеличиваем счётчик вопросов
	s.sessionMgr.IncrementQuestion(sessionID)

	s.logger.Info("First question sent",
		zap.String("session_id", sessionID),
		zap.String("question", firstQuestion),
		zap.Int("question_number", 1),
	)

	return nil
}

// HandleUserMessage обрабатывает сообщение пользователя - сохраняет его, затем отправляет следующий вопрос или завершает чат.
func (s *ModelService) HandleUserMessage(ctx context.Context, sessionID, userID, content, messageID string) error {
	s.logger.Info("User message received",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.String("message_id", messageID),
		zap.String("content", content),
	)

	// Парсим sessionID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session_id: %w", err)
	}

	// Сначала сохраняем сообщение пользователя в chat-crud
	if err := s.chatCRUDCl.SaveMessage(ctx, sessionUUID, client.MessageTypeUser, content); err != nil {
		s.logger.Warn("Failed to save user message to chat-crud, continuing",
			zap.String("session_id", sessionID),
			zap.Error(err),
		)
		// Продолжаем, даже если не удалось сохранить
	} else {
		s.logger.Info("User message saved to chat-crud",
			zap.String("session_id", sessionID),
		)
	}

	// Получаем состояние сессии
	s.sessionMgr.GetOrCreate(sessionID, userID)

	// Проверяем, есть ли ещё вопросы в программе
	if s.sessionMgr.ShouldComplete(sessionID) {
		questionsAsked := s.sessionMgr.GetQuestionCount(sessionID)
		s.logger.Info("All questions answered, completing chat",
			zap.String("session_id", sessionID),
			zap.Int("questions_asked", questionsAsked),
		)

		// Генерируем результаты
		score := s.generateScore(questionsAsked)
		feedback := s.generateFeedback(score)

		// Небольшая задержка перед завершением
		time.Sleep(s.questionDelay)

		// Сохраняем финальное сообщение системы (последнее)
		completionMsg := "Интервью завершено. Результаты будут доступны в разделе результатов."
		if err := s.chatCRUDCl.SaveMessage(ctx, sessionUUID, client.MessageTypeSystem, completionMsg); err != nil {
			s.logger.Warn("Failed to save completion message to chat-crud",
				zap.String("session_id", sessionID),
				zap.Error(err),
			)
		}

		// Сигнализируем chat-crud о завершении чата (создаём дамп)
		if err := s.chatCRUDCl.CreateChatDump(ctx, sessionUUID); err != nil {
			s.logger.Warn("Failed to create chat dump",
				zap.String("session_id", sessionID),
				zap.Error(err),
			)
		}

		// Сохраняем результаты в results-crud
		if err := s.resultsCRUDCl.SaveResult(ctx, sessionUUID, score, feedback); err != nil {
			s.logger.Error("Failed to save result to results-crud",
				zap.String("session_id", sessionID),
				zap.Error(err),
			)
			// Продолжаем, даже если не удалось сохранить результат
		}

		// Закрываем сессию в session-manager
		if err := s.sessionManagerCl.CloseSession(ctx, sessionUUID); err != nil {
			s.logger.Warn("Failed to close session in session-manager",
				zap.String("session_id", sessionID),
				zap.Error(err),
			)
		}

		// Отправляем событие завершения в Kafka (с результатами)
		recommendations := s.generateRecommendations(score)
		if err := s.producer.SendChatCompleted(sessionID, userID, score, feedback, recommendations); err != nil {
			return fmt.Errorf("send chat completed: %w", err)
		}

		// Удаляем сессию из локального менеджера
		s.sessionMgr.Delete(sessionID)

		s.logger.Info("Chat completed",
			zap.String("session_id", sessionID),
			zap.Int("score", score),
		)

		return nil
	}

	// Отправляем следующий вопрос
	time.Sleep(s.questionDelay)

	// Получаем следующий вопрос из программы
	nextQuestion, ok := s.sessionMgr.GetNextQuestion(sessionID)
	if !ok {
		return fmt.Errorf("failed to get next question from program")
	}

	// Сначала сохраняем сообщение системы в chat-crud
	systemMsg := fmt.Sprintf("Вопрос: %s", nextQuestion)
	if err := s.chatCRUDCl.SaveMessage(ctx, sessionUUID, client.MessageTypeSystem, systemMsg); err != nil {
		s.logger.Warn("Failed to save system message to chat-crud, continuing",
			zap.String("session_id", sessionID),
			zap.Error(err),
		)
	} else {
		s.logger.Info("System message saved to chat-crud",
			zap.String("session_id", sessionID),
		)
	}

	// Затем отправляем вопрос в Kafka
	if err := s.producer.SendModelQuestion(sessionID, userID, nextQuestion); err != nil {
		return fmt.Errorf("send question: %w", err)
	}

	// Увеличиваем счётчик после отправки вопроса
	s.sessionMgr.IncrementQuestion(sessionID)

	s.logger.Info("Question sent",
		zap.String("session_id", sessionID),
		zap.String("question", nextQuestion),
		zap.Int("question_number", s.sessionMgr.GetQuestionCount(sessionID)),
	)

	return nil
}

// generateScore генерирует оценку на основе количества вопросов и случайного фактора.
func (s *ModelService) generateScore(questionsAsked int) int {
	// Базовая оценка зависит от количества вопросов
	baseScore := 60 + (questionsAsked * 5)

	// Добавляем случайный фактор ±10
	randomFactor := rand.Intn(21) - 10

	score := baseScore + randomFactor
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score
}

// generateFeedback генерирует обратную связь на основе оценки.
func (s *ModelService) generateFeedback(score int) string {
	if score >= 90 {
		return "Отличное понимание основ машинного обучения. Вы продемонстрировали глубокие знания в ключевых областях."
	} else if score >= 75 {
		return "Хорошее понимание основ ML. Есть пробелы в некоторых областях, но общая картина ясна."
	} else if score >= 60 {
		return "Базовое понимание машинного обучения. Рекомендуется углубить знания в ключевых концепциях."
	} else {
		return "Требуется дополнительное изучение основ машинного обучения. Рекомендуется начать с фундаментальных концепций."
	}
}

// generateRecommendations генерирует рекомендации на основе оценки.
func (s *ModelService) generateRecommendations(score int) []string {
	recommendations := []string{}

	if score < 70 {
		recommendations = append(recommendations,
			"Изучить основы машинного обучения: линейная регрессия, классификация",
			"Практиковаться с реальными датасетами",
			"Изучить метрики оценки моделей (accuracy, precision, recall, F1)",
		)
	} else if score < 85 {
		recommendations = append(recommendations,
			"Углубить знания в регуляризации и обобщающей способности",
			"Изучить продвинутые техники feature engineering",
			"Практиковаться с ensemble методами",
		)
	} else {
		recommendations = append(recommendations,
			"Изучить продвинутые архитектуры нейронных сетей",
			"Практиковаться с оптимизацией гиперпараметров",
			"Изучить ML System Design",
		)
	}

	return recommendations
}
