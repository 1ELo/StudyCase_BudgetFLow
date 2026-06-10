package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	deliveryhttp "github.com/1ELo/StudyCase_BudgetFLow/internal/delivery/http"
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

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
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
	sessionRepo := repository.NewSessionRepository(db)

	// Usecases
	authUC := usecase.NewAuthUsecase(db, userRepo, managerRepo, employeeRepo, sessionRepo)
	topupUC := usecase.NewTopupUsecase(db, topupRepo, managerRepo)
	projectUC := usecase.NewProjectUsecase(db, projectRepo, managerRepo, userRepo)
	claimUC := usecase.NewClaimUsecase(db, claimRepo, projectRepo, managerRepo, employeeRepo)
	payoutUC := usecase.NewPayoutUsecase(db, payoutRepo, employeeRepo)

	// Handlers
	authHandler := deliveryhttp.NewAuthHandler(authUC, validate)
	topupHandler := deliveryhttp.NewTopupHandler(topupUC, validate)
	projectHandler := deliveryhttp.NewProjectHandler(projectUC, validate)
	claimHandler := deliveryhttp.NewClaimHandler(claimUC, validate)
	payoutHandler := deliveryhttp.NewPayoutHandler(payoutUC, validate)

	// Configure slog JSON logger
	slogHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(slogHandler))

	// Router
	router := deliveryhttp.SetupRouter(db, authHandler, topupHandler, projectHandler, claimHandler, payoutHandler)

	// Start server with Graceful Shutdown
	port := getEnv("APP_PORT", "8080")
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		slog.Info("Server starting", slog.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	// 5 seconds timeout for active connections to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	slog.Info("Server exiting gracefully")
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
