package domain

// Manager represents the managers table row.
// It extends User with budget tracking fields.
type Manager struct {
	UserID          int64
	BudgetAvailable int64
	BudgetLocked    int64
}
