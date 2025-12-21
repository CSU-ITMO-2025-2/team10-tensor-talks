package models

import (
	"time"

	"github.com/google/uuid"
)

// Result представляет результат интервью.
type Result struct {
	ID        uint      `gorm:"primaryKey"`
	SessionID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	Score     int       `gorm:"not null"`
	Feedback  string    `gorm:"type:text;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName указывает имя таблицы в БД.
func (Result) TableName() string {
	return "results"
}
