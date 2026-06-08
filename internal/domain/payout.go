package domain

import (
	"time"

	"github.com/google/uuid"
)

// Payout represents a reimburse payout request to an employee.
type Payout struct {
	ID         int64
	PublicID   uuid.UUID
	EmployeeID int64
	Amount     int64
	Fee        *int64
	NetAmount  *int64
	Status     PayoutStatus
	ReviewedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
