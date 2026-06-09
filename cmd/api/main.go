package main

import (
	"fmt"
	"log"
	"os"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/repository"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/usecase"
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

	// TODO Phase 7-8: wire middleware, handlers, and router
	_ = authUC
	_ = topupUC
	_ = projectUC
	_ = claimUC
	_ = payoutUC

	port := getEnv("APP_PORT", "8080")
	log.Printf("Usecases initialized. Server wiring pending — port will be %s", port)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
