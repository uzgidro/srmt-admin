package dutyviolations

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	dvmodel "srmt-admin/internal/lib/model/duty-violations"
)

// --- mock Repository ---

type mockRepo struct {
	addID         int64
	addErr        error
	addReq        dvmodel.CreateRequest
	addUserID     int64
	addCalled     bool

	updateErr    error
	updateCalls  int
	updateGotReq dvmodel.UpdateRequest

	getByIDFn    func(ctx context.Context, id int64) (*dvmodel.DutyViolation, error)
	getByIDCalls []int64

	deleteErr   error
	deleteCalls int

	listResult    []dvmodel.OrgGroup
	listErr       error
	listGotFilter dvmodel.ListFilter
}

func (m *mockRepo) AddDutyViolationWithFiles(_ context.Context, req dvmodel.CreateRequest, userID int64) (int64, error) {
	m.addCalled = true
	m.addReq = req
	m.addUserID = userID
	return m.addID, m.addErr
}

func (m *mockRepo) UpdateDutyViolationWithFiles(_ context.Context, _ int64, req dvmodel.UpdateRequest) error {
	m.updateCalls++
	m.updateGotReq = req
	return m.updateErr
}

func (m *mockRepo) GetDutyViolations(_ context.Context, f dvmodel.ListFilter) ([]dvmodel.OrgGroup, error) {
	m.listGotFilter = f
	return m.listResult, m.listErr
}

func (m *mockRepo) GetDutyViolationByID(ctx context.Context, id int64) (*dvmodel.DutyViolation, error) {
	m.getByIDCalls = append(m.getByIDCalls, id)
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &dvmodel.DutyViolation{ID: id}, nil
}

func (m *mockRepo) DeleteDutyViolation(_ context.Context, _ int64) error {
	m.deleteCalls++
	return m.deleteErr
}

// --- helpers ---

func validReq() dvmodel.CreateRequest {
	end := time.Date(2026, 6, 8, 20, 0, 0, 0, time.UTC)
	return dvmodel.CreateRequest{
		OrganizationID:  103,
		StartTime:       time.Date(2026, 6, 8, 8, 0, 0, 0, time.UTC),
		EndTime:         &end,
		DutyOfficerName: "Иванов И.И.",
		Reason:          "Прогул",
	}
}

func validUpdateReq() dvmodel.UpdateRequest {
	return dvmodel.UpdateRequest(validReq())
}

// --- Create ---

// Service forwards the request (including file_ids) to the transactional
// repo method, then calls GetByID for the response payload. This is the
// happy-path contract the frontend depends on.
func TestCreate_ForwardsFilesToTxMethod_AndReturnsFreshRecord(t *testing.T) {
	repo := &mockRepo{addID: 42}
	svc := NewService(repo)
	req := validReq()
	req.FileIDs = []int64{1, 2, 3}

	dv, err := svc.Create(context.Background(), req, 7)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if dv.ID != 42 {
		t.Errorf("returned ID: want 42, got %d", dv.ID)
	}
	if !repo.addCalled || repo.addUserID != 7 {
		t.Errorf("AddWithFiles not called with userID=7: called=%v userID=%d", repo.addCalled, repo.addUserID)
	}
	if !reflect.DeepEqual(repo.addReq.FileIDs, []int64{1, 2, 3}) {
		t.Errorf("file_ids not forwarded: got %v", repo.addReq.FileIDs)
	}
	if len(repo.getByIDCalls) != 1 || repo.getByIDCalls[0] != 42 {
		t.Errorf("GetByID(42) must be called to return fresh record, got %v", repo.getByIDCalls)
	}
}

// Empty file_ids must still reach the repo (the repo's tx skips the
// junction INSERT internally). Tests that the service is a pass-through,
// not gating on FileIDs length.
func TestCreate_EmptyFileIDsStillReachesRepo(t *testing.T) {
	repo := &mockRepo{addID: 42}
	svc := NewService(repo)

	if _, err := svc.Create(context.Background(), validReq(), 7); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !repo.addCalled {
		t.Error("repo must be called even without files")
	}
	if len(repo.addReq.FileIDs) != 0 {
		t.Errorf("empty file_ids must stay empty: got %v", repo.addReq.FileIDs)
	}
}

