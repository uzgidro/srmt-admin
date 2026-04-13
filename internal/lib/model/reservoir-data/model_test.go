package reservoirdata

import (
	"encoding/json"
	"testing"

	optional "srmt-admin/internal/lib/optional"
)

func TestReservoirDataItem_OptionalFields_Absent(t *testing.T) {
	const body = `{"organization_id":1,"date":"2026-04-12"}`

	var item ReservoirDataItem
	if err := json.Unmarshal([]byte(body), &item); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	cases := []struct {
		name string
		got  optional.Optional[float64]
	}{
		{"income", item.Income},
		{"level", item.Level},
		{"release", item.Release},
		{"volume", item.Volume},
	}
	for _, c := range cases {
		if c.got.Set {
			t.Errorf("%s: Set=true, want false (field absent)", c.name)
		}
		if c.got.Value != nil {
			t.Errorf("%s: Value=%v, want nil", c.name, *c.got.Value)
		}
	}
}

func TestReservoirDataItem_OptionalFields_Null(t *testing.T) {
	const body = `{"organization_id":1,"date":"2026-04-12","income":null,"level":null,"release":null,"volume":null}`

	var item ReservoirDataItem
	if err := json.Unmarshal([]byte(body), &item); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	cases := []struct {
		name string
		got  optional.Optional[float64]
	}{
		{"income", item.Income},
		{"level", item.Level},
		{"release", item.Release},
		{"volume", item.Volume},
	}
	for _, c := range cases {
		if !c.got.Set {
			t.Errorf("%s: Set=false, want true (field explicitly null)", c.name)
		}
		if c.got.Value != nil {
			t.Errorf("%s: Value=%v, want nil (null payload)", c.name, *c.got.Value)
		}
	}
}

func TestReservoirDataItem_OptionalFields_Value(t *testing.T) {
	const body = `{"organization_id":1,"date":"2026-04-12","income":1.5,"level":2.5,"release":3.5,"volume":4.5}`

	var item ReservoirDataItem
	if err := json.Unmarshal([]byte(body), &item); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	cases := []struct {
		name string
		got  optional.Optional[float64]
		want float64
	}{
		{"income", item.Income, 1.5},
		{"level", item.Level, 2.5},
		{"release", item.Release, 3.5},
		{"volume", item.Volume, 4.5},
	}
	for _, c := range cases {
		if !c.got.Set {
			t.Errorf("%s: Set=false, want true", c.name)
		}
		if c.got.Value == nil {
			t.Fatalf("%s: Value=nil, want %v", c.name, c.want)
		}
		if *c.got.Value != c.want {
			t.Errorf("%s: Value=%v, want %v", c.name, *c.got.Value, c.want)
		}
	}
}

func TestReservoirDataItem_OptionalFields_ZeroValueIsNotAbsent(t *testing.T) {
	const body = `{"organization_id":1,"date":"2026-04-12","income":0,"level":0,"release":0,"volume":0}`

	var item ReservoirDataItem
	if err := json.Unmarshal([]byte(body), &item); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	fields := []struct {
		name string
		got  optional.Optional[float64]
	}{
		{"income", item.Income},
		{"level", item.Level},
		{"release", item.Release},
		{"volume", item.Volume},
	}
	for _, f := range fields {
		if !f.got.Set {
			t.Errorf("%s: Set=false, want true (0 must be treated as a real value)", f.name)
		}
		if f.got.Value == nil || *f.got.Value != 0 {
			t.Errorf("%s: Value=%v, want 0", f.name, f.got.Value)
		}
	}
}
