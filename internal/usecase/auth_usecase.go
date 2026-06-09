package usecase

import (
	"context"
	"errors"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/repository"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/apperror"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/hash"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/token"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuthUsecase defines the business logic interface for authentication.
type AuthUsecase interface {
	Register(ctx context.Context, input domain.RegisterInput) (*domain.User, error)
	Login(ctx context.Context, input domain.LoginInput) (string, *domain.User, error)
	GetBalance(ctx context.Context, userID int64, role string) (map[string]interface{}, error)
}

type authUsecase struct {
	db           *gorm.DB
	userRepo     repository.UserRepository
	managerRepo  repository.ManagerRepository
	employeeRepo repository.EmployeeRepository
}

// NewAuthUsecase creates a new AuthUsecase.
func NewAuthUsecase(
	db *gorm.DB,
	userRepo repository.UserRepository,
	managerRepo repository.ManagerRepository,
	employeeRepo repository.EmployeeRepository,
) AuthUsecase {
	return &authUsecase{
		db:           db,
		userRepo:     userRepo,
		managerRepo:  managerRepo,
		employeeRepo: employeeRepo,
	}
}

// Register creates a new user with the given role.
// In a single transaction: creates user + manager/employee record.
func (u *authUsecase) Register(ctx context.Context, input domain.RegisterInput) (*domain.User, error) {
	// Check if email already exists
	existing, err := u.userRepo.FindByEmail(ctx, input.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.ErrInternal
	}
	if existing != nil {
		return nil, apperror.ErrConflict
	}

	// Hash password
	hashedPassword, err := hash.Hash(input.Password)
	if err != nil {
		return nil, apperror.ErrInternal
	}

	user := &domain.User{
		PublicID: uuid.New(),
		Name:     input.Name,
		Email:    input.Email,
		Password: hashedPassword,
		Role:     input.Role,
	}

	// Transaction: create user + role-specific record
	err = u.db.Transaction(func(tx *gorm.DB) error {
		userRepoTx := u.userRepo.WithTx(tx)
		if err := userRepoTx.Create(ctx, user); err != nil {
			return err
		}

		switch input.Role {
		case domain.RoleManager:
			managerRepoTx := u.managerRepo.WithTx(tx)
			manager := &domain.Manager{
				UserID:          user.ID,
				BudgetAvailable: 0,
				BudgetLocked:    0,
			}
			return managerRepoTx.Create(ctx, manager)
		case domain.RoleEmployee:
			employeeRepoTx := u.employeeRepo.WithTx(tx)
			employee := &domain.Employee{
				UserID:             user.ID,
				ReimburseAvailable: 0,
				ReimburseLocked:    0,
			}
			return employeeRepoTx.Create(ctx, employee)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Clear password before returning
	user.Password = ""
	return user, nil
}

// Login authenticates a user and returns a JWT token.
func (u *authUsecase) Login(ctx context.Context, input domain.LoginInput) (string, *domain.User, error) {
	user, err := u.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, apperror.ErrUnauthorized
		}
		return "", nil, apperror.ErrInternal
	}

	if !hash.Compare(input.Password, user.Password) {
		return "", nil, apperror.ErrUnauthorized
	}

	accessToken, err := token.GenerateToken(user.ID, user.PublicID.String(), string(user.Role))
	if err != nil {
		return "", nil, apperror.ErrInternal
	}

	user.Password = ""
	return accessToken, user, nil
}

// GetBalance returns the balance for the authenticated user based on their role.
func (u *authUsecase) GetBalance(ctx context.Context, userID int64, role string) (map[string]interface{}, error) {
	switch domain.Role(role) {
	case domain.RoleManager:
		manager, err := u.managerRepo.FindByUserID(ctx, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperror.ErrNotFound
			}
			return nil, apperror.ErrInternal
		}
		return map[string]interface{}{
			"budget_available": manager.BudgetAvailable,
			"budget_locked":    manager.BudgetLocked,
		}, nil

	case domain.RoleEmployee:
		employee, err := u.employeeRepo.FindByUserID(ctx, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperror.ErrNotFound
			}
			return nil, apperror.ErrInternal
		}
		return map[string]interface{}{
			"reimburse_available": employee.ReimburseAvailable,
			"reimburse_locked":    employee.ReimburseLocked,
		}, nil

	case domain.RoleFinance:
		return map[string]interface{}{}, nil

	default:
		return nil, apperror.ErrForbidden
	}
}
