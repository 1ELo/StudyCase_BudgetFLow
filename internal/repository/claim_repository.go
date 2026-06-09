package repository

import (
	"context"
	"time"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"gorm.io/gorm"
)

// ClaimRepository defines the data access interface for expense claims.
type ClaimRepository interface {
	WithTx(tx *gorm.DB) ClaimRepository
	Create(ctx context.Context, c *domain.ExpenseClaim) error
	FindByPublicID(ctx context.Context, publicID string) (*domain.ExpenseClaim, error)
	UpdateStatus(ctx context.Context, id int64, status domain.ClaimStatus, reviewedAt time.Time) error
	ExistsByProjectAndEmployee(ctx context.Context, projectID, employeeID int64) (bool, error)
	ListByEmployeeID(ctx context.Context, employeeID int64) ([]*domain.ExpenseClaim, error)
}

type claimRepository struct {
	db *gorm.DB
}

// NewClaimRepository creates a new ClaimRepository.
func NewClaimRepository(db *gorm.DB) ClaimRepository {
	return &claimRepository{db: db}
}

func (r *claimRepository) WithTx(tx *gorm.DB) ClaimRepository {
	return &claimRepository{db: tx}
}

func (r *claimRepository) Create(ctx context.Context, c *domain.ExpenseClaim) error {
	return r.db.WithContext(ctx).
		Table("expense_claims").
		Create(c).Error
}

func (r *claimRepository) FindByPublicID(ctx context.Context, publicID string) (*domain.ExpenseClaim, error) {
	var claim domain.ExpenseClaim
	err := r.db.WithContext(ctx).
		Table("expense_claims").
		Where("public_id = ?", publicID).
		First(&claim).Error
	if err != nil {
		return nil, err
	}
	return &claim, nil
}

// UpdateStatus updates the claim status and reviewed_at timestamp.
func (r *claimRepository) UpdateStatus(ctx context.Context, id int64, status domain.ClaimStatus, reviewedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Table("expense_claims").
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      status,
			"reviewed_at": reviewedAt,
			"updated_at":  time.Now(),
		})
	return result.Error
}

// ExistsByProjectAndEmployee checks if a claim exists for the given project and employee.
func (r *claimRepository) ExistsByProjectAndEmployee(ctx context.Context, projectID, employeeID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("expense_claims").
		Where("project_id = ? AND employee_id = ?", projectID, employeeID).
		Count(&count).Error
	return count > 0, err
}

// ListByEmployeeID returns all claims for a given employee.
func (r *claimRepository) ListByEmployeeID(ctx context.Context, employeeID int64) ([]*domain.ExpenseClaim, error) {
	var claims []*domain.ExpenseClaim
	err := r.db.WithContext(ctx).
		Table("expense_claims").
		Where("employee_id = ?", employeeID).
		Order("created_at DESC").
		Find(&claims).Error
	return claims, err
}
