package domain

import (
	"time"

	"github.com/google/uuid"
)

// BudgetTopup represents a budget topup request from a manager.
type BudgetTopup struct {
	ID         int64
	PublicID   uuid.UUID
	ManagerID  int64
	Amount     int64
	Status     TopupStatus
	ReviewedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
