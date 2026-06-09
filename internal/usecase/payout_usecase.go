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

type PayoutUsecase interface {
	CreatePayout(ctx context.Context, userID int64, input domain.CreatePayoutInput) (*domain.Payout, error)
	ReviewPayout(ctx context.Context, publicID string, input domain.ReviewPayoutInput) (*domain.Payout, error)
	ListMyPayouts(ctx context.Context, userID int64) ([]*domain.Payout, error)
}

type payoutUsecase struct {
	db           *gorm.DB
	payoutRepo   repository.PayoutRepository
	employeeRepo repository.EmployeeRepository
}

func NewPayoutUsecase(db *gorm.DB, payoutRepo repository.PayoutRepository, employeeRepo repository.EmployeeRepository) PayoutUsecase {
	return &payoutUsecase{db: db, payoutRepo: payoutRepo, employeeRepo: employeeRepo}
}

func (u *payoutUsecase) CreatePayout(ctx context.Context, userID int64, input domain.CreatePayoutInput) (*domain.Payout, error) {
	var payout *domain.Payout
	err := u.db.Transaction(func(tx *gorm.DB) error {
		employeeRepoTx := u.employeeRepo.WithTx(tx)
		payoutRepoTx := u.payoutRepo.WithTx(tx)
		if err := employeeRepoTx.LockForPayout(ctx, userID, input.Amount); err != nil {
			return err
		}
		payout = &domain.Payout{
			PublicID: uuid.New(), EmployeeID: userID,
			Amount: input.Amount, Status: domain.PayoutPending,
		}
		return payoutRepoTx.Create(ctx, payout)
	})
	if err != nil {
		return nil, err
	}
	return payout, nil
}

func (u *payoutUsecase) ReviewPayout(ctx context.Context, publicID string, input domain.ReviewPayoutInput) (*domain.Payout, error) {
	payout, err := u.payoutRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal
	}
	if payout.Status != domain.PayoutPending {
		return nil, apperror.ErrInvalidStatusTrans
	}
	now := time.Now()
	err = u.db.Transaction(func(tx *gorm.DB) error {
		payoutRepoTx := u.payoutRepo.WithTx(tx)
		employeeRepoTx := u.employeeRepo.WithTx(tx)
		if input.Status == domain.PayoutCompleted {
			fee := payout.Amount * 25 / 1000
			netAmount := payout.Amount - fee
			payout.Fee = &fee
			payout.NetAmount = &netAmount
			if err := employeeRepoTx.ConsumeLockedPayout(ctx, payout.EmployeeID, payout.Amount); err != nil {
				return err
			}
			return payoutRepoTx.UpdateReview(ctx, payout.ID, domain.PayoutCompleted, &fee, &netAmount, now)
		}
		// Failed: release locked back to available
		if err := employeeRepoTx.ReleaseLockedPayout(ctx, payout.EmployeeID, payout.Amount); err != nil {
			return err
		}
		return payoutRepoTx.UpdateReview(ctx, payout.ID, domain.PayoutFailed, nil, nil, now)
	})
	if err != nil {
		return nil, err
	}
	payout.Status = input.Status
	payout.ReviewedAt = &now
	return payout, nil
}

func (u *payoutUsecase) ListMyPayouts(ctx context.Context, userID int64) ([]*domain.Payout, error) {
	return u.payoutRepo.ListByEmployeeID(ctx, userID)
}
