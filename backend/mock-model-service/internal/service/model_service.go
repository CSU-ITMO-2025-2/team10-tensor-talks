package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/tensor-talks/mock-model-service/internal/kafka"
	"github.com/tensor-talks/mock-model-service/internal/model"
	"go.uber.org/zap"
)

// ModelService обрабатывает события чата и генерирует ответы модели.
type ModelService struct {
	producer      *kafka.Producer
	sessionMgr    *model.SessionManager
	logger        *zap.Logger
	maxQuestions  int
	questionDelay time.Duration
}

// NewModelService создаёт новый сервис модели.
func NewModelService(producer *kafka.Producer, maxQuestions, questionDelaySeconds int, logger *zap.Logger) *ModelService {
	return &ModelService{
		producer:      producer,
		sessionMgr:    model.NewSessionManager(),
		logger:        logger,
		maxQuestions:  maxQuestions,
		questionDelay: time.Duration(questionDelaySeconds) * time.Second,
	}
}

// HandleChatStarted обрабатывает начало чата - отправляет первый вопрос.
func (s *ModelService) HandleChatStarted(ctx context.Context, sessionID, userID string) error {
	s.logger.Info("Chat started, sending first question",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
	)

	// Создаём или получаем состояние сессии
	s.sessionMgr.GetOrCreate(sessionID, userID)

	// Небольшая задержка перед отправкой первого вопроса
	time.Sleep(s.questionDelay)

	// Отправляем первый вопрос (индекс 0)
	question := model.GetQuestionByIndex(0)
	if err := s.producer.SendModelQuestion(sessionID, userID, question); err != nil {
		return fmt.Errorf("send first question: %w", err)
	}

	// Увеличиваем счётчик вопросов (теперь 1)
	s.sessionMgr.IncrementQuestion(sessionID)

	s.logger.Info("First question sent",
		zap.String("session_id", sessionID),
		zap.String("question", question),
		zap.Int("question_number", 1),
	)

	return nil
}

// HandleUserMessage обрабатывает сообщение пользователя - отправляет следующий вопрос или завершает чат.
func (s *ModelService) HandleUserMessage(ctx context.Context, sessionID, userID, content, messageID string) error {
	s.logger.Info("User message received",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.String("message_id", messageID),
		zap.String("content", content),
	)

	// Получаем состояние сессии
	s.sessionMgr.GetOrCreate(sessionID, userID)
	currentQuestions := s.sessionMgr.GetQuestionCount(sessionID)

	s.logger.Info("Processing user message",
		zap.String("session_id", sessionID),
		zap.Int("questions_asked", currentQuestions),
		zap.Int("max_questions", s.maxQuestions),
	)

	// Проверяем, нужно ли завершить чат (после ответа на последний вопрос)
	// Если уже задано max_questions вопросов, значит пользователь ответил на последний - завершаем
	if currentQuestions >= s.maxQuestions {
		s.logger.Info("Max questions reached, completing chat",
			zap.String("session_id", sessionID),
			zap.Int("questions_asked", currentQuestions),
		)

		// Генерируем результаты
		score := s.generateScore(currentQuestions)
		feedback := s.generateFeedback(score)
		recommendations := s.generateRecommendations(score)

		// Небольшая задержка перед завершением
		time.Sleep(s.questionDelay)

		if err := s.producer.SendChatCompleted(sessionID, userID, score, feedback, recommendations); err != nil {
			return fmt.Errorf("send chat completed: %w", err)
		}

		// Удаляем сессию
		s.sessionMgr.Delete(sessionID)

		s.logger.Info("Chat completed",
			zap.String("session_id", sessionID),
			zap.Int("score", score),
		)

		return nil
	}

	// Отправляем следующий вопрос
	time.Sleep(s.questionDelay)

	// Используем индекс для разнообразия вопросов (currentQuestions уже содержит количество заданных)
	question := model.GetQuestionByIndex(currentQuestions)
	if err := s.producer.SendModelQuestion(sessionID, userID, question); err != nil {
		return fmt.Errorf("send question: %w", err)
	}

	// Увеличиваем счётчик после отправки вопроса
	s.sessionMgr.IncrementQuestion(sessionID)

	s.logger.Info("Question sent",
		zap.String("session_id", sessionID),
		zap.String("question", question),
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
