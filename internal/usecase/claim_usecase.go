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

// ClaimUsecase defines the business logic interface for expense claims.
type ClaimUsecase interface {
	CreateClaim(ctx context.Context, employeeID int64, projectPublicID string, input domain.CreateClaimInput) (*domain.ExpenseClaim, error)
	ReviewClaim(ctx context.Context, callerUserID int64, callerRole string, claimPublicID string, input domain.ReviewClaimInput) (*domain.ExpenseClaim, error)
	ListMyClaims(ctx context.Context, employeeID int64) ([]*domain.ExpenseClaim, error)
}

type claimUsecase struct {
	db           *gorm.DB
	claimRepo    repository.ClaimRepository
	projectRepo  repository.ProjectRepository
	managerRepo  repository.ManagerRepository
	employeeRepo repository.EmployeeRepository
}

// NewClaimUsecase creates a new ClaimUsecase.
func NewClaimUsecase(
	db *gorm.DB,
	claimRepo repository.ClaimRepository,
	projectRepo repository.ProjectRepository,
	managerRepo repository.ManagerRepository,
	employeeRepo repository.EmployeeRepository,
) ClaimUsecase {
	return &claimUsecase{
		db:           db,
		claimRepo:    claimRepo,
		projectRepo:  projectRepo,
		managerRepo:  managerRepo,
		employeeRepo: employeeRepo,
	}
}

// CreateClaim creates a new expense claim for an employee on a project.
func (u *claimUsecase) CreateClaim(ctx context.Context, employeeID int64, projectPublicID string, input domain.CreateClaimInput) (*domain.ExpenseClaim, error) {
	// Find project
	project, err := u.projectRepo.FindByPublicID(ctx, projectPublicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal
	}

	// Check project is open
	if project.Status != domain.ProjectOpen {
		return nil, apperror.ErrProjectClosed
	}

	// Check no existing claim for this project + employee
	exists, err := u.claimRepo.ExistsByProjectAndEmployee(ctx, project.ID, employeeID)
	if err != nil {
		return nil, apperror.ErrInternal
	}
	if exists {
		return nil, apperror.ErrDuplicateClaim
	}

	claim := &domain.ExpenseClaim{
		PublicID:   uuid.New(),
		ProjectID:  project.ID,
		EmployeeID: employeeID,
		ReceiptURL: input.ReceiptURL,
		Status:     domain.ClaimPending,
	}

	if err := u.claimRepo.Create(ctx, claim); err != nil {
		return nil, apperror.ErrInternal
	}

	return claim, nil
}

// ReviewClaim reviews an expense claim (finance or owning manager).
// If approved: decrements envelope, decrements manager locked, increments employee available — all in TX.
func (u *claimUsecase) ReviewClaim(ctx context.Context, callerUserID int64, callerRole string, claimPublicID string, input domain.ReviewClaimInput) (*domain.ExpenseClaim, error) {
	claim, err := u.claimRepo.FindByPublicID(ctx, claimPublicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal
	}

	// Guard: status must be pending
	if claim.Status != domain.ClaimPending {
		return nil, apperror.ErrInvalidStatusTrans
	}

	// Get project for claim_amount and IDOR check
	project, err := u.projectRepo.FindByID(ctx, claim.ProjectID)
	if err != nil {
		return nil, apperror.ErrInternal
	}

	// IDOR check: if caller is manager, must be the project owner
	if domain.Role(callerRole) == domain.RoleManager && project.ManagerID != callerUserID {
		return nil, apperror.ErrForbidden
	}

	now := time.Now()

	if input.Status == domain.ClaimRejected {
		// Simple rejection — just update status
		if err := u.claimRepo.UpdateStatus(ctx, claim.ID, domain.ClaimRejected, now); err != nil {
			return nil, apperror.ErrInternal
		}
		claim.Status = domain.ClaimRejected
		claim.ReviewedAt = &now
		return claim, nil
	}

	// Approval — multi-table transaction
	err = u.db.Transaction(func(tx *gorm.DB) error {
		claimRepoTx := u.claimRepo.WithTx(tx)
		projectRepoTx := u.projectRepo.WithTx(tx)
		managerRepoTx := u.managerRepo.WithTx(tx)
		employeeRepoTx := u.employeeRepo.WithTx(tx)

		// Decrement project envelope (anti-double-spend)
		if err := projectRepoTx.DecrementEnvelope(ctx, project.ID, project.ClaimAmount); err != nil {
			return err
		}

		// Decrement manager's locked budget
		if err := managerRepoTx.DecrementLocked(ctx, project.ManagerID, project.ClaimAmount); err != nil {
			return err
		}

		// Credit employee's available balance
		if err := employeeRepoTx.IncrementAvailable(ctx, claim.EmployeeID, project.ClaimAmount); err != nil {
			return err
		}

		// Update claim status
		if err := claimRepoTx.UpdateStatus(ctx, claim.ID, domain.ClaimApproved, now); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	claim.Status = domain.ClaimApproved
	claim.ReviewedAt = &now
	return claim, nil
}

// ListMyClaims returns all claims for the authenticated employee.
func (u *claimUsecase) ListMyClaims(ctx context.Context, employeeID int64) ([]*domain.ExpenseClaim, error) {
	return u.claimRepo.ListByEmployeeID(ctx, employeeID)
}
