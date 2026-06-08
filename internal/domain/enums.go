package domain

// Role represents the user role in the system.
type Role string

const (
	RoleFinance  Role = "finance"
	RoleManager  Role = "manager"
	RoleEmployee Role = "employee"
)

// TopupStatus represents the status of a budget topup request.
type TopupStatus string

const (
	TopupPending  TopupStatus = "pending"
	TopupApproved TopupStatus = "approved"
	TopupRejected TopupStatus = "rejected"
)

// ProjectStatus represents the status of a project.
type ProjectStatus string

const (
	ProjectOpen   ProjectStatus = "open"
	ProjectClosed ProjectStatus = "closed"
)

// ClaimStatus represents the status of an expense claim.
type ClaimStatus string

const (
	ClaimPending  ClaimStatus = "pending"
	ClaimApproved ClaimStatus = "approved"
	ClaimRejected ClaimStatus = "rejected"
)

// PayoutStatus represents the status of a payout.
type PayoutStatus string

const (
	PayoutPending   PayoutStatus = "pending"
	PayoutCompleted PayoutStatus = "completed"
	PayoutFailed    PayoutStatus = "failed"
)
