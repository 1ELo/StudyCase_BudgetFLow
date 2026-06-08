package domain

// Employee represents the employees table row.
// It extends User with reimburse balance tracking fields.
type Employee struct {
	UserID             int64
	ReimburseAvailable int64
	ReimburseLocked    int64
}
