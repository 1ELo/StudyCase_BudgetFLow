package http

import (
	"net/http"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/response"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type ProjectHandler struct {
	projectUC usecase.ProjectUsecase
	validate  *validator.Validate
}

func NewProjectHandler(projectUC usecase.ProjectUsecase, validate *validator.Validate) *ProjectHandler {
	return &ProjectHandler{projectUC: projectUC, validate: validate}
}

func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var input domain.CreateProjectInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ValidationError(c, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	userID := c.GetInt64("userID")
	project, err := h.projectUC.CreateProject(c.Request.Context(), userID, input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "project created", gin.H{
		"public_id": project.PublicID, "title": project.Title,
		"claim_amount": project.ClaimAmount, "envelope_total": project.EnvelopeTotal,
		"envelope_remaining": project.EnvelopeRemaining, "status": project.Status,
	})
}

func (h *ProjectHandler) ListProjects(c *gin.Context) {
	var query domain.ProjectListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, "invalid query parameters")
		return
	}
	projects, total, err := h.projectUC.ListProjects(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	items := make([]gin.H, len(projects))
	for i, p := range projects {
		items[i] = gin.H{
			"public_id": p.PublicID, "title": p.Title,
			"claim_amount": p.ClaimAmount, "envelope_total": p.EnvelopeTotal,
			"envelope_remaining": p.EnvelopeRemaining, "status": p.Status, "created_at": p.CreatedAt,
		}
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}
	totalPages := (total + int64(limit) - 1) / int64(limit)
	page := query.Page
	if page <= 0 {
		page = 1
	}
	response.Success(c, http.StatusOK, "projects retrieved", response.PaginatedData{
		Items:      items,
		Pagination: response.PaginationMeta{Page: page, Limit: limit, TotalItems: total, TotalPages: totalPages},
	})
}

func (h *ProjectHandler) GetProject(c *gin.Context) {
	publicID := c.Param("public_id")
	project, err := h.projectUC.GetProject(c.Request.Context(), publicID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "project retrieved", gin.H{
		"public_id": project.PublicID, "title": project.Title,
		"claim_amount": project.ClaimAmount, "envelope_total": project.EnvelopeTotal,
		"envelope_remaining": project.EnvelopeRemaining, "status": project.Status, "created_at": project.CreatedAt,
	})
}
