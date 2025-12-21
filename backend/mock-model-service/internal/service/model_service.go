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

	// DEBUG: Log first question structure
	if len(program.Questions) > 0 {
		firstQ := program.Questions[0]
		s.logger.Warn("DEBUG: First question in program",
			zap.String("session_id", sessionID),
			zap.String("question", firstQ.Question),
			zap.Int("question_length", len(firstQ.Question)),
			zap.String("theory", firstQ.Theory),
			zap.Int("theory_length", len(firstQ.Theory)),
			zap.Int("order", firstQ.Order),
		)
	}

	// Создаём или получаем состояние сессии и устанавливаем программу
	s.sessionMgr.GetOrCreate(sessionID, userID)
	s.sessionMgr.SetProgram(sessionID, program)

	// Проверяем, нужно ли отправлять следующий вопрос
	if !s.sessionMgr.HasMoreQuestions(sessionID) {
		s.logger.Info("No more questions to ask, session already completed",
			zap.String("session_id", sessionID),
		)
		// Все вопросы уже заданы, завершаем чат
		questionsAsked := s.sessionMgr.GetQuestionCount(sessionID)
		score := s.generateScore(questionsAsked)
		feedback := s.generateFeedback(score)

		// Сохраняем финальное сообщение системы
		completionMsg := "Интервью завершено. Результаты будут доступны в разделе результатов."
		if err := s.chatCRUDCl.SaveMessage(ctx, sessionUUID, client.MessageTypeSystem, completionMsg); err != nil {
			s.logger.Warn("Failed to save completion message to chat-crud",
				zap.String("session_id", sessionID),
				zap.Error(err),
			)
		}

		// Создаём дамп чата
		if err := s.chatCRUDCl.CreateChatDump(ctx, sessionUUID); err != nil {
			s.logger.Warn("Failed to create chat dump",
				zap.String("session_id", sessionID),
				zap.Error(err),
			)
		}

		// Сохраняем результаты
		if err := s.resultsCRUDCl.SaveResult(ctx, sessionUUID, score, feedback, false); err != nil {
			s.logger.Error("Failed to save result to results-crud",
				zap.String("session_id", sessionID),
				zap.Error(err),
			)
		}

		// Закрываем сессию
		if err := s.sessionManagerCl.CloseSession(ctx, sessionUUID); err != nil {
			s.logger.Warn("Failed to close session in session-manager",
				zap.String("session_id", sessionID),
				zap.Error(err),
			)
		}

		// Отправляем событие завершения
		recommendations := s.generateRecommendations(score)
		if err := s.producer.SendChatCompleted(sessionID, userID, score, feedback, recommendations); err != nil {
			return fmt.Errorf("send chat completed: %w", err)
		}

		s.sessionMgr.Delete(sessionID)
		s.logger.Info("Chat completed during restoration",
			zap.String("session_id", sessionID),
			zap.Int("score", score),
		)
		return nil
	}

	// Небольшая задержка перед отправкой следующего вопроса
	time.Sleep(s.questionDelay)

	// Получаем следующий вопрос из программы
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
	state := s.sessionMgr.GetOrCreate(sessionID, userID)

	// Если программа не установлена, это ошибка - программа должна быть установлена при старте или восстановлении
	if state.Program == nil {
		s.logger.Error("Session program not found, this should not happen",
			zap.String("session_id", sessionID),
		)
		return fmt.Errorf("session program not found, session may need to be resumed first")
	}

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
		if err := s.resultsCRUDCl.SaveResult(ctx, sessionUUID, score, feedback, false); err != nil {
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

// HandleChatResumed обрабатывает восстановление активной сессии чата.
func (s *ModelService) HandleChatResumed(ctx context.Context, sessionID, userID string) error {
	s.logger.Info("Chat resumed, restoring session state",
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

	s.logger.Info("Interview program received for resume",
		zap.String("session_id", sessionID),
		zap.Int("questions_count", len(program.Questions)),
	)

	// Получаем историю чата для восстановления состояния
	messages, err := s.chatCRUDCl.GetMessages(ctx, sessionUUID)
	if err != nil {
		s.logger.Warn("Failed to get chat history for resume, assuming new session state",
			zap.String("session_id", sessionID),
			zap.Error(err),
		)
		// Если не удалось получить историю, считаем что это начало сессии
		s.sessionMgr.GetOrCreate(sessionID, userID)
		s.sessionMgr.SetProgram(sessionID, program)
		return nil
	}

	// Подсчитываем количество системных сообщений (вопросов) и пользовательских сообщений
	systemMessagesCount := 0
	userMessagesCount := 0
	for _, msg := range messages {
		if msg.Type == "system" {
			systemMessagesCount++
		} else if msg.Type == "user" {
			userMessagesCount++
		}
	}

	// Восстанавливаем состояние на основе истории
	s.logger.Info("Restoring session state from chat history",
		zap.String("session_id", sessionID),
		zap.Int("system_messages_count", systemMessagesCount),
		zap.Int("user_messages_count", userMessagesCount),
		zap.Int("total_messages_count", len(messages)),
	)

	s.sessionMgr.RestoreStateFromChatHistory(sessionID, program, systemMessagesCount)

	s.logger.Info("Session restored successfully",
		zap.String("session_id", sessionID),
		zap.Int("questions_asked", s.sessionMgr.GetQuestionCount(sessionID)),
	)

	// После восстановления состояния проверяем, нужно ли отправлять следующий вопрос
	// Если пользователь уже ответил на последний вопрос (userMessagesCount == systemMessagesCount),
	// но чат еще не завершен, значит нужно отправить следующий вопрос (если он есть)
	if s.sessionMgr.HasMoreQuestions(sessionID) && userMessagesCount == systemMessagesCount && len(messages) > 0 {
		// Пользователь ответил на все заданные вопросы, нужно отправить следующий
		time.Sleep(s.questionDelay)

		nextQuestion, ok := s.sessionMgr.GetNextQuestion(sessionID)
		if ok {
			// Сохраняем вопрос в chat-crud
			systemMsg := fmt.Sprintf("Вопрос: %s", nextQuestion)
			if err := s.chatCRUDCl.SaveMessage(ctx, sessionUUID, client.MessageTypeSystem, systemMsg); err != nil {
				s.logger.Warn("Failed to save system message after resume",
					zap.String("session_id", sessionID),
					zap.Error(err),
				)
			}

			// Отправляем вопрос в Kafka
			if err := s.producer.SendModelQuestion(sessionID, userID, nextQuestion); err != nil {
				s.logger.Error("Failed to send question after resume",
					zap.String("session_id", sessionID),
					zap.Error(err),
				)
			} else {
				s.sessionMgr.IncrementQuestion(sessionID)
				s.logger.Info("Next question sent after resume",
					zap.String("session_id", sessionID),
					zap.String("question", nextQuestion),
				)
			}
		}
	}

	return nil
}

// HandleChatTerminated обрабатывает досрочное завершение чата пользователем.
func (s *ModelService) HandleChatTerminated(ctx context.Context, sessionID, userID string) error {
	s.logger.Info("Chat terminated by user",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
	)

	// Парсим sessionID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session_id: %w", err)
	}

	// Сохраняем финальное сообщение о досрочном завершении
	terminationMsg := "Чат завершен пользователем."
	if err := s.chatCRUDCl.SaveMessage(ctx, sessionUUID, client.MessageTypeSystem, terminationMsg); err != nil {
		s.logger.Warn("Failed to save termination message to chat-crud",
			zap.String("session_id", sessionID),
			zap.Error(err),
		)
	}

	// Создаём дамп чата
	if err := s.chatCRUDCl.CreateChatDump(ctx, sessionUUID); err != nil {
		s.logger.Warn("Failed to create chat dump",
			zap.String("session_id", sessionID),
			zap.Error(err),
		)
	}

	// Получаем количество заданных вопросов для оценки
	questionsAsked := s.sessionMgr.GetQuestionCount(sessionID)
	score := s.generateScore(questionsAsked)
	feedback := "Интервью было досрочно завершено пользователем."

	// Сохраняем результаты с флагом terminated_early = true
	if err := s.resultsCRUDCl.SaveResult(ctx, sessionUUID, score, feedback, true); err != nil {
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

	s.logger.Info("Chat terminated successfully",
		zap.String("session_id", sessionID),
		zap.Int("score", score),
	)

	return nil
}
