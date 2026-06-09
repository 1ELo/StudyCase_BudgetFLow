package repository

import (
	"context"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"gorm.io/gorm"
)

type SessionRepository interface {
	CreateSession(ctx context.Context, session *domain.Session) error
	GetSession(ctx context.Context, id string) (*domain.Session, error)
	BlockSession(ctx context.Context, id string) error
	DeleteAllUserSessions(ctx context.Context, userID int64) error
}

type sessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) CreateSession(ctx context.Context, session *domain.Session) error {
	return r.db.WithContext(ctx).Table("sessions").Create(session).Error
}

func (r *sessionRepository) GetSession(ctx context.Context, id string) (*domain.Session, error) {
	var session domain.Session
	err := r.db.WithContext(ctx).Table("sessions").Where("id = ?", id).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *sessionRepository) BlockSession(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Table("sessions").Where("id = ?", id).Update("is_blocked", true).Error
}

func (r *sessionRepository) DeleteAllUserSessions(ctx context.Context, userID int64) error {
	// Alternatively, just block them
	return r.db.WithContext(ctx).Table("sessions").Where("user_id = ?", userID).Update("is_blocked", true).Error
}

// Session struct (would normally be in domain, defining here temporarily to avoid multiple files if not strictly needed,
// actually I should put this in domain.go)
