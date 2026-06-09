package domain

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID           uuid.UUID
	UserID       int64
	RefreshToken string
	IsBlocked    bool
	ExpiresAt    time.Time
	CreatedAt    time.Time
}
