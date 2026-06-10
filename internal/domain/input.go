package domain

// RegisterInput represents the JSON body for user registration.
type RegisterInput struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Role     Role   `json:"role" validate:"required,oneof=manager employee"`
}

// LoginInput represents the JSON body for user login.
type LoginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RefreshInput represents the JSON body for refreshing tokens.
type RefreshInput struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// TopupInput represents the JSON body for requesting a budget topup.
type TopupInput struct {
	Amount int64 `json:"amount" validate:"required,gt=0"`
}

// ReviewTopupInput represents the JSON body for reviewing a topup request.
type ReviewTopupInput struct {
	Status TopupStatus `json:"status" validate:"required,oneof=approved rejected"`
}

// CreateProjectInput represents the JSON body for creating a new project.
type CreateProjectInput struct {
	Title         string `json:"title" validate:"required"`
	ClaimAmount   int64  `json:"claim_amount" validate:"required,gt=0"`
	EnvelopeTotal int64  `json:"envelope_total" validate:"required,gtefield=ClaimAmount"`
}

// ProjectListQuery represents the URL query parameters for listing projects.
type ProjectListQuery struct {
	Page            int    `form:"page" validate:"gte=1"`
	Limit           int    `form:"limit" validate:"gte=1,lte=100"`
	Search          string `form:"search"`
	Status          string `form:"status" validate:"omitempty,oneof=open closed"`
	Sort            string `form:"sort"`
	ManagerPublicID string `form:"manager_id" validate:"omitempty,uuid4"`
}

// CreateClaimInput represents the JSON body for creating an expense claim.
type CreateClaimInput struct {
	ReceiptURL string `json:"receipt_url" validate:"required,url"`
}

// ReviewClaimInput represents the JSON body for reviewing an expense claim.
type ReviewClaimInput struct {
	Status ClaimStatus `json:"status" validate:"required,oneof=approved rejected"`
}

// CreatePayoutInput represents the JSON body for requesting a payout.
type CreatePayoutInput struct {
	Amount int64 `json:"amount" validate:"required,gt=0"`
}

// ReviewPayoutInput represents the JSON body for reviewing a payout.
type ReviewPayoutInput struct {
	Status PayoutStatus `json:"status" validate:"required,oneof=completed failed"`
}
