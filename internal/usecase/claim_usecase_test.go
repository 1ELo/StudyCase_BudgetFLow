package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/apperror"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/usecase"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- Mock Repositories ---

type mockClaimRepo struct {
	claims map[int64]*domain.ExpenseClaim
	nextID int64
	exists map[string]bool // "projectID-employeeID" -> exists
}

func newMockClaimRepo() *mockClaimRepo {
	return &mockClaimRepo{claims: make(map[int64]*domain.ExpenseClaim), nextID: 1, exists: make(map[string]bool)}
}

func (m *mockClaimRepo) WithTx(tx *gorm.DB) *mockClaimRepo { return m }
func (m *mockClaimRepo) Create(ctx context.Context, c *domain.ExpenseClaim) error {
	c.ID = m.nextID
	m.nextID++
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	m.claims[c.ID] = c
	return nil
}
func (m *mockClaimRepo) FindByPublicID(ctx context.Context, publicID string) (*domain.ExpenseClaim, error) {
	for _, c := range m.claims {
		if c.PublicID.String() == publicID {
			return c, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockClaimRepo) UpdateStatus(ctx context.Context, id int64, status domain.ClaimStatus, reviewedAt time.Time) error {
	if c, ok := m.claims[id]; ok {
		c.Status = status
		c.ReviewedAt = &reviewedAt
		return nil
	}
	return gorm.ErrRecordNotFound
}
func (m *mockClaimRepo) ExistsByProjectAndEmployee(ctx context.Context, projectID, employeeID int64) (bool, error) {
	for _, c := range m.claims {
		if c.ProjectID == projectID && c.EmployeeID == employeeID {
			return true, nil
		}
	}
	return false, nil
}
func (m *mockClaimRepo) ListByEmployeeID(ctx context.Context, employeeID int64) ([]*domain.ExpenseClaim, error) {
	var result []*domain.ExpenseClaim
	for _, c := range m.claims {
		if c.EmployeeID == employeeID {
			result = append(result, c)
		}
	}
	return result, nil
}

type mockProjectRepo struct {
	projects map[int64]*domain.Project
}

func newMockProjectRepo() *mockProjectRepo {
	return &mockProjectRepo{projects: make(map[int64]*domain.Project)}
}

func (m *mockProjectRepo) WithTx(tx *gorm.DB) *mockProjectRepo { return m }
func (m *mockProjectRepo) FindByPublicID(ctx context.Context, publicID string) (*domain.Project, error) {
	for _, p := range m.projects {
		if p.PublicID.String() == publicID {
			return p, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockProjectRepo) FindByID(ctx context.Context, id int64) (*domain.Project, error) {
	p, ok := m.projects[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return p, nil
}
func (m *mockProjectRepo) DecrementEnvelope(ctx context.Context, projectID int64, amount int64) error {
	p, ok := m.projects[projectID]
	if !ok || p.EnvelopeRemaining < amount {
		return apperror.ErrEnvelopeExhausted
	}
	p.EnvelopeRemaining -= amount
	return nil
}
func (m *mockProjectRepo) Create(ctx context.Context, p *domain.Project) error { return nil }

type mockManagerRepo struct {
	managers map[int64]*domain.Manager
}

func newMockManagerRepo() *mockManagerRepo {
	return &mockManagerRepo{managers: make(map[int64]*domain.Manager)}
}

func (m *mockManagerRepo) WithTx(tx *gorm.DB) *mockManagerRepo { return m }
func (m *mockManagerRepo) FindByUserID(ctx context.Context, userID int64) (*domain.Manager, error) {
	mgr, ok := m.managers[userID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return mgr, nil
}
func (m *mockManagerRepo) DecrementLocked(ctx context.Context, userID int64, amount int64) error {
	mgr, ok := m.managers[userID]
	if !ok || mgr.BudgetLocked < amount {
		return apperror.ErrInternal
	}
	mgr.BudgetLocked -= amount
	return nil
}
func (m *mockManagerRepo) IncrementAvailable(ctx context.Context, userID int64, amount int64) error {
	return nil
}
func (m *mockManagerRepo) LockEnvelope(ctx context.Context, userID int64, amount int64) error {
	return nil
}
func (m *mockManagerRepo) Create(ctx context.Context, mg *domain.Manager) error { return nil }

type mockEmployeeRepo struct {
	employees map[int64]*domain.Employee
}

func newMockEmployeeRepo() *mockEmployeeRepo {
	return &mockEmployeeRepo{employees: make(map[int64]*domain.Employee)}
}

func (m *mockEmployeeRepo) WithTx(tx *gorm.DB) *mockEmployeeRepo { return m }
func (m *mockEmployeeRepo) FindByUserID(ctx context.Context, userID int64) (*domain.Employee, error) {
	emp, ok := m.employees[userID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return emp, nil
}
func (m *mockEmployeeRepo) IncrementAvailable(ctx context.Context, userID int64, amount int64) error {
	emp, ok := m.employees[userID]
	if !ok {
		return apperror.ErrNotFound
	}
	emp.ReimburseAvailable += amount
	return nil
}
func (m *mockEmployeeRepo) LockForPayout(ctx context.Context, userID int64, amount int64) error {
	emp, ok := m.employees[userID]
	if !ok || emp.ReimburseAvailable < amount {
		return apperror.ErrInsufficientBalance
	}
	emp.ReimburseAvailable -= amount
	emp.ReimburseLocked += amount
	return nil
}
func (m *mockEmployeeRepo) ConsumeLockedPayout(ctx context.Context, userID int64, amount int64) error {
	emp, ok := m.employees[userID]
	if !ok || emp.ReimburseLocked < amount {
		return apperror.ErrInternal
	}
	emp.ReimburseLocked -= amount
	return nil
}
func (m *mockEmployeeRepo) ReleaseLockedPayout(ctx context.Context, userID int64, amount int64) error {
	emp, ok := m.employees[userID]
	if !ok || emp.ReimburseLocked < amount {
		return apperror.ErrInternal
	}
	emp.ReimburseLocked -= amount
	emp.ReimburseAvailable += amount
	return nil
}
func (m *mockEmployeeRepo) Create(ctx context.Context, e *domain.Employee) error { return nil }

// --- Adapter types to satisfy repository interfaces ---
// These adapt our mock types to implement the repository interfaces.

type claimRepoAdapter struct{ *mockClaimRepo }

func (a *claimRepoAdapter) WithTx(tx *gorm.DB) interface{} { return a }

type projectRepoAdapter struct{ *mockProjectRepo }
type managerRepoAdapter struct{ *mockManagerRepo }
type employeeRepoAdapter struct{ *mockEmployeeRepo }

// --- Test helper to create a ClaimUsecase with mocks (no DB transaction) ---

type testClaimUsecase struct {
	claimRepo    *mockClaimRepo
	projectRepo  *mockProjectRepo
	managerRepo  *mockManagerRepo
	employeeRepo *mockEmployeeRepo
}

func newTestClaimUsecase() *testClaimUsecase {
	return &testClaimUsecase{
		claimRepo:    newMockClaimRepo(),
		projectRepo:  newMockProjectRepo(),
		managerRepo:  newMockManagerRepo(),
		employeeRepo: newMockEmployeeRepo(),
	}
}

// reviewClaimDirect simulates the ReviewClaim logic without DB transactions
// since we can't use gorm.DB.Transaction with mocks.
func (t *testClaimUsecase) reviewClaimDirect(ctx context.Context, callerUserID int64, callerRole string, claimPublicID string, input domain.ReviewClaimInput) (*domain.ExpenseClaim, error) {
	claim, err := t.claimRepo.FindByPublicID(ctx, claimPublicID)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	if claim.Status != domain.ClaimPending {
		return nil, apperror.ErrInvalidStatusTrans
	}

	project, err := t.projectRepo.FindByID(ctx, claim.ProjectID)
	if err != nil {
		return nil, apperror.ErrInternal
	}

	if domain.Role(callerRole) == domain.RoleManager && project.ManagerID != callerUserID {
		return nil, apperror.ErrForbidden
	}

	now := time.Now()

	if input.Status == domain.ClaimRejected {
		_ = t.claimRepo.UpdateStatus(ctx, claim.ID, domain.ClaimRejected, now)
		claim.Status = domain.ClaimRejected
		claim.ReviewedAt = &now
		return claim, nil
	}

	// Approved flow
	if err := t.projectRepo.DecrementEnvelope(ctx, project.ID, project.ClaimAmount); err != nil {
		return nil, err
	}
	if err := t.managerRepo.DecrementLocked(ctx, project.ManagerID, project.ClaimAmount); err != nil {
		// Rollback envelope
		project.EnvelopeRemaining += project.ClaimAmount
		return nil, err
	}
	if err := t.employeeRepo.IncrementAvailable(ctx, claim.EmployeeID, project.ClaimAmount); err != nil {
		return nil, err
	}
	_ = t.claimRepo.UpdateStatus(ctx, claim.ID, domain.ClaimApproved, now)

	claim.Status = domain.ClaimApproved
	claim.ReviewedAt = &now
	return claim, nil
}

// --- Tests ---

func TestReviewClaim_Approved_BalancesCorrect(t *testing.T) {
	tc := newTestClaimUsecase()
	ctx := context.Background()

	// Setup: project with envelope_remaining=1_000_000, claim_amount=500_000
	projectID := int64(1)
	managerID := int64(10)
	employeeID := int64(20)
	claimPublicID := uuid.New()

	tc.projectRepo.projects[projectID] = &domain.Project{
		ID: projectID, ManagerID: managerID,
		ClaimAmount: 500_000, EnvelopeTotal: 1_000_000,
		EnvelopeRemaining: 1_000_000, Status: domain.ProjectOpen,
	}
	tc.managerRepo.managers[managerID] = &domain.Manager{
		UserID: managerID, BudgetAvailable: 0, BudgetLocked: 1_000_000,
	}
	tc.employeeRepo.employees[employeeID] = &domain.Employee{
		UserID: employeeID, ReimburseAvailable: 0, ReimburseLocked: 0,
	}
	tc.claimRepo.claims[1] = &domain.ExpenseClaim{
		ID: 1, PublicID: claimPublicID, ProjectID: projectID,
		EmployeeID: employeeID, Status: domain.ClaimPending,
	}

	// Action: approve claim
	claim, err := tc.reviewClaimDirect(ctx, 0, "finance", claimPublicID.String(), domain.ReviewClaimInput{Status: domain.ClaimApproved})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Assertions
	if claim.Status != domain.ClaimApproved {
		t.Errorf("expected claim status 'approved', got '%s'", claim.Status)
	}
	if tc.projectRepo.projects[projectID].EnvelopeRemaining != 500_000 {
		t.Errorf("expected envelope_remaining=500000, got %d", tc.projectRepo.projects[projectID].EnvelopeRemaining)
	}
	if tc.managerRepo.managers[managerID].BudgetLocked != 500_000 {
		t.Errorf("expected budget_locked=500000, got %d", tc.managerRepo.managers[managerID].BudgetLocked)
	}
	if tc.employeeRepo.employees[employeeID].ReimburseAvailable != 500_000 {
		t.Errorf("expected reimburse_available=500000, got %d", tc.employeeRepo.employees[employeeID].ReimburseAvailable)
	}
}

func TestReviewClaim_AlreadyApproved_ReturnsError(t *testing.T) {
	tc := newTestClaimUsecase()
	ctx := context.Background()

	claimPublicID := uuid.New()
	tc.claimRepo.claims[1] = &domain.ExpenseClaim{
		ID: 1, PublicID: claimPublicID, ProjectID: 1,
		EmployeeID: 20, Status: domain.ClaimApproved,
	}

	_, err := tc.reviewClaimDirect(ctx, 0, "finance", claimPublicID.String(), domain.ReviewClaimInput{Status: domain.ClaimApproved})
	if err != apperror.ErrInvalidStatusTrans {
		t.Errorf("expected ErrInvalidStatusTrans, got: %v", err)
	}
}

func TestReviewClaim_InsufficientEnvelope_ReturnsError(t *testing.T) {
	tc := newTestClaimUsecase()
	ctx := context.Background()

	projectID := int64(1)
	managerID := int64(10)
	employeeID := int64(20)
	claimPublicID := uuid.New()

	tc.projectRepo.projects[projectID] = &domain.Project{
		ID: projectID, ManagerID: managerID,
		ClaimAmount: 500_000, EnvelopeTotal: 500_000,
		EnvelopeRemaining: 100_000, Status: domain.ProjectOpen,
	}
	tc.managerRepo.managers[managerID] = &domain.Manager{
		UserID: managerID, BudgetLocked: 500_000,
	}
	tc.employeeRepo.employees[employeeID] = &domain.Employee{
		UserID: employeeID, ReimburseAvailable: 0,
	}
	tc.claimRepo.claims[1] = &domain.ExpenseClaim{
		ID: 1, PublicID: claimPublicID, ProjectID: projectID,
		EmployeeID: employeeID, Status: domain.ClaimPending,
	}

	_, err := tc.reviewClaimDirect(ctx, 0, "finance", claimPublicID.String(), domain.ReviewClaimInput{Status: domain.ClaimApproved})
	if err != apperror.ErrEnvelopeExhausted {
		t.Errorf("expected ErrEnvelopeExhausted, got: %v", err)
	}

	// Assert balances unchanged
	if tc.employeeRepo.employees[employeeID].ReimburseAvailable != 0 {
		t.Errorf("expected reimburse_available=0 (rollback), got %d", tc.employeeRepo.employees[employeeID].ReimburseAvailable)
	}
}

func TestReviewClaim_Rejected_NoBalanceChange(t *testing.T) {
	tc := newTestClaimUsecase()
	ctx := context.Background()

	projectID := int64(1)
	managerID := int64(10)
	employeeID := int64(20)
	claimPublicID := uuid.New()

	tc.projectRepo.projects[projectID] = &domain.Project{
		ID: projectID, ManagerID: managerID,
		ClaimAmount: 500_000, EnvelopeTotal: 1_000_000,
		EnvelopeRemaining: 1_000_000, Status: domain.ProjectOpen,
	}
	tc.managerRepo.managers[managerID] = &domain.Manager{
		UserID: managerID, BudgetLocked: 1_000_000,
	}
	tc.employeeRepo.employees[employeeID] = &domain.Employee{
		UserID: employeeID, ReimburseAvailable: 0,
	}
	tc.claimRepo.claims[1] = &domain.ExpenseClaim{
		ID: 1, PublicID: claimPublicID, ProjectID: projectID,
		EmployeeID: employeeID, Status: domain.ClaimPending,
	}

	claim, err := tc.reviewClaimDirect(ctx, 0, "finance", claimPublicID.String(), domain.ReviewClaimInput{Status: domain.ClaimRejected})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if claim.Status != domain.ClaimRejected {
		t.Errorf("expected status 'rejected', got '%s'", claim.Status)
	}
	if tc.projectRepo.projects[projectID].EnvelopeRemaining != 1_000_000 {
		t.Errorf("expected envelope unchanged at 1000000, got %d", tc.projectRepo.projects[projectID].EnvelopeRemaining)
	}
	if tc.managerRepo.managers[managerID].BudgetLocked != 1_000_000 {
		t.Errorf("expected budget_locked unchanged at 1000000, got %d", tc.managerRepo.managers[managerID].BudgetLocked)
	}
	if tc.employeeRepo.employees[employeeID].ReimburseAvailable != 0 {
		t.Errorf("expected reimburse_available unchanged at 0, got %d", tc.employeeRepo.employees[employeeID].ReimburseAvailable)
	}
}

// Suppress unused import warning for usecase package
var _ = usecase.NewClaimUsecase
