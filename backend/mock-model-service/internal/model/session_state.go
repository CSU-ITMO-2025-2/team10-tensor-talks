package model

import (
	"sync"
	"time"
)

// SessionState хранит состояние сессии чата.
type SessionState struct {
	SessionID      string
	UserID         string
	QuestionsAsked int
	LastQuestionAt time.Time
	StartedAt      time.Time
	mu             sync.RWMutex
}

// SessionManager управляет состоянием сессий.
type SessionManager struct {
	sessions sync.Map // map[string]*SessionState
}

// NewSessionManager создаёт новый менеджер сессий.
func NewSessionManager() *SessionManager {
	return &SessionManager{}
}

// GetOrCreate получает или создаёт состояние сессии.
func (sm *SessionManager) GetOrCreate(sessionID, userID string) *SessionState {
	if state, ok := sm.sessions.Load(sessionID); ok {
		return state.(*SessionState)
	}

	newState := &SessionState{
		SessionID:      sessionID,
		UserID:         userID,
		QuestionsAsked: 0,
		StartedAt:      time.Now(),
		LastQuestionAt: time.Time{},
	}

	sm.sessions.Store(sessionID, newState)
	return newState
}

// IncrementQuestion увеличивает счётчик вопросов.
func (sm *SessionManager) IncrementQuestion(sessionID string) {
	if state, ok := sm.sessions.Load(sessionID); ok {
		s := state.(*SessionState)
		s.mu.Lock()
		s.QuestionsAsked++
		s.LastQuestionAt = time.Now()
		s.mu.Unlock()
	}
}

// GetQuestionCount возвращает количество заданных вопросов.
func (sm *SessionManager) GetQuestionCount(sessionID string) int {
	if state, ok := sm.sessions.Load(sessionID); ok {
		s := state.(*SessionState)
		s.mu.RLock()
		defer s.mu.RUnlock()
		return s.QuestionsAsked
	}
	return 0
}

// Delete удаляет сессию.
func (sm *SessionManager) Delete(sessionID string) {
	sm.sessions.Delete(sessionID)
}

// ShouldComplete проверяет, нужно ли завершить чат.
func (sm *SessionManager) ShouldComplete(sessionID string, maxQuestions int) bool {
	return sm.GetQuestionCount(sessionID) >= maxQuestions
}
