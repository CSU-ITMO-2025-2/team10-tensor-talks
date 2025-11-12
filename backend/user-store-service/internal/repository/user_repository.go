package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/tensor-talks/user-store-service/internal/models"
	"gorm.io/gorm"
)

var (
	// ErrNotFound indicates no user matched the query.
	ErrNotFound = errors.New("user not found")
	// ErrDuplicateLogin is returned when login already exists.
	ErrDuplicateLogin = errors.New("login already exists")
)

// UserRepository provides access to stored users.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByExternalID(ctx context.Context, externalID uuid.UUID) (*models.User, error)
	GetByLogin(ctx context.Context, login string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, externalID uuid.UUID) error
}

// GormUserRepository is a GORM-backed repository.
type GormUserRepository struct {
	db *gorm.DB
}

// NewGormUserRepository constructs a repository instance.
func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

// Create inserts a new user record.
func (r *GormUserRepository) Create(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return mapPGError(err)
	}
	return nil
}

// GetByExternalID fetches a user by GUID.
func (r *GormUserRepository) GetByExternalID(ctx context.Context, externalID uuid.UUID) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("external_id = ?", externalID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByLogin fetches a user by login.
func (r *GormUserRepository) GetByLogin(ctx context.Context, login string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("login = ?", login).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// Update persists user changes.
func (r *GormUserRepository) Update(ctx context.Context, user *models.User) error {
	result := r.db.WithContext(ctx).Model(&models.User{}).Where("external_id = ?", user.ExternalID).Updates(map[string]any{
		"login":         user.Login,
		"password_hash": user.PasswordHash,
	})
	if err := mapPGError(result.Error); err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a user by external ID.
func (r *GormUserRepository) Delete(ctx context.Context, externalID uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("external_id = ?", externalID).Delete(&models.User{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func mapPGError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrDuplicateLogin
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicateLogin
	}
	return err
}
