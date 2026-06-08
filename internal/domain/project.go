package domain

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a budget project created by a manager.
type Project struct {
	ID                int64
	PublicID          uuid.UUID
	ManagerID         int64
	Title             string
	ClaimAmount       int64
	EnvelopeTotal     int64
	EnvelopeRemaining int64
	Status            ProjectStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}
