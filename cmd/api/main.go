package main

import (
	"fmt"
	"log"
	"os"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/delivery/http"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/repository"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Connect to database
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "postgres"),
		getEnv("DB_NAME", "budgetflow"),
		getEnv("DB_SSLMODE", "disable"),
	)

	gormLogger := logger.Default.LogMode(logger.Silent)
	if getEnv("APP_ENV", "development") == "development" {
		gormLogger = logger.Default.LogMode(logger.Info)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connected successfully")

	// Validator
	validate := validator.New()

	// Repositories
	userRepo := repository.NewUserRepository(db)
	managerRepo := repository.NewManagerRepository(db)
	employeeRepo := repository.NewEmployeeRepository(db)
	topupRepo := repository.NewTopupRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	claimRepo := repository.NewClaimRepository(db)
	payoutRepo := repository.NewPayoutRepository(db)

	// Usecases
	authUC := usecase.NewAuthUsecase(db, userRepo, managerRepo, employeeRepo)
	topupUC := usecase.NewTopupUsecase(db, topupRepo, managerRepo)
	projectUC := usecase.NewProjectUsecase(db, projectRepo, managerRepo, userRepo)
	claimUC := usecase.NewClaimUsecase(db, claimRepo, projectRepo, managerRepo, employeeRepo)
	payoutUC := usecase.NewPayoutUsecase(db, payoutRepo, employeeRepo)

	// Handlers
	authHandler := http.NewAuthHandler(authUC, validate)
	topupHandler := http.NewTopupHandler(topupUC, validate)
	projectHandler := http.NewProjectHandler(projectUC, validate)
	claimHandler := http.NewClaimHandler(claimUC, validate)
	payoutHandler := http.NewPayoutHandler(payoutUC, validate)

	// Router
	router := http.SetupRouter(
		authHandler,
		topupHandler,
		projectHandler,
		claimHandler,
		payoutHandler,
	)

	// Start server
	port := getEnv("APP_PORT", "8080")
	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
