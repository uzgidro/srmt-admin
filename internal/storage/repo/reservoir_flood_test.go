package repo

import (
	"testing"

	"srmt-admin/internal/lib/optional"
)

// blankToNull must collapse an explicitly-set empty string to NULL so the
// all-NULL prune trigger can drop a row whose only value is "". Absent fields
// and real values must be left untouched.
func TestBlankToNull(t *testing.T) {
	empty := ""
	value := "Иванов И.И."

	cases := []struct {
		name     string
		in       optional.Optional[string]
		wantNil  bool
		wantSet  bool
	}{
		{
			name:    "explicit empty string -> nil",
			in:      optional.Optional[string]{Set: true, Value: &empty},
			wantNil: true,
			wantSet: true,
		},
		{
			name:    "absent field -> untouched",
			in:      optional.Optional[string]{Set: false, Value: nil},
			wantNil: true,
			wantSet: false,
		},
		{
			name:    "explicit null -> untouched",
			in:      optional.Optional[string]{Set: true, Value: nil},
			wantNil: true,
			wantSet: true,
		},
		{
			name:    "real value -> untouched",
			in:      optional.Optional[string]{Set: true, Value: &value},
			wantNil: false,
			wantSet: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := tc.in
			blankToNull(&o)
			if (o.Value == nil) != tc.wantNil {
				t.Errorf("Value nil: want %v, got %v", tc.wantNil, o.Value == nil)
			}
			if o.Set != tc.wantSet {
				t.Errorf("Set: want %v, got %v", tc.wantSet, o.Set)
			}
			if !tc.wantNil && o.Value != nil && *o.Value != value {
				t.Errorf("Value: real value must be preserved, got %q", *o.Value)
			}
		})
	}
}
