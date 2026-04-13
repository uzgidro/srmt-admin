package optional

import (
	"encoding/json"
	"testing"
)

type wrapper struct {
	X Optional[float64] `json:"x"`
}

func TestOptional_AbsentField(t *testing.T) {
	var w wrapper
	if err := json.Unmarshal([]byte(`{}`), &w); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if w.X.Set {
		t.Errorf("Set: got true, want false")
	}
	if w.X.Value != nil {
		t.Errorf("Value: got %v, want nil", w.X.Value)
	}
}

func TestOptional_NullField(t *testing.T) {
	var w wrapper
	if err := json.Unmarshal([]byte(`{"x": null}`), &w); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !w.X.Set {
		t.Errorf("Set: got false, want true")
	}
	if w.X.Value != nil {
		t.Errorf("Value: got %v, want nil", w.X.Value)
	}
}

func TestOptional_ValueField(t *testing.T) {
	var w wrapper
	if err := json.Unmarshal([]byte(`{"x": 42.5}`), &w); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !w.X.Set {
		t.Errorf("Set: got false, want true")
	}
	if w.X.Value == nil {
		t.Fatal("Value: got nil, want pointer")
	}
	if *w.X.Value != 42.5 {
		t.Errorf("Value: got %v, want 42.5", *w.X.Value)
	}
}

func TestOptional_ZeroValueIsNotAbsent(t *testing.T) {
	var w wrapper
	if err := json.Unmarshal([]byte(`{"x": 0}`), &w); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !w.X.Set {
		t.Errorf("Set: got false, want true (explicit zero must NOT look like absent)")
	}
	if w.X.Value == nil {
		t.Fatal("Value: got nil, want pointer to 0")
	}
	if *w.X.Value != 0 {
		t.Errorf("Value: got %v, want 0", *w.X.Value)
	}
}
