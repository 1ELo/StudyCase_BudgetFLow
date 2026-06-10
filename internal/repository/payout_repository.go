package repository

import (
	"context"
	"time"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"gorm.io/gorm"
)

// PayoutRepository defines the data access interface for payouts.
type PayoutRepository interface {
	WithTx(tx *gorm.DB) PayoutRepository
	Create(ctx context.Context, p *domain.Payout) error
	FindByPublicID(ctx context.Context, publicID string) (*domain.Payout, error)
	UpdateReview(ctx context.Context, id int64, status domain.PayoutStatus, fee, netAmount *int64, reviewedAt time.Time) error
	ListByEmployeeID(ctx context.Context, employeeID int64) ([]*domain.Payout, error)
}

type payoutRepository struct {
	db *gorm.DB
}

// NewPayoutRepository creates a new PayoutRepository.
func NewPayoutRepository(db *gorm.DB) PayoutRepository {
	return &payoutRepository{db: db}
}

func (r *payoutRepository) WithTx(tx *gorm.DB) PayoutRepository {
	return &payoutRepository{db: tx}
}

func (r *payoutRepository) Create(ctx context.Context, p *domain.Payout) error {
	return r.db.WithContext(ctx).
		Table("payouts").
		Create(p).Error
}

func (r *payoutRepository) FindByPublicID(ctx context.Context, publicID string) (*domain.Payout, error) {
	var payout domain.Payout
	err := r.db.WithContext(ctx).
		Table("payouts").
		Where("public_id = ?", publicID).
		First(&payout).Error
	if err != nil {
		return nil, err
	}
	return &payout, nil
}

// UpdateReview updates the payout status, fee, net amount, and review time.
func (r *payoutRepository) UpdateReview(ctx context.Context, id int64, status domain.PayoutStatus, fee, netAmount *int64, reviewedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Table("payouts").
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      status,
			"fee":         fee,
			"net_amount":  netAmount,
			"reviewed_at": reviewedAt,
			"updated_at":  time.Now(),
		})
	return result.Error
}

// ListByEmployeeID returns all payouts for an employee.
func (r *payoutRepository) ListByEmployeeID(ctx context.Context, employeeID int64) ([]*domain.Payout, error) {
	var payouts []*domain.Payout
	err := r.db.WithContext(ctx).
		Table("payouts").
		Where("employee_id = ?", employeeID).
		Order("created_at DESC").
		Find(&payouts).Error
	return payouts, err
}
