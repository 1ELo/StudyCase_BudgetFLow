package http

import (
	"net/http"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/response"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type TopupHandler struct {
	topupUC  usecase.TopupUsecase
	validate *validator.Validate
}

func NewTopupHandler(topupUC usecase.TopupUsecase, validate *validator.Validate) *TopupHandler {
	return &TopupHandler{topupUC: topupUC, validate: validate}
}

func (h *TopupHandler) CreateTopup(c *gin.Context) {
	var input domain.TopupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ValidationError(c, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	userID := c.GetInt64("userID")
	topup, err := h.topupUC.CreateTopup(c.Request.Context(), userID, input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "topup request created", gin.H{
		"public_id": topup.PublicID, "amount": topup.Amount, "status": topup.Status,
	})
}

func (h *TopupHandler) ReviewTopup(c *gin.Context) {
	publicID := c.Param("public_id")
	var input domain.ReviewTopupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ValidationError(c, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	topup, err := h.topupUC.ReviewTopup(c.Request.Context(), publicID, input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "topup reviewed", gin.H{
		"public_id": topup.PublicID, "amount": topup.Amount,
		"status": topup.Status, "reviewed_at": topup.ReviewedAt,
	})
}

func (h *TopupHandler) ListMyTopups(c *gin.Context) {
	userID := c.GetInt64("userID")
	topups, err := h.topupUC.ListMyTopups(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	items := make([]gin.H, len(topups))
	for i, t := range topups {
		items[i] = gin.H{"public_id": t.PublicID, "amount": t.Amount, "status": t.Status, "created_at": t.CreatedAt, "reviewed_at": t.ReviewedAt}
	}
	response.Success(c, http.StatusOK, "topups retrieved", gin.H{"items": items})
}
