package salary

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/salary"
)

// mockRepo implements RepoInterface. Only the 4 methods under test have
// configurable func fields; every other method panics to catch accidental calls.
type mockRepo struct {
	getAllSalariesFunc     func(ctx context.Context, filters dto.SalaryFilters) ([]*salary.Salary, error)
	getAllSalaryStructFunc func(ctx context.Context) ([]*salary.SalaryStructure, error)
	getAllBonusesFunc      func(ctx context.Context) ([]*salary.Bonus, error)
	getAllDeductionsFunc   func(ctx context.Context) ([]*salary.Deduction, error)
}

func (m *mockRepo) CreateSalary(context.Context, dto.CreateSalaryRequest) (int64, error) {
	panic("not implemented")
}
func (m *mockRepo) GetSalaryByID(context.Context, int64) (*salary.Salary, error) {
	panic("not implemented")
}
func (m *mockRepo) GetAllSalaries(ctx context.Context, filters dto.SalaryFilters) ([]*salary.Salary, error) {
	if m.getAllSalariesFunc != nil {
		return m.getAllSalariesFunc(ctx, filters)
	}
	return nil, nil
}
func (m *mockRepo) UpdateSalary(context.Context, int64, dto.UpdateSalaryRequest) error {
	panic("not implemented")
}
func (m *mockRepo) DeleteSalary(context.Context, int64) error {
	panic("not implemented")
}
func (m *mockRepo) UpdateSalaryCalculation(context.Context, *salary.Salary) error {
	panic("not implemented")
}
func (m *mockRepo) ApproveSalary(context.Context, int64, int64) error {
	panic("not implemented")
}
func (m *mockRepo) MarkSalaryPaid(context.Context, int64) error {
	panic("not implemented")
}
func (m *mockRepo) GetActiveSalaryStructure(context.Context, int64, string) (*salary.SalaryStructure, error) {
	panic("not implemented")
}
func (m *mockRepo) GetSalaryStructureByEmployee(context.Context, int64) ([]*salary.SalaryStructure, error) {
	panic("not implemented")
}
func (m *mockRepo) GetAllSalaryStructures(ctx context.Context) ([]*salary.SalaryStructure, error) {
	if m.getAllSalaryStructFunc != nil {
		return m.getAllSalaryStructFunc(ctx)
	}
	return nil, nil
}
func (m *mockRepo) CreateBonuses(context.Context, int64, []dto.BonusInput) error {
	panic("not implemented")
}
func (m *mockRepo) CreateDeductions(context.Context, int64, []dto.DeductionInput) error {
	panic("not implemented")
}
func (m *mockRepo) GetBonuses(context.Context, int64) ([]*salary.Bonus, error) {
	panic("not implemented")
}
func (m *mockRepo) GetDeductions(context.Context, int64) ([]*salary.Deduction, error) {
	panic("not implemented")
}
func (m *mockRepo) GetAllBonuses(ctx context.Context) ([]*salary.Bonus, error) {
	if m.getAllBonusesFunc != nil {
		return m.getAllBonusesFunc(ctx)
	}
	return nil, nil
}
func (m *mockRepo) GetAllDeductions(ctx context.Context) ([]*salary.Deduction, error) {
	if m.getAllDeductionsFunc != nil {
		return m.getAllDeductionsFunc(ctx)
	}
	return nil, nil
}
func (m *mockRepo) GetActiveEmployeesByDepartment(context.Context, *int64) ([]int64, error) {
	panic("not implemented")
}
func (m *mockRepo) SalaryExists(context.Context, int64, int, int) (bool, error) {
	panic("not implemented")
}

func newTestService(repo *mockRepo) *Service {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewService(repo, log)
}

func TestGetAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			getAllSalariesFunc: func(_ context.Context, _ dto.SalaryFilters) ([]*salary.Salary, error) {
				return []*salary.Salary{{ID: 1}, {ID: 2}}, nil
			},
		}
		svc := newTestService(repo)

		result, err := svc.GetAll(context.Background(), dto.SalaryFilters{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("len = %d, want 2", len(result))
		}
	})

	t.Run("nil from repo → empty slice", func(t *testing.T) {
		repo := &mockRepo{
			getAllSalariesFunc: func(_ context.Context, _ dto.SalaryFilters) ([]*salary.Salary, error) {
				return nil, nil
			},
		}
		svc := newTestService(repo)

		result, err := svc.GetAll(context.Background(), dto.SalaryFilters{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil slice, got nil")
		}
		if len(result) != 0 {
			t.Errorf("len = %d, want 0", len(result))
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &mockRepo{
			getAllSalariesFunc: func(_ context.Context, _ dto.SalaryFilters) ([]*salary.Salary, error) {
				return nil, errors.New("db error")
			},
		}
		svc := newTestService(repo)

		_, err := svc.GetAll(context.Background(), dto.SalaryFilters{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("filters passthrough", func(t *testing.T) {
		var captured dto.SalaryFilters
		empID := int64(42)
		repo := &mockRepo{
			getAllSalariesFunc: func(_ context.Context, filters dto.SalaryFilters) ([]*salary.Salary, error) {
				captured = filters
				return []*salary.Salary{}, nil
			},
		}
		svc := newTestService(repo)

		_, err := svc.GetAll(context.Background(), dto.SalaryFilters{EmployeeID: &empID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if captured.EmployeeID == nil {
			t.Fatal("expected EmployeeID to be set")
		}
		if *captured.EmployeeID != 42 {
			t.Errorf("EmployeeID = %d, want 42", *captured.EmployeeID)
		}
	})
}

func TestGetAllStructures(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			getAllSalaryStructFunc: func(_ context.Context) ([]*salary.SalaryStructure, error) {
				return []*salary.SalaryStructure{{ID: 1}}, nil
			},
		}
		svc := newTestService(repo)

		result, err := svc.GetAllStructures(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("len = %d, want 1", len(result))
		}
	})

	t.Run("nil → empty slice", func(t *testing.T) {
		repo := &mockRepo{
			getAllSalaryStructFunc: func(_ context.Context) ([]*salary.SalaryStructure, error) {
				return nil, nil
			},
		}
		svc := newTestService(repo)

		result, err := svc.GetAllStructures(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil slice, got nil")
		}
		if len(result) != 0 {
			t.Errorf("len = %d, want 0", len(result))
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &mockRepo{
			getAllSalaryStructFunc: func(_ context.Context) ([]*salary.SalaryStructure, error) {
				return nil, errors.New("db error")
			},
		}
		svc := newTestService(repo)

		_, err := svc.GetAllStructures(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetAllBonuses(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			getAllBonusesFunc: func(_ context.Context) ([]*salary.Bonus, error) {
				return []*salary.Bonus{{ID: 1}}, nil
			},
		}
		svc := newTestService(repo)

		result, err := svc.GetAllBonuses(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("len = %d, want 1", len(result))
		}
	})

	t.Run("nil → empty slice", func(t *testing.T) {
		repo := &mockRepo{
			getAllBonusesFunc: func(_ context.Context) ([]*salary.Bonus, error) {
				return nil, nil
			},
		}
		svc := newTestService(repo)

		result, err := svc.GetAllBonuses(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil slice, got nil")
		}
		if len(result) != 0 {
			t.Errorf("len = %d, want 0", len(result))
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &mockRepo{
			getAllBonusesFunc: func(_ context.Context) ([]*salary.Bonus, error) {
				return nil, errors.New("db error")
			},
		}
		svc := newTestService(repo)

		_, err := svc.GetAllBonuses(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetAllDeductions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			getAllDeductionsFunc: func(_ context.Context) ([]*salary.Deduction, error) {
				return []*salary.Deduction{{ID: 1}}, nil
			},
		}
		svc := newTestService(repo)

		result, err := svc.GetAllDeductions(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("len = %d, want 1", len(result))
		}
	})

	t.Run("nil → empty slice", func(t *testing.T) {
		repo := &mockRepo{
			getAllDeductionsFunc: func(_ context.Context) ([]*salary.Deduction, error) {
				return nil, nil
			},
		}
		svc := newTestService(repo)

		result, err := svc.GetAllDeductions(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil slice, got nil")
		}
		if len(result) != 0 {
			t.Errorf("len = %d, want 0", len(result))
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &mockRepo{
			getAllDeductionsFunc: func(_ context.Context) ([]*salary.Deduction, error) {
				return nil, errors.New("db error")
			},
		}
		svc := newTestService(repo)

		_, err := svc.GetAllDeductions(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
