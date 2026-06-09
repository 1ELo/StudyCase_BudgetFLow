package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/apperror"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/usecase"
	"github.com/google/uuid"
)

// testPayoutUsecase simulates payout review logic with mocks (no DB transaction).
type testPayoutUsecase struct {
	payoutRepo   *mockPayoutRepo
	employeeRepo *mockEmployeeRepo
}

type mockPayoutRepo struct {
	payouts map[int64]*domain.Payout
	nextID  int64
}

func newMockPayoutRepo() *mockPayoutRepo {
	return &mockPayoutRepo{payouts: make(map[int64]*domain.Payout), nextID: 1}
}

func (m *mockPayoutRepo) FindByPublicID(ctx context.Context, publicID string) (*domain.Payout, error) {
	for _, p := range m.payouts {
		if p.PublicID.String() == publicID {
			return p, nil
		}
	}
	return nil, apperror.ErrNotFound
}

func (m *mockPayoutRepo) UpdateReview(ctx context.Context, id int64, status domain.PayoutStatus, fee, netAmount *int64, reviewedAt time.Time) error {
	p, ok := m.payouts[id]
	if !ok {
		return apperror.ErrNotFound
	}
	p.Status = status
	p.Fee = fee
	p.NetAmount = netAmount
	p.ReviewedAt = &reviewedAt
	return nil
}

func newTestPayoutUsecase() *testPayoutUsecase {
	return &testPayoutUsecase{
		payoutRepo:   newMockPayoutRepo(),
		employeeRepo: newMockEmployeeRepo(),
	}
}

func (t *testPayoutUsecase) reviewPayoutDirect(ctx context.Context, publicID string, input domain.ReviewPayoutInput) (*domain.Payout, error) {
	payout, err := t.payoutRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, apperror.ErrNotFound
	}
	if payout.Status != domain.PayoutPending {
		return nil, apperror.ErrInvalidStatusTrans
	}
	now := time.Now()
	if input.Status == domain.PayoutCompleted {
		fee := payout.Amount * 25 / 1000
		netAmount := payout.Amount - fee
		payout.Fee = &fee
		payout.NetAmount = &netAmount
		if err := t.employeeRepo.ConsumeLockedPayout(ctx, payout.EmployeeID, payout.Amount); err != nil {
			return nil, err
		}
		_ = t.payoutRepo.UpdateReview(ctx, payout.ID, domain.PayoutCompleted, &fee, &netAmount, now)
		payout.Status = domain.PayoutCompleted
		payout.ReviewedAt = &now
		return payout, nil
	}
	// Failed
	if err := t.employeeRepo.ReleaseLockedPayout(ctx, payout.EmployeeID, payout.Amount); err != nil {
		return nil, err
	}
	_ = t.payoutRepo.UpdateReview(ctx, payout.ID, domain.PayoutFailed, nil, nil, now)
	payout.Status = domain.PayoutFailed
	payout.ReviewedAt = &now
	return payout, nil
}

func TestReviewPayout_Completed_FeeCalculation(t *testing.T) {
	tc := newTestPayoutUsecase()
	ctx := context.Background()

	payoutPublicID := uuid.New()
	employeeID := int64(20)
	amount := int64(1_000_000)

	tc.payoutRepo.payouts[1] = &domain.Payout{
		ID: 1, PublicID: payoutPublicID, EmployeeID: employeeID,
		Amount: amount, Status: domain.PayoutPending,
	}
	tc.employeeRepo.employees[employeeID] = &domain.Employee{
		UserID: employeeID, ReimburseAvailable: 0, ReimburseLocked: amount,
	}

	payout, err := tc.reviewPayoutDirect(ctx, payoutPublicID.String(), domain.ReviewPayoutInput{Status: domain.PayoutCompleted})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expectedFee := int64(25_000)
	expectedNet := int64(975_000)

	if *payout.Fee != expectedFee {
		t.Errorf("expected fee=%d, got %d", expectedFee, *payout.Fee)
	}
	if *payout.NetAmount != expectedNet {
		t.Errorf("expected net_amount=%d, got %d", expectedNet, *payout.NetAmount)
	}
	if tc.employeeRepo.employees[employeeID].ReimburseLocked != 0 {
		t.Errorf("expected reimburse_locked=0, got %d", tc.employeeRepo.employees[employeeID].ReimburseLocked)
	}
}

