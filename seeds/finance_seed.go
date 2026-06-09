package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/hash"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "postgres"),
		getEnv("DB_NAME", "budgetflow"),
		getEnv("DB_SSLMODE", "disable"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	ctx := context.Background()

	seedUser(ctx, db, "BudgetFlow Finance", "finance@budgetflow.id", "Finance123!", domain.RoleFinance)
	seedUser(ctx, db, "BudgetFlow Manager", "manager@budgetflow.id", "Manager123!", domain.RoleManager)
	seedUser(ctx, db, "BudgetFlow Employee", "employee@budgetflow.id", "Employee123!", domain.RoleEmployee)

	log.Println("Seeding completed successfully!")
}

func seedUser(ctx context.Context, db *gorm.DB, name, email, password string, role domain.Role) {
	// Check if user already exists (idempotent)
	var count int64
	db.WithContext(ctx).Table("users").Where("email = ?", email).Count(&count)
	if count > 0 {
		log.Printf("User %s already exists, skipping", email)
		return
	}

	hashedPassword, err := hash.Hash(password)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	user := &domain.User{
		PublicID: uuid.New(),
		Name:     name,
		Email:    email,
		Password: hashedPassword,
		Role:     role,
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Table("users").Create(user).Error; err != nil {
			return err
		}

		switch role {
		case domain.RoleManager:
			manager := &domain.Manager{UserID: user.ID, BudgetAvailable: 0, BudgetLocked: 0}
			return tx.WithContext(ctx).Table("managers").Create(manager).Error
		case domain.RoleEmployee:
			employee := &domain.Employee{UserID: user.ID, ReimburseAvailable: 0, ReimburseLocked: 0}
			return tx.WithContext(ctx).Table("employees").Create(employee).Error
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to seed user %s: %v", email, err)
	}

	log.Printf("Seeded user: %s (%s)", email, role)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
