package repository

import (
	"context"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/apperror"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/querybuilder"
	"gorm.io/gorm"
)

// ProjectRepository defines the data access interface for projects.
type ProjectRepository interface {
	WithTx(tx *gorm.DB) ProjectRepository
	Create(ctx context.Context, p *domain.Project) error
	FindByPublicID(ctx context.Context, publicID string) (*domain.Project, error)
	FindByPublicIDUnscoped(ctx context.Context, publicID string) (*domain.Project, error)
	FindByID(ctx context.Context, id int64) (*domain.Project, error)
	List(ctx context.Context, filter querybuilder.ProjectFilter) ([]*domain.Project, int64, error)
	DecrementEnvelope(ctx context.Context, projectID int64, amount int64) error
	Delete(ctx context.Context, publicID string) error
	Restore(ctx context.Context, publicID string) error
}

type projectRepository struct {
	db *gorm.DB
}

// NewProjectRepository creates a new ProjectRepository.
func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{db: db}
}

func (r *projectRepository) WithTx(tx *gorm.DB) ProjectRepository {
	return &projectRepository{db: tx}
}

func (r *projectRepository) Create(ctx context.Context, p *domain.Project) error {
	return r.db.WithContext(ctx).
		Table("projects").
		Create(p).Error
}

func (r *projectRepository) FindByPublicID(ctx context.Context, publicID string) (*domain.Project, error) {
	var project domain.Project
	err := r.db.WithContext(ctx).
		Table("projects").
		Where("public_id = ? AND deleted_at IS NULL", publicID).
		First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *projectRepository) FindByPublicIDUnscoped(ctx context.Context, publicID string) (*domain.Project, error) {
	var project domain.Project
	err := r.db.WithContext(ctx).
		Unscoped().
		Table("projects").
		Where("public_id = ?", publicID).
		First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// FindByID is used internally where we already know the internal ID.
func (r *projectRepository) FindByID(ctx context.Context, id int64) (*domain.Project, error) {
	var project domain.Project
	err := r.db.WithContext(ctx).
		Table("projects").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// DecrementEnvelope atomically decreases the envelope_remaining if sufficient funds exist.
// Returns ErrEnvelopeExhausted if envelope_remaining < amount.
func (r *projectRepository) DecrementEnvelope(ctx context.Context, projectID int64, amount int64) error {
	result := r.db.WithContext(ctx).
		Table("projects").
		Where("id = ? AND envelope_remaining >= ? AND deleted_at IS NULL", projectID, amount).
		Update("envelope_remaining", gorm.Expr("envelope_remaining - ?", amount))

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.ErrEnvelopeExhausted
	}
	return nil
}

// List implements dynamic filtering and pagination for projects.
func (r *projectRepository) List(ctx context.Context, filter querybuilder.ProjectFilter) ([]*domain.Project, int64, error) {
	query := r.db.WithContext(ctx).Table("projects").Where("deleted_at IS NULL")

	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.ManagerID != nil {
		query = query.Where("manager_id = ?", *filter.ManagerID)
	}
	if filter.Search != nil && *filter.Search != "" {
		query = query.Where("title ILIKE ?", "%"+*filter.Search+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := filter.SortColumn
	if filter.SortDesc {
		order += " DESC"
	}
	query = query.Order(order)

	offset := (filter.Page - 1) * filter.Limit
	query = query.Offset(offset).Limit(filter.Limit)

	var projects []*domain.Project
	if err := query.Find(&projects).Error; err != nil {
		return nil, 0, err
	}

	return projects, total, nil
}

// Delete performs a soft delete on a project.
func (r *projectRepository) Delete(ctx context.Context, publicID string) error {
	return r.db.WithContext(ctx).
		Table("projects").
		Where("public_id = ?", publicID).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}

// Restore removes the soft delete timestamp, restoring the project.
func (r *projectRepository) Restore(ctx context.Context, publicID string) error {
	return r.db.WithContext(ctx).
		Unscoped().
		Table("projects").
		Where("public_id = ?", publicID).
		Update("deleted_at", nil).Error
}
