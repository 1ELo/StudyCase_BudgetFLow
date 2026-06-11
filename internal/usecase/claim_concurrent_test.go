// go:build integration

package usecase_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/repository"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/usecase"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// getTestDB connects to a real PostgreSQL database for concurrency testing.
// It reads database credentials from the project's .env file.
func getTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	// Load .env from project root
	_ = godotenv.Load("../../.env")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		envOrDefault("DB_HOST", "localhost"),
		envOrDefault("DB_PORT", "5432"),
		envOrDefault("DB_USER", "postgres"),
		envOrDefault("DB_PASSWORD", "postgres"),
		envOrDefault("DB_NAME", "budgetflow"),
		envOrDefault("DB_SSLMODE", "disable"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("Skipping concurrency test: cannot connect to Postgres: %v", err)
	}

	// Verify connection
	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("Skipping concurrency test: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Skipf("Skipping concurrency test: cannot ping Postgres: %v", err)
	}

	return db
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// TestConcurrentClaimApproval_NoNegativeEnvelope tests that concurrent claim
// approvals never cause envelope_remaining to go negative.
//
// Setup:
//   - 1 project with envelope_remaining = claim_amount * 3 (room for exactly 3 claims)
//   - 5 employees, each with 1 pending claim
//
// Action:
//   - Launch 5 goroutines simultaneously, each approving 1 claim
//
// Assert:
//   - envelope_remaining >= 0 (NEVER negative)
//   - Exactly 3 claims become 'approved'
//   - Exactly 2 claims fail (envelope exhausted)
func TestConcurrentClaimApproval_NoNegativeEnvelope(t *testing.T) {
	db := getTestDB(t)

	ctx := context.Background()

	// ── Cleanup: use a unique test run suffix to avoid collisions ──
	testRunID := uuid.New().String()[:8]

	// ── Setup test data directly via SQL for isolation ──
	// 1. Create a manager user
	managerPublicID := uuid.New()
	managerEmail := fmt.Sprintf("test-manager-%s@test.com", testRunID)
	err := db.Exec(`
		INSERT INTO users (public_id, name, email, password, role, created_at, updated_at)
		VALUES (?, ?, ?, '$2a$12$dummy', 'manager', NOW(), NOW())`,
		managerPublicID, "Test Manager", managerEmail,
	).Error
	if err != nil {
		t.Fatalf("failed to create test manager user: %v", err)
	}

	var managerUserID int64
	db.Raw("SELECT id FROM users WHERE email = ?", managerEmail).Scan(&managerUserID)

	// 2. Create manager record with enough locked budget
	claimAmount := int64(500_000)
	envelopeTotal := claimAmount * 3 // room for exactly 3 claims

	err = db.Exec(`
		INSERT INTO managers (user_id, budget_available, budget_locked)
		VALUES (?, 0, ?)`,
		managerUserID, envelopeTotal,
	).Error
	if err != nil {
		t.Fatalf("failed to create test manager: %v", err)
	}

	// 3. Create a project with envelope for exactly 3 claims
	projectPublicID := uuid.New()
	err = db.Exec(`
		INSERT INTO projects (public_id, manager_id, title, claim_amount, envelope_total, envelope_remaining, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'open', NOW(), NOW())`,
		projectPublicID, managerUserID,
		fmt.Sprintf("Concurrent Test Project %s", testRunID),
		claimAmount, envelopeTotal, envelopeTotal,
	).Error
	if err != nil {
		t.Fatalf("failed to create test project: %v", err)
	}

	var projectID int64
	db.Raw("SELECT id FROM projects WHERE public_id = ?", projectPublicID).Scan(&projectID)

	// 4. Create 5 employee users + employee records + pending claims
	numEmployees := 5
	claimPublicIDs := make([]uuid.UUID, numEmployees)
	employeeUserIDs := make([]int64, numEmployees)

	for i := 0; i < numEmployees; i++ {
		empPublicID := uuid.New()
		empEmail := fmt.Sprintf("test-emp-%s-%d@test.com", testRunID, i)

		err = db.Exec(`
			INSERT INTO users (public_id, name, email, password, role, created_at, updated_at)
			VALUES (?, ?, ?, '$2a$12$dummy', 'employee', NOW(), NOW())`,
			empPublicID, fmt.Sprintf("Test Employee %d", i), empEmail,
		).Error
		if err != nil {
			t.Fatalf("failed to create test employee user %d: %v", i, err)
		}

		var empUserID int64
		db.Raw("SELECT id FROM users WHERE email = ?", empEmail).Scan(&empUserID)
		employeeUserIDs[i] = empUserID

		// Create employee record
		err = db.Exec(`
			INSERT INTO employees (user_id, reimburse_available, reimburse_locked)
			VALUES (?, 0, 0)`, empUserID,
		).Error
		if err != nil {
			t.Fatalf("failed to create test employee %d: %v", i, err)
		}

		// Create pending claim
		claimPubID := uuid.New()
		claimPublicIDs[i] = claimPubID
		err = db.Exec(`
			INSERT INTO expense_claims (public_id, project_id, employee_id, receipt_url, status, created_at, updated_at)
			VALUES (?, ?, ?, 'https://receipt.example.com/test', 'pending', NOW(), NOW())`,
			claimPubID, projectID, empUserID,
		).Error
		if err != nil {
			t.Fatalf("failed to create test claim %d: %v", i, err)
		}
	}

	// ── Build the real usecase with real repositories ──
	claimRepo := repository.NewClaimRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	managerRepo := repository.NewManagerRepository(db)
	employeeRepo := repository.NewEmployeeRepository(db)
	claimUC := usecase.NewClaimUsecase(db, claimRepo, projectRepo, managerRepo, employeeRepo)

	// ── Action: launch 5 goroutines simultaneously ──
	var wg sync.WaitGroup
	results := make([]error, numEmployees)

	// Use a channel as a barrier so all goroutines start at the same time
	barrier := make(chan struct{})

	for i := 0; i < numEmployees; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-barrier // wait for the barrier to be released
			_, err := claimUC.ReviewClaim(
				ctx,
				int64(0), // callerUserID=0 (finance role bypasses IDOR check)
				string(domain.RoleFinance),
				claimPublicIDs[idx].String(),
				domain.ReviewClaimInput{Status: domain.ClaimApproved},
			)
			results[idx] = err
		}(i)
	}

	// Release all goroutines at once
	close(barrier)
	wg.Wait()

	// ── Assertions ──

	// 1. Count successes and failures
	approvedCount := 0
	failedCount := 0
	for i, err := range results {
		if err == nil {
			approvedCount++
			t.Logf("Goroutine %d: APPROVED", i)
		} else {
			failedCount++
			t.Logf("Goroutine %d: FAILED (%v)", i, err)
		}
	}

	// 2. Assert exactly 3 approved, 2 failed
	if approvedCount != 3 {
		t.Errorf("expected exactly 3 approved claims, got %d", approvedCount)
	}
	if failedCount != 2 {
		t.Errorf("expected exactly 2 failed claims, got %d", failedCount)
	}

	// 3. Assert envelope_remaining is NEVER negative
	var envelopeRemaining int64
	db.Raw("SELECT envelope_remaining FROM projects WHERE id = ?", projectID).Scan(&envelopeRemaining)
	t.Logf("Final envelope_remaining: %d", envelopeRemaining)

	if envelopeRemaining < 0 {
		t.Fatalf("CRITICAL: envelope_remaining is negative (%d) — anti-double-spend FAILED", envelopeRemaining)
	}
	if envelopeRemaining != 0 {
		t.Errorf("expected envelope_remaining=0 (all 3 slots used), got %d", envelopeRemaining)
	}

	// 4. Verify DB state: count approved claims in DB
	var dbApprovedCount int64
	db.Raw("SELECT COUNT(*) FROM expense_claims WHERE project_id = ? AND status = 'approved'", projectID).Scan(&dbApprovedCount)
	if dbApprovedCount != 3 {
		t.Errorf("expected 3 approved claims in DB, got %d", dbApprovedCount)
	}

	// 5. Verify employee balances: each approved employee should have claim_amount available
	for i, empID := range employeeUserIDs {
		if results[i] == nil { // this employee was approved
			var reimburseAvailable int64
			db.Raw("SELECT reimburse_available FROM employees WHERE user_id = ?", empID).Scan(&reimburseAvailable)
			if reimburseAvailable != claimAmount {
				t.Errorf("employee %d: expected reimburse_available=%d, got %d", i, claimAmount, reimburseAvailable)
			}
		}
	}

	// 6. Verify manager budget_locked decreased correctly
	var budgetLocked int64
	db.Raw("SELECT budget_locked FROM managers WHERE user_id = ?", managerUserID).Scan(&budgetLocked)
	expectedLocked := int64(0) // started at envelopeTotal, 3 claims approved should drain it
	if budgetLocked != expectedLocked {
		t.Errorf("expected manager budget_locked=%d, got %d", expectedLocked, budgetLocked)
	}

	// ── Cleanup test data ──
	cleanupTestData(t, db, testRunID)
}

// cleanupTestData removes all test records created during this test run.
func cleanupTestData(t *testing.T, db *gorm.DB, testRunID string) {
	t.Helper()

	likePattern := fmt.Sprintf("%%-%s@test.com", testRunID)
	likePatternProject := fmt.Sprintf("%%Test Project %s", testRunID)

	// Delete claims for test project
	db.Exec(`DELETE FROM expense_claims WHERE project_id IN (SELECT id FROM projects WHERE title LIKE ?)`, likePatternProject)
	// Delete test project
	db.Exec(`DELETE FROM projects WHERE title LIKE ?`, likePatternProject)
	// Delete employee records
	db.Exec(`DELETE FROM employees WHERE user_id IN (SELECT id FROM users WHERE email LIKE ?)`, likePattern)
	// Delete manager records
	db.Exec(`DELETE FROM managers WHERE user_id IN (SELECT id FROM users WHERE email LIKE ?)`, likePattern)
	// Delete test users
	db.Exec(`DELETE FROM users WHERE email LIKE ?`, likePattern)
}

// Ensure test file imports are used
var _ = time.Now
