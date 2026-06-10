package usecase

import (
	"context"
	"errors"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/repository"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/apperror"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/querybuilder"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProjectUsecase defines the business logic interface for projects.
type ProjectUsecase interface {
	CreateProject(ctx context.Context, userID int64, input domain.CreateProjectInput) (*domain.Project, error)
	ListProjects(ctx context.Context, query domain.ProjectListQuery) ([]*domain.Project, int64, error)
	GetProject(ctx context.Context, publicID string) (*domain.Project, error)
	DeleteProject(ctx context.Context, userID int64, publicID string) error
	RestoreProject(ctx context.Context, userID int64, publicID string) error
}

type projectUsecase struct {
	db          *gorm.DB
	projectRepo repository.ProjectRepository
	managerRepo repository.ManagerRepository
	userRepo    repository.UserRepository
}

// NewProjectUsecase creates a new ProjectUsecase.
func NewProjectUsecase(
	db *gorm.DB,
	projectRepo repository.ProjectRepository,
	managerRepo repository.ManagerRepository,
	userRepo repository.UserRepository,
) ProjectUsecase {
	return &projectUsecase{
		db:          db,
		projectRepo: projectRepo,
		managerRepo: managerRepo,
		userRepo:    userRepo,
	}
}

// CreateProject creates a new project and locks the manager's budget envelope.
func (u *projectUsecase) CreateProject(ctx context.Context, userID int64, input domain.CreateProjectInput) (*domain.Project, error) {
	// Validate: claim_amount <= envelope_total
	if input.ClaimAmount > input.EnvelopeTotal {
		return nil, apperror.NewAppError(400, "VALIDATION_ERROR", "claim_amount must be less than or equal to envelope_total")
	}

	var project *domain.Project

	err := u.db.Transaction(func(tx *gorm.DB) error {
		managerRepoTx := u.managerRepo.WithTx(tx)
		projectRepoTx := u.projectRepo.WithTx(tx)

		// Lock envelope from manager's available budget
		if err := managerRepoTx.LockEnvelope(ctx, userID, input.EnvelopeTotal); err != nil {
			return err // ErrInsufficientBudget if not enough
		}

		project = &domain.Project{
			PublicID:          uuid.New(),
			ManagerID:         userID,
			Title:             input.Title,
			ClaimAmount:       input.ClaimAmount,
			EnvelopeTotal:     input.EnvelopeTotal,
			EnvelopeRemaining: input.EnvelopeTotal,
			Status:            domain.ProjectOpen,
		}

		return projectRepoTx.Create(ctx, project)
	})
	if err != nil {
		return nil, err
	}

	return project, nil
}

// ListProjects returns a paginated, filtered list of projects.
func (u *projectUsecase) ListProjects(ctx context.Context, query domain.ProjectListQuery) ([]*domain.Project, int64, error) {
	// Parse and validate query params
	query, err := querybuilder.BuildProjectFilter(query)
	if err != nil {
		return nil, 0, apperror.NewAppError(400, "VALIDATION_ERROR", err.Error())
	}

	sortColumn, sortDesc, err := querybuilder.ParseSort(query.Sort)
	if err != nil {
		return nil, 0, apperror.NewAppError(400, "VALIDATION_ERROR", err.Error())
	}

	filter := querybuilder.ProjectFilter{
		SortColumn: sortColumn,
		SortDesc:   sortDesc,
		Page:       query.Page,
		Limit:      query.Limit,
	}

	if query.Status != "" {
		filter.Status = &query.Status
	}
	if query.Search != "" {
		filter.Search = &query.Search
	}

	// Resolve manager_public_id to internal manager_id
	if query.ManagerPublicID != "" {
		user, err := u.userRepo.FindByPublicID(ctx, query.ManagerPublicID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Return empty result if manager not found
				return []*domain.Project{}, 0, nil
			}
			return nil, 0, apperror.ErrInternal
		}
		filter.ManagerID = &user.ID
	}

	return u.projectRepo.List(ctx, filter)
}

// GetProject returns a project by its public ID.
func (u *projectUsecase) GetProject(ctx context.Context, publicID string) (*domain.Project, error) {
	project, err := u.projectRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, apperror.ErrInternal
	}
	return project, nil
}

// DeleteProject soft deletes a project if the user is the owner.
func (u *projectUsecase) DeleteProject(ctx context.Context, userID int64, publicID string) error {
	project, err := u.projectRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.ErrNotFound
		}
		return apperror.ErrInternal
	}

	if project.ManagerID != userID {
		return apperror.ErrForbidden // Only the owner can delete
	}

	if err := u.projectRepo.Delete(ctx, publicID); err != nil {
		return apperror.ErrInternal
	}

	return nil
}

// RestoreProject restores a soft-deleted project.
func (u *projectUsecase) RestoreProject(ctx context.Context, userID int64, publicID string) error {
	var project domain.Project
	err := u.db.WithContext(ctx).Unscoped().Table("projects").Where("public_id = ?", publicID).First(&project).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.ErrNotFound
		}
		return apperror.ErrInternal
	}

	if project.ManagerID != userID {
		return apperror.ErrForbidden
	}

	if err := u.projectRepo.Restore(ctx, publicID); err != nil {
		return apperror.ErrInternal
	}

	return nil
}
