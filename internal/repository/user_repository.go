package repository

import (
	"context"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"gorm.io/gorm"
)

// UserRepository defines the data access interface for users.
type UserRepository interface {
	WithTx(tx *gorm.DB) UserRepository
	Create(ctx context.Context, u *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByPublicID(ctx context.Context, publicID string) (*domain.User, error)
	FindByID(ctx context.Context, id int64) (*domain.User, error)
}

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// WithTx returns a new repository instance bound to the provided transaction.
func (r *userRepository) WithTx(tx *gorm.DB) UserRepository {
	return &userRepository{db: tx}
}

func (r *userRepository) Create(ctx context.Context, u *domain.User) error {
	return r.db.WithContext(ctx).
		Table("users").
		Create(u).Error
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).
		Table("users").
		Where("email = ? AND deleted_at IS NULL", email).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByPublicID(ctx context.Context, publicID string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).
		Table("users").
		Where("public_id = ? AND deleted_at IS NULL", publicID).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).
		Table("users").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
