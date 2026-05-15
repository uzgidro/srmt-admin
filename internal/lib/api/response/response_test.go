package response

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestBadRequestStructured_JSONShape pins the wire format of the structured
// 400-error helper. Frontend relies on three keys: `error` (human message,
// kept for backwards compatibility with existing handlers), `code` (stable
// machine identifier for localization/branching), and `details` (array of
// free-form objects describing the violation per item).
//
// Empty/zero `details` MUST omit the key entirely (omitempty) so old clients
// don't see a confusing `"details": null`.
func TestBadRequestStructured_JSONShape(t *testing.T) {
	r := BadRequestStructured(
		"save.field_negative",
		"consumption_m3_s must be >= 0",
		[]Detail{
			{"organization_id": int64(16), "field": "consumption_m3_s", "value": -1.5},
		},
	)
	if r.Status != http.StatusBadRequest {
		t.Errorf("Status: want 400, got %d", r.Status)
	}

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got["error"] != "consumption_m3_s must be >= 0" {
		t.Errorf("error: want %q, got %v", "consumption_m3_s must be >= 0", got["error"])
	}
	if got["code"] != "save.field_negative" {
		t.Errorf("code: want %q, got %v", "save.field_negative", got["code"])
	}
	details, ok := got["details"].([]any)
	if !ok || len(details) != 1 {
		t.Fatalf("details: want array of len 1, got %T %v", got["details"], got["details"])
	}
	first, ok := details[0].(map[string]any)
	if !ok {
		t.Fatalf("details[0]: want object, got %T", details[0])
	}
	if first["field"] != "consumption_m3_s" {
		t.Errorf("details[0].field: want %q, got %v", "consumption_m3_s", first["field"])
	}
	// Status must NOT appear on the wire (json:"-").
	if _, has := got["Status"]; has {
		t.Error("Status leaked to JSON")
	}
}

// TestBadRequest_BackwardsCompatibility ensures the legacy plain helper still
// produces the original shape — only `error`, no `code`/`details` fields.
// Adding `Code`/`Details` to the Response struct must NOT introduce empty
// keys for callers that didn't ask for them.
func TestBadRequest_BackwardsCompatibility(t *testing.T) {
	r := BadRequest("plain message")

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["error"] != "plain message" {
		t.Errorf("error: want %q, got %v", "plain message", got["error"])
	}
	if _, has := got["code"]; has {
		t.Errorf("code key must be omitted when empty, got: %v", got["code"])
	}
	if _, has := got["details"]; has {
		t.Errorf("details key must be omitted when empty, got: %v", got["details"])
	}
}
