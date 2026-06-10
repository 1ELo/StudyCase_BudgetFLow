package response

import (
	"errors"
	"net/http"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/apperror"
	"github.com/gin-gonic/gin"
)

// SuccessResponse is the standard JSON success response.
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse is the standard JSON error response.
type ErrorResponse struct {
	Success bool       `json:"success"`
	Message string     `json:"message"`
	Error   *ErrorData `json:"error,omitempty"`
}

// ErrorData holds error code details.
type ErrorData struct {
	Code string `json:"code"`
}

// PaginatedData wraps list data with pagination metadata.
type PaginatedData struct {
	Items      interface{}    `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

// PaginationMeta holds pagination information.
type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalItems int64 `json:"total_items"`
	TotalPages int64 `json:"total_pages"`
}

// Success sends a standardized success JSON response.
func Success(c *gin.Context, httpStatus int, message string, data interface{}) {
	c.JSON(httpStatus, SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Error sends a standardized error JSON response.
// It handles *AppError types for structured error codes, or falls back to 500.
func Error(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.HTTPStatus, ErrorResponse{
			Success: false,
			Message: appErr.Message,
			Error: &ErrorData{
				Code: appErr.Code,
			},
		})
		return
	}

	// Fallback: never expose internal error details
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Success: false,
		Message: "internal server error",
		Error: &ErrorData{
			Code: "INTERNAL_ERROR",
		},
	})
}

// ValidationError sends a validation error response with a custom message.
func ValidationError(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Success: false,
		Message: message,
		Error: &ErrorData{
			Code: "VALIDATION_ERROR",
		},
	})
}
