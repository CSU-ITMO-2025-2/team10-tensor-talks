package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User describes a registered account stored in PostgreSQL.
type User struct {
	ID           uint      `gorm:"primaryKey"`
	ExternalID   uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	Login        string    `gorm:"type:varchar(64);uniqueIndex;not null"`
	PasswordHash string    `gorm:"type:text;not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// BeforeCreate fills in defaults where needed.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ExternalID == uuid.Nil {
		u.ExternalID = uuid.New()
	}
	return nil
}

// PublicUser is a sanitized representation safe for returning via APIs.
type PublicUser struct {
	ExternalID   uuid.UUID `json:"id"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ToPublic converts database entity to an API friendly struct.
func (u User) ToPublic() PublicUser {
	return PublicUser{
		ExternalID:   u.ExternalID,
		Login:        u.Login,
		PasswordHash: u.PasswordHash,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}