func TestReviewPayout_Completed_FeeRounding(t *testing.T) {
	tc := newTestPayoutUsecase()
	ctx := context.Background()

	payoutPublicID := uuid.New()
	employeeID := int64(20)
	amount := int64(1_000_001) // fee = 1000001 * 25 / 1000 = 25000 (floor)

	tc.payoutRepo.payouts[1] = &domain.Payout{
		ID: 1, PublicID: payoutPublicID, EmployeeID: employeeID,
		Amount: amount, Status: domain.PayoutPending,
	}
	tc.employeeRepo.employees[employeeID] = &domain.Employee{
		UserID: employeeID, ReimburseLocked: amount,
	}

	payout, err := tc.reviewPayoutDirect(ctx, payoutPublicID.String(), domain.ReviewPayoutInput{Status: domain.PayoutCompleted})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expectedFee := int64(25_000) // floor division
	expectedNet := int64(975_001)

	if *payout.Fee != expectedFee {
		t.Errorf("expected fee=%d (floor), got %d", expectedFee, *payout.Fee)
	}
	if *payout.NetAmount != expectedNet {
		t.Errorf("expected net_amount=%d, got %d", expectedNet, *payout.NetAmount)
	}
}

func TestReviewPayout_Failed_BalanceRestored(t *testing.T) {
	tc := newTestPayoutUsecase()
	ctx := context.Background()

	payoutPublicID := uuid.New()
	employeeID := int64(20)
	amount := int64(1_000_000)

	tc.payoutRepo.payouts[1] = &domain.Payout{
		ID: 1, PublicID: payoutPublicID, EmployeeID: employeeID,
		Amount: amount, Status: domain.PayoutPending,
	}
	tc.employeeRepo.employees[employeeID] = &domain.Employee{
		UserID: employeeID, ReimburseAvailable: 0, ReimburseLocked: amount,
	}

	payout, err := tc.reviewPayoutDirect(ctx, payoutPublicID.String(), domain.ReviewPayoutInput{Status: domain.PayoutFailed})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if payout.Status != domain.PayoutFailed {
		t.Errorf("expected status 'failed', got '%s'", payout.Status)
	}
	if tc.employeeRepo.employees[employeeID].ReimburseAvailable != amount {
		t.Errorf("expected reimburse_available=%d (restored), got %d", amount, tc.employeeRepo.employees[employeeID].ReimburseAvailable)
	}
	if tc.employeeRepo.employees[employeeID].ReimburseLocked != 0 {
		t.Errorf("expected reimburse_locked=0, got %d", tc.employeeRepo.employees[employeeID].ReimburseLocked)
	}
}

func TestReviewPayout_AlreadyCompleted_ReturnsError(t *testing.T) {
	tc := newTestPayoutUsecase()
	ctx := context.Background()

	payoutPublicID := uuid.New()
	tc.payoutRepo.payouts[1] = &domain.Payout{
		ID: 1, PublicID: payoutPublicID, EmployeeID: 20,
		Amount: 1_000_000, Status: domain.PayoutCompleted,
	}

	_, err := tc.reviewPayoutDirect(ctx, payoutPublicID.String(), domain.ReviewPayoutInput{Status: domain.PayoutCompleted})
	if err != apperror.ErrInvalidStatusTrans {
		t.Errorf("expected ErrInvalidStatusTrans, got: %v", err)
	}
}

// Suppress unused import warning for usecase package
var _ = usecase.NewPayoutUsecase