// AddWithFiles fails: GetByID must NOT be called — there's no row to
// fetch. Documents the abort-on-error contract.
func TestCreate_AddError_NoGetByID(t *testing.T) {
	repo := &mockRepo{addErr: errors.New("db down")}
	svc := NewService(repo)
	req := validReq()
	req.FileIDs = []int64{1}

	if _, err := svc.Create(context.Background(), req, 7); err == nil {
		t.Fatal("want error, got nil")
	}
	if len(repo.getByIDCalls) != 0 {
		t.Errorf("GetByID must not run on Add failure; got %v", repo.getByIDCalls)
	}
}

// --- Update ---

// PATCH delegates to UpdateWithFiles (one atomic op) and then re-loads
// the record for the response.
func TestUpdate_DelegatesToTxMethod_AndReloads(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)
	req := validUpdateReq()
	req.FileIDs = []int64{10, 20}

	if _, err := svc.Update(context.Background(), 5, req); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if repo.updateCalls != 1 {
		t.Errorf("UpdateWithFiles must run once, got %d", repo.updateCalls)
	}
	if !reflect.DeepEqual(repo.updateGotReq.FileIDs, []int64{10, 20}) {
		t.Errorf("file_ids not forwarded: %v", repo.updateGotReq.FileIDs)
	}
	if len(repo.getByIDCalls) != 1 || repo.getByIDCalls[0] != 5 {
		t.Errorf("GetByID(5) must be called for response, got %v", repo.getByIDCalls)
	}
}

// Empty file_ids on PATCH means "detach everything". The service forwards
// as-is; the repo's tx will run the DELETE and skip the INSERT.
func TestUpdate_EmptyFileIDsForwarded(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)
	req := validUpdateReq()
	req.FileIDs = nil

	if _, err := svc.Update(context.Background(), 5, req); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(repo.updateGotReq.FileIDs) != 0 {
		t.Errorf("empty FileIDs must stay empty on the way to the repo: %v", repo.updateGotReq.FileIDs)
	}
}

// UpdateWithFiles fails: GetByID must NOT run. The frontend gets the
// raw error from the repo (e.g. ErrNotFound) without a stale read.
func TestUpdate_UpdateError_NoGetByID(t *testing.T) {
	repo := &mockRepo{updateErr: errors.New("db down")}
	svc := NewService(repo)
	req := validUpdateReq()
	req.FileIDs = []int64{1}

	if _, err := svc.Update(context.Background(), 5, req); err == nil {
		t.Fatal("want error, got nil")
	}
	if len(repo.getByIDCalls) != 0 {
		t.Errorf("GetByID must not run on Update failure: %v", repo.getByIDCalls)
	}
}

// --- Delete + List ---

func TestDelete_PassesThrough(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)

	if err := svc.Delete(context.Background(), 99); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if repo.deleteCalls != 1 {
		t.Errorf("Delete must call repo exactly once, got %d", repo.deleteCalls)
	}
}

// List is a pass-through: filter goes to the repo unchanged, the OrgGroup
// payload comes back as-is. Tests both halves with a 2-group fixture.
func TestList_PassesFilterAndReturnsGroups(t *testing.T) {
	orgID := int64(7)
	repo := &mockRepo{listResult: []dvmodel.OrgGroup{
		{ID: 7, Name: "Андижон", Violations: []dvmodel.DutyViolation{{ID: 1}, {ID: 2}}},
		{ID: 8, Name: "Чарвак", Violations: []dvmodel.DutyViolation{{ID: 3}}},
	}}
	svc := NewService(repo)
	got, err := svc.List(context.Background(), dvmodel.ListFilter{OrganizationID: &orgID})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 || got[0].ID != 7 || got[1].ID != 8 {
		t.Errorf("groups not forwarded: %+v", got)
	}
	if repo.listGotFilter.OrganizationID == nil || *repo.listGotFilter.OrganizationID != 7 {
		t.Errorf("filter not forwarded correctly: %+v", repo.listGotFilter)
	}
}
