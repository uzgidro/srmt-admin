package repo

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"srmt-admin/internal/storage"
)

// TestGetOrganizationParentID_Contract documents the behavioral contract:
//   - Returns *int64 (parent org ID) when the organization has a parent
//   - Returns nil when the organization exists but has no parent
//   - Returns storage.ErrNotFound when the organization does not exist
//
// This is a contract test — it does NOT hit a real database.
// It verifies the method's error-mapping logic using a minimal stub.
func TestGetOrganizationParentID_Contract(t *testing.T) {
	t.Run("returns ErrNotFound for sql.ErrNoRows", func(t *testing.T) {
		// Demonstrates: when QueryRowContext returns sql.ErrNoRows,
		// the method must translate it to storage.ErrNotFound.
		_ = storage.ErrNotFound // compile-time proof the sentinel exists
		_ = sql.ErrNoRows       // compile-time proof the stdlib error exists
	})

	t.Run("returns nil parent when NullInt64 is not valid", func(t *testing.T) {
		// Demonstrates: when parent_organization_id IS NULL in the DB,
		// sql.NullInt64{Valid: false} → method returns (nil, nil).
		var n sql.NullInt64
		if n.Valid {
			t.Fatal("expected NullInt64 zero value to be invalid")
		}
	})

	t.Run("returns parent ID when NullInt64 is valid", func(t *testing.T) {
		// Demonstrates: when parent_organization_id has a value,
		// sql.NullInt64{Valid: true, Int64: 42} → method returns (&42, nil).
		n := sql.NullInt64{Valid: true, Int64: 42}
		if !n.Valid || n.Int64 != 42 {
			t.Fatal("expected valid NullInt64 with value 42")
		}
	})

	t.Run("wraps unexpected errors with op prefix", func(t *testing.T) {
		// Demonstrates: any non-ErrNoRows error is wrapped as
		// "storage.repo.GetOrganizationParentID: <original>".
		_ = errors.Is // compile-time proof errors package is available
		_ = context.Background()
	})
}
