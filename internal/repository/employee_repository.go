package repository

import (
	"context"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/apperror"
	"gorm.io/gorm"
)

// EmployeeRepository defines the data access interface for employees.
type EmployeeRepository interface {
	WithTx(tx *gorm.DB) EmployeeRepository
	Create(ctx context.Context, e *domain.Employee) error
	FindByUserID(ctx context.Context, userID int64) (*domain.Employee, error)
	IncrementAvailable(ctx context.Context, userID int64, amount int64) error
	LockForPayout(ctx context.Context, userID int64, amount int64) error
	ConsumeLockedPayout(ctx context.Context, userID int64, amount int64) error
	ReleaseLockedPayout(ctx context.Context, userID int64, amount int64) error
}

type employeeRepository struct {
	db *gorm.DB
}

// NewEmployeeRepository creates a new EmployeeRepository.
func NewEmployeeRepository(db *gorm.DB) EmployeeRepository {
	return &employeeRepository{db: db}
}

func (r *employeeRepository) WithTx(tx *gorm.DB) EmployeeRepository {
	return &employeeRepository{db: tx}
}

func (r *employeeRepository) Create(ctx context.Context, e *domain.Employee) error {
	return r.db.WithContext(ctx).
		Table("employees").
		Create(e).Error
}

func (r *employeeRepository) FindByUserID(ctx context.Context, userID int64) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.db.WithContext(ctx).
		Table("employees").
		Where("user_id = ?", userID).
		First(&employee).Error
	if err != nil {
		return nil, err
	}
	return &employee, nil
}

// IncrementAvailable atomically adds amount to reimburse_available (when claim is approved).
func (r *employeeRepository) IncrementAvailable(ctx context.Context, userID int64, amount int64) error {
	result := r.db.WithContext(ctx).
		Table("employees").
		Where("user_id = ?", userID).
		Update("reimburse_available", gorm.Expr("reimburse_available + ?", amount))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

// LockForPayout atomically moves amount from reimburse_available to reimburse_locked.
// Returns ErrInsufficientBalance if reimburse_available < amount.
func (r *employeeRepository) LockForPayout(ctx context.Context, userID int64, amount int64) error {
	result := r.db.WithContext(ctx).
		Table("employees").
		Where("user_id = ? AND reimburse_available >= ?", userID, amount).
		Updates(map[string]interface{}{
			"reimburse_available": gorm.Expr("reimburse_available - ?", amount),
			"reimburse_locked":    gorm.Expr("reimburse_locked + ?", amount),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.ErrInsufficientBalance
	}
	return nil
}

// ConsumeLockedPayout atomically decrements reimburse_locked when payout is completed.
func (r *employeeRepository) ConsumeLockedPayout(ctx context.Context, userID int64, amount int64) error {
	result := r.db.WithContext(ctx).
		Table("employees").
		Where("user_id = ? AND reimburse_locked >= ?", userID, amount).
		Update("reimburse_locked", gorm.Expr("reimburse_locked - ?", amount))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.ErrInternal
	}
	return nil
}

// ReleaseLockedPayout atomically moves amount from reimburse_locked back to reimburse_available
// when a payout fails.
func (r *employeeRepository) ReleaseLockedPayout(ctx context.Context, userID int64, amount int64) error {
	result := r.db.WithContext(ctx).
		Table("employees").
		Where("user_id = ? AND reimburse_locked >= ?", userID, amount).
		Updates(map[string]interface{}{
			"reimburse_locked":    gorm.Expr("reimburse_locked - ?", amount),
			"reimburse_available": gorm.Expr("reimburse_available + ?", amount),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.ErrInternal
	}
	return nil
}
