package http

import (
	"net/http"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/response"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AuthHandler struct {
	authUC   usecase.AuthUsecase
	validate *validator.Validate
}

func NewAuthHandler(authUC usecase.AuthUsecase, validate *validator.Validate) *AuthHandler {
	return &AuthHandler{authUC: authUC, validate: validate}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input domain.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ValidationError(c, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	user, err := h.authUC.Register(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "registration successful", gin.H{
		"public_id": user.PublicID, "name": user.Name,
		"email": user.Email, "role": user.Role,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input domain.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ValidationError(c, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	accessToken, user, err := h.authUC.Login(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "login successful", gin.H{
		"access_token": accessToken,
		"user": gin.H{
			"public_id": user.PublicID, "name": user.Name,
			"email": user.Email, "role": user.Role,
		},
	})
}

func (h *AuthHandler) GetBalance(c *gin.Context) {
	userID := c.GetInt64("userID")
	role := c.GetString("role")
	balance, err := h.authUC.GetBalance(c.Request.Context(), userID, role)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "balance retrieved", balance)
}
