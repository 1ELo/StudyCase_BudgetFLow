package http

import (
	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(
	db *gorm.DB,
	authHandler *AuthHandler,
	topupHandler *TopupHandler,
	projectHandler *ProjectHandler,
	claimHandler *ClaimHandler,
	payoutHandler *PayoutHandler,
) *gin.Engine {
	r := gin.New()

	// 100 requests per minute = 1.66 rps
	r.Use(middleware.StructuredLogger(), gin.Recovery(), middleware.RateLimiter(100.0/60.0, 100))

	v1 := r.Group("/api/v1")
	{
		// Auth (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.Authenticate())
		{
			// Balance
			protected.GET("/me/balance", authHandler.GetBalance)

			// Topups
			protected.POST("/topups", middleware.Authorize(domain.RoleManager), topupHandler.CreateTopup)
			protected.POST("/topups/:public_id/review", middleware.Authorize(domain.RoleFinance), middleware.Idempotency(db), topupHandler.ReviewTopup)
			protected.GET("/me/topups", middleware.Authorize(domain.RoleManager), topupHandler.ListMyTopups)

			// Projects
			protected.POST("/projects", middleware.Authorize(domain.RoleManager), projectHandler.CreateProject)
			protected.GET("/projects", projectHandler.ListProjects)
			protected.GET("/projects/:public_id", projectHandler.GetProject)

			// Claims
			protected.POST("/projects/:public_id/claims", middleware.Authorize(domain.RoleEmployee), claimHandler.CreateClaim)
			protected.POST("/claims/:public_id/review", middleware.Authorize(domain.RoleFinance, domain.RoleManager), middleware.Idempotency(db), claimHandler.ReviewClaim)
			protected.GET("/me/claims", middleware.Authorize(domain.RoleEmployee), claimHandler.ListMyClaims)

			// Payouts
			protected.POST("/payouts", middleware.Authorize(domain.RoleEmployee), payoutHandler.CreatePayout)
			protected.GET("/payouts", middleware.Authorize(domain.RoleEmployee), payoutHandler.ListMyPayouts)
			protected.POST("/payouts/:public_id/review", middleware.Authorize(domain.RoleFinance), middleware.Idempotency(db), payoutHandler.ReviewPayout)
		}
	}

	return r
}
