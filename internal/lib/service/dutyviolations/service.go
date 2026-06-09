// Package dutyviolations holds the business logic for duty-officer
// violation records. The service orchestrates the repo around the
// record↔files invariant ("the request body owns the full file list")
// that would be wrong to leak into either repo or handler.
package dutyviolations

import (
	"context"
	"fmt"

	dvmodel "srmt-admin/internal/lib/model/duty-violations"
)

// Repository describes the storage-level methods the service depends on.
// Defined here (not in the repo package) so handler/test code can satisfy
// it with a thin mock and the service stays decoupled from *Repo.
//
// The *WithFiles methods are transactional — both halves (record + file
// links) succeed together or neither commits. This is the contract the
// service relies on for the "record↔files" invariant; do not replace
// them with the non-transactional Add/Edit + Link/Unlink pair.
type Repository interface {
	AddDutyViolationWithFiles(ctx context.Context, req dvmodel.CreateRequest, createdByUserID int64) (int64, error)
	UpdateDutyViolationWithFiles(ctx context.Context, id int64, req dvmodel.UpdateRequest) error
	GetDutyViolations(ctx context.Context, f dvmodel.ListFilter) ([]*dvmodel.DutyViolation, error)
	GetDutyViolationByID(ctx context.Context, id int64) (*dvmodel.DutyViolation, error)
	DeleteDutyViolation(ctx context.Context, id int64) error
}

// Service is the duty-violations business layer.
type Service struct {
	repo Repository
}

// NewService wires the Service to a Repository implementation.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create inserts a new record together with its file links atomically.
// The repo wraps the INSERT and the junction inserts in one transaction —
// on any failure, the record is rolled back and the caller sees only the
// error (no orphan rows). The fresh record is loaded for the response so
// the frontend gets org name + file metadata + timestamps in one round-trip.
func (s *Service) Create(ctx context.Context, req dvmodel.CreateRequest, createdByUserID int64) (*dvmodel.DutyViolation, error) {
	const op = "service.dutyviolations.Create"

	id, err := s.repo.AddDutyViolationWithFiles(ctx, req, createdByUserID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return s.repo.GetDutyViolationByID(ctx, id)
}

// List passes the filter through to the repo unchanged. It exists on the
// service only to give the handler a single dependency surface (Service)
// instead of repo+service mixed.
func (s *Service) List(ctx context.Context, f dvmodel.ListFilter) ([]*dvmodel.DutyViolation, error) {
	const op = "service.dutyviolations.List"
	out, err := s.repo.GetDutyViolations(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return out, nil
}

// Update replaces the record's content AND its full file list atomically.
// The repo performs UPDATE + DELETE old links + INSERT new links inside
// one transaction. On any failure, the prior state is intact.
//
// req.FileIDs is the authoritative new list, not a delta — pass [] to
// detach every file, pass [...old, new] to add one.
func (s *Service) Update(ctx context.Context, id int64, req dvmodel.UpdateRequest) (*dvmodel.DutyViolation, error) {
	const op = "service.dutyviolations.Update"

	if err := s.repo.UpdateDutyViolationWithFiles(ctx, id, req); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return s.repo.GetDutyViolationByID(ctx, id)
}

// Delete removes the record. ON DELETE CASCADE on the junction table
// removes the file links; files in storage are intentionally retained
// (they may be referenced elsewhere; cleanup is a separate concern).
func (s *Service) Delete(ctx context.Context, id int64) error {
	const op = "service.dutyviolations.Delete"
	if err := s.repo.DeleteDutyViolation(ctx, id); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
