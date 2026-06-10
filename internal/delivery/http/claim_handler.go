package http

import (
	"net/http"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/response"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type ClaimHandler struct {
	claimUC  usecase.ClaimUsecase
	validate *validator.Validate
}

func NewClaimHandler(claimUC usecase.ClaimUsecase, validate *validator.Validate) *ClaimHandler {
	return &ClaimHandler{claimUC: claimUC, validate: validate}
}

func (h *ClaimHandler) CreateClaim(c *gin.Context) {
	projectPublicID := c.Param("public_id")
	var input domain.CreateClaimInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ValidationError(c, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	employeeID := c.GetInt64("userID")
	claim, err := h.claimUC.CreateClaim(c.Request.Context(), employeeID, projectPublicID, input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "claim created", gin.H{
		"public_id": claim.PublicID, "receipt_url": claim.ReceiptURL, "status": claim.Status,
	})
}

func (h *ClaimHandler) ReviewClaim(c *gin.Context) {
	claimPublicID := c.Param("public_id")
	var input domain.ReviewClaimInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ValidationError(c, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	callerUserID := c.GetInt64("userID")
	callerRole := c.GetString("role")
	claim, err := h.claimUC.ReviewClaim(c.Request.Context(), callerUserID, callerRole, claimPublicID, input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "claim reviewed", gin.H{
		"public_id": claim.PublicID, "status": claim.Status, "reviewed_at": claim.ReviewedAt,
	})
}

func (h *ClaimHandler) ListMyClaims(c *gin.Context) {
	employeeID := c.GetInt64("userID")
	claims, err := h.claimUC.ListMyClaims(c.Request.Context(), employeeID)
	if err != nil {
		response.Error(c, err)
		return
	}
	items := make([]gin.H, len(claims))
	for i, cl := range claims {
		items[i] = gin.H{"public_id": cl.PublicID, "receipt_url": cl.ReceiptURL, "status": cl.Status, "created_at": cl.CreatedAt, "reviewed_at": cl.ReviewedAt}
	}
	response.Success(c, http.StatusOK, "claims retrieved", gin.H{"items": items})
}
