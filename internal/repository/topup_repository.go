package repository

import (
	"context"
	"time"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"gorm.io/gorm"
)

// TopupRepository defines the data access interface for budget topups.
type TopupRepository interface {
	WithTx(tx *gorm.DB) TopupRepository
	Create(ctx context.Context, t *domain.BudgetTopup) error
	FindByPublicID(ctx context.Context, publicID string) (*domain.BudgetTopup, error)
	UpdateStatus(ctx context.Context, id int64, status domain.TopupStatus, reviewedAt time.Time) (int64, error)
	ListByManagerID(ctx context.Context, managerID int64) ([]*domain.BudgetTopup, error)
}

type topupRepository struct {
	db *gorm.DB
}

// NewTopupRepository creates a new TopupRepository.
func NewTopupRepository(db *gorm.DB) TopupRepository {
	return &topupRepository{db: db}
}

func (r *topupRepository) WithTx(tx *gorm.DB) TopupRepository {
	return &topupRepository{db: tx}
}

func (r *topupRepository) Create(ctx context.Context, t *domain.BudgetTopup) error {
	return r.db.WithContext(ctx).
		Table("budget_topups").
		Create(t).Error
}

func (r *topupRepository) FindByPublicID(ctx context.Context, publicID string) (*domain.BudgetTopup, error) {
	var topup domain.BudgetTopup
	err := r.db.WithContext(ctx).
		Table("budget_topups").
		Where("public_id = ?", publicID).
		First(&topup).Error
	if err != nil {
		return nil, err
	}
	return &topup, nil
}

// UpdateStatus updates the status of a topup safely.
// Returns rows affected to ensure idempotent updates.
func (r *topupRepository) UpdateStatus(ctx context.Context, id int64, status domain.TopupStatus, reviewedAt time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Table("budget_topups").
		Where("id = ? AND status = ?", id, domain.TopupPending). // Idempotent guard
		Updates(map[string]interface{}{
			"status":      status,
			"reviewed_at": reviewedAt,
			"updated_at":  time.Now(),
		})
	return result.RowsAffected, result.Error
}

// ListByManagerID returns all topup requests for a specific manager.
func (r *topupRepository) ListByManagerID(ctx context.Context, managerID int64) ([]*domain.BudgetTopup, error) {
	var topups []*domain.BudgetTopup
	err := r.db.WithContext(ctx).
		Table("budget_topups").
		Where("manager_id = ?", managerID).
		Order("created_at DESC").
		Find(&topups).Error
	return topups, err
}
