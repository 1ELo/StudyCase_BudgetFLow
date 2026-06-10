package domain

import (
	"time"

	"github.com/google/uuid"
)

// ExpenseClaim represents an expense claim submitted by an employee.
type ExpenseClaim struct {
	ID         int64
	PublicID   uuid.UUID
	ProjectID  int64
	EmployeeID int64
	ReceiptURL string
	Status     ClaimStatus
	ReviewedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
