package http

import (
	"net/http"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/response"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type PayoutHandler struct {
	payoutUC usecase.PayoutUsecase
	validate *validator.Validate
}

func NewPayoutHandler(payoutUC usecase.PayoutUsecase, validate *validator.Validate) *PayoutHandler {
	return &PayoutHandler{payoutUC: payoutUC, validate: validate}
}

func (h *PayoutHandler) CreatePayout(c *gin.Context) {
	var input domain.CreatePayoutInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ValidationError(c, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	userID := c.GetInt64("userID")
	payout, err := h.payoutUC.CreatePayout(c.Request.Context(), userID, input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "payout request created", gin.H{
		"public_id": payout.PublicID, "amount": payout.Amount, "status": payout.Status,
	})
}

func (h *PayoutHandler) ReviewPayout(c *gin.Context) {
	publicID := c.Param("public_id")
	var input domain.ReviewPayoutInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ValidationError(c, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	payout, err := h.payoutUC.ReviewPayout(c.Request.Context(), publicID, input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "payout reviewed", gin.H{
		"public_id": payout.PublicID, "amount": payout.Amount,
		"fee": payout.Fee, "net_amount": payout.NetAmount,
		"status": payout.Status, "reviewed_at": payout.ReviewedAt,
	})
}

func (h *PayoutHandler) ListMyPayouts(c *gin.Context) {
	userID := c.GetInt64("userID")
	payouts, err := h.payoutUC.ListMyPayouts(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	items := make([]gin.H, len(payouts))
	for i, p := range payouts {
		items[i] = gin.H{"public_id": p.PublicID, "amount": p.Amount, "fee": p.Fee, "net_amount": p.NetAmount, "status": p.Status, "created_at": p.CreatedAt, "reviewed_at": p.ReviewedAt}
	}
	response.Success(c, http.StatusOK, "payouts retrieved", gin.H{"items": items})
}
