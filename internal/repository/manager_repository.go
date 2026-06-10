package repository

import (
	"context"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/apperror"
	"gorm.io/gorm"
)

// ManagerRepository defines the data access interface for managers.
type ManagerRepository interface {
	WithTx(tx *gorm.DB) ManagerRepository
	Create(ctx context.Context, m *domain.Manager) error
	FindByUserID(ctx context.Context, userID int64) (*domain.Manager, error)
	IncrementAvailable(ctx context.Context, userID int64, amount int64) error
	LockEnvelope(ctx context.Context, userID int64, amount int64) error
	DecrementLocked(ctx context.Context, userID int64, amount int64) error
}

type managerRepository struct {
	db *gorm.DB
}

// NewManagerRepository creates a new ManagerRepository.
func NewManagerRepository(db *gorm.DB) ManagerRepository {
	return &managerRepository{db: db}
}

func (r *managerRepository) WithTx(tx *gorm.DB) ManagerRepository {
	return &managerRepository{db: tx}
}

func (r *managerRepository) Create(ctx context.Context, m *domain.Manager) error {
	return r.db.WithContext(ctx).
		Table("managers").
		Create(m).Error
}

func (r *managerRepository) FindByUserID(ctx context.Context, userID int64) (*domain.Manager, error) {
	var manager domain.Manager
	err := r.db.WithContext(ctx).
		Table("managers").
		Where("user_id = ?", userID).
		First(&manager).Error
	if err != nil {
		return nil, err
	}
	return &manager, nil
}

// IncrementAvailable atomically adds amount to budget_available.
func (r *managerRepository) IncrementAvailable(ctx context.Context, userID int64, amount int64) error {
	result := r.db.WithContext(ctx).
		Table("managers").
		Where("user_id = ?", userID).
		Update("budget_available", gorm.Expr("budget_available + ?", amount))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

// LockEnvelope atomically moves amount from budget_available to budget_locked.
// Returns ErrInsufficientBudget if budget_available < amount.
func (r *managerRepository) LockEnvelope(ctx context.Context, userID int64, amount int64) error {
	result := r.db.WithContext(ctx).
		Table("managers").
		Where("user_id = ? AND budget_available >= ?", userID, amount).
		Updates(map[string]interface{}{
			"budget_available": gorm.Expr("budget_available - ?", amount),
			"budget_locked":    gorm.Expr("budget_locked + ?", amount),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.ErrInsufficientBudget
	}
	return nil
}

// DecrementLocked atomically decrements budget_locked when a claim is approved.
func (r *managerRepository) DecrementLocked(ctx context.Context, userID int64, amount int64) error {
	result := r.db.WithContext(ctx).
		Table("managers").
		Where("user_id = ? AND budget_locked >= ?", userID, amount).
		Update("budget_locked", gorm.Expr("budget_locked - ?", amount))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.ErrInternal
	}
	return nil
}
