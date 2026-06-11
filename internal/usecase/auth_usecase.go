package usecase

import (
	"context"
	"errors"
	"time"

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
	Login(ctx context.Context, input domain.LoginInput) (string, string, *domain.User, error)
	RefreshToken(ctx context.Context, input domain.RefreshInput) (string, string, error)
	GetBalance(ctx context.Context, userID int64, role string) (map[string]interface{}, error)
}

type authUsecase struct {
	db           *gorm.DB
	userRepo     repository.UserRepository
	managerRepo  repository.ManagerRepository
	employeeRepo repository.EmployeeRepository
	sessionRepo  repository.SessionRepository
}

// NewAuthUsecase creates a new AuthUsecase.
func NewAuthUsecase(
	db *gorm.DB,
	userRepo repository.UserRepository,
	managerRepo repository.ManagerRepository,
	employeeRepo repository.EmployeeRepository,
	sessionRepo repository.SessionRepository,
) AuthUsecase {
	return &authUsecase{
		db:           db,
		userRepo:     userRepo,
		managerRepo:  managerRepo,
		employeeRepo: employeeRepo,
		sessionRepo:  sessionRepo,
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
func (u *authUsecase) Login(ctx context.Context, input domain.LoginInput) (string, string, *domain.User, error) {
	user, err := u.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", nil, apperror.ErrUnauthorized
		}
		return "", "", nil, apperror.ErrInternal
	}

	if !hash.Compare(input.Password, user.Password) {
		return "", "", nil, apperror.ErrUnauthorized
	}

	accessToken, refreshToken, err := token.GenerateTokens(user.ID, user.PublicID.String(), string(user.Role))
	if err != nil {
		return "", "", nil, apperror.ErrInternal
	}

	// Store session
	session := &domain.Session{
		ID:           uuid.New(),
		UserID:       user.ID,
		RefreshToken: refreshToken,
		IsBlocked:    false,
		ExpiresAt:    time.Now().Add(168 * time.Hour), // 7 days (should ideally come from env)
	}
	if err := u.sessionRepo.CreateSession(ctx, session); err != nil {
		return "", "", nil, apperror.ErrInternal
	}

	user.Password = ""
	return accessToken, refreshToken, user, nil
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

// RefreshToken validates a refresh token, rotates it, and returns a new token pair.
func (u *authUsecase) RefreshToken(ctx context.Context, input domain.RefreshInput) (string, string, error) {
	// Find session by refresh token
	session, err := u.sessionRepo.GetSessionByRefreshToken(ctx, input.RefreshToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", apperror.ErrUnauthorized // Invalid token
		}
		return "", "", apperror.ErrInternal
	}

	// Check if session is blocked (Token Rotation Security mechanism)
	// If a blocked session's token is used, someone is trying to reuse an old token.
	if session.IsBlocked {
		return "", "", apperror.ErrUnauthorized
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		return "", "", apperror.ErrUnauthorized
	}

	// Get user
	user, err := u.userRepo.FindByID(ctx, session.UserID)
	if err != nil {
		return "", "", apperror.ErrInternal
	}

	// Generate new tokens
	accessToken, newRefreshToken, err := token.GenerateTokens(user.ID, user.PublicID.String(), string(user.Role))
	if err != nil {
		return "", "", apperror.ErrInternal
	}

	// Rotate token securely using a transaction
	err = u.db.Transaction(func(tx *gorm.DB) error {
		sessionRepoTx := u.sessionRepo.WithTx(tx)

		// Block the old session
		if err := sessionRepoTx.BlockSession(ctx, session.ID.String()); err != nil {
			return err
		}

		// Create new session
		newSession := &domain.Session{
			ID:           uuid.New(),
			UserID:       user.ID,
			RefreshToken: newRefreshToken,
			IsBlocked:    false,
			ExpiresAt:    time.Now().Add(168 * time.Hour),
		}
		return sessionRepoTx.CreateSession(ctx, newSession)
	})

	if err != nil {
		return "", "", apperror.ErrInternal
	}

	return accessToken, newRefreshToken, nil
}
