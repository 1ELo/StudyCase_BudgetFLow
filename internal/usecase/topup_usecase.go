package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/repository"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TopupUsecase defines the business logic interface for budget topups.
type TopupUsecase interface {
	CreateTopup(ctx context.Context, userID int64, input domain.TopupInput) (*domain.BudgetTopup, error)
	ReviewTopup(ctx context.Context, publicID string, input domain.ReviewTopupInput) (*domain.BudgetTopup, error)
	ListMyTopups(ctx context.Context, userID int64) ([]*domain.BudgetTopup, error)
}

type topupUsecase struct {
	db          *gorm.DB
	topupRepo   repository.TopupRepository
	managerRepo repository.ManagerRepository
}

// NewTopupUsecase creates a new TopupUsecase.
func NewTopupUsecase(
	db *gorm.DB,
	topupRepo repository.TopupRepository,
	managerRepo repository.ManagerRepository,
) TopupUsecase {
	return &topupUsecase{
		db:          db,
		topupRepo:   topupRepo,
		managerRepo: managerRepo,
	}
}

// CreateTopup creates a new budget topup request (manager only).
func (u *topupUsecase) CreateTopup(ctx context.Context, userID int64, input domain.TopupInput) (*domain.BudgetTopup, error) {
	// Verify manager exists
	_, err := u.managerRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal
	}

	topup := &domain.BudgetTopup{
		PublicID:  uuid.New(),
		ManagerID: userID,
		Amount:    input.Amount,
		Status:    domain.TopupPending,
	}

	if err := u.topupRepo.Create(ctx, topup); err != nil {
		return nil, apperror.ErrInternal
	}

	return topup, nil
}

// ReviewTopup reviews a budget topup request (finance only).
// If approved, increments the manager's budget_available within a transaction.
func (u *topupUsecase) ReviewTopup(ctx context.Context, publicID string, input domain.ReviewTopupInput) (*domain.BudgetTopup, error) {
	topup, err := u.topupRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal
	}

	// Guard: only pending topups can be reviewed
	if topup.Status != domain.TopupPending {
		return nil, apperror.ErrInvalidStatusTrans
	}

	now := time.Now()

	err = u.db.Transaction(func(tx *gorm.DB) error {
		topupRepoTx := u.topupRepo.WithTx(tx)

		rowsAffected, err := topupRepoTx.UpdateStatus(ctx, topup.ID, input.Status, now)
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return apperror.ErrInvalidStatusTrans
		}

		// If approved, increment manager's available budget
		if input.Status == domain.TopupApproved {
			managerRepoTx := u.managerRepo.WithTx(tx)
			if err := managerRepoTx.IncrementAvailable(ctx, topup.ManagerID, topup.Amount); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	topup.Status = input.Status
	topup.ReviewedAt = &now
	return topup, nil
}

// ListMyTopups returns all topups for the authenticated manager.
func (u *topupUsecase) ListMyTopups(ctx context.Context, userID int64) ([]*domain.BudgetTopup, error) {
	return u.topupRepo.ListByManagerID(ctx, userID)
}
