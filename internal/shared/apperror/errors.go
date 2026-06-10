package apperror

import "net/http"

// AppError is a typed domain error that maps to HTTP status codes.
type AppError struct {
	HTTPStatus int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func (e *AppError) Error() string { return e.Message }

// NewAppError creates a new AppError with the given HTTP status, code, and message.
func NewAppError(httpStatus int, code, message string) *AppError {
	return &AppError{
		HTTPStatus: httpStatus,
		Code:       code,
		Message:    message,
	}
}

// Predefined application errors.
var (
	ErrUnauthorized        = &AppError{http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized"}
	ErrForbidden           = &AppError{http.StatusForbidden, "FORBIDDEN", "access denied"}
	ErrNotFound            = &AppError{http.StatusNotFound, "NOT_FOUND", "resource not found"}
	ErrConflict            = &AppError{http.StatusConflict, "CONFLICT", "resource conflict"}
	ErrInsufficientBudget  = &AppError{http.StatusUnprocessableEntity, "INSUFFICIENT_BUDGET", "insufficient budget"}
	ErrEnvelopeExhausted   = &AppError{http.StatusUnprocessableEntity, "ENVELOPE_EXHAUSTED", "project envelope insufficient"}
	ErrInsufficientBalance = &AppError{http.StatusUnprocessableEntity, "INSUFFICIENT_BALANCE", "insufficient reimburse balance"}
	ErrDuplicateClaim      = &AppError{http.StatusConflict, "DUPLICATE_CLAIM", "claim already exists for this project"}
	ErrInvalidStatusTrans  = &AppError{http.StatusConflict, "INVALID_STATUS_TRANSITION", "invalid status transition"}
	ErrProjectClosed       = &AppError{http.StatusUnprocessableEntity, "PROJECT_CLOSED", "project is not open"}
	ErrValidation          = &AppError{http.StatusBadRequest, "VALIDATION_ERROR", "validation failed"}
	ErrInternal            = &AppError{http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error"}
)
