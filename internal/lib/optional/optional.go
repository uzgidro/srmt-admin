// Package optional provides a three-state generic wrapper for JSON fields.
//
// Optional[T] distinguishes between three states a JSON field can have:
//   - absent          (field not present in JSON)        → Set = false, Value = nil
//   - explicit null   (field present, value is null)     → Set = true,  Value = nil
//   - actual value    (field present with value v)       → Set = true,  Value = &v
//
// This is the contract used by partial-update endpoints to tell "leave this
// column alone" (absent) from "set this column to NULL" (null) from "write
// this concrete value" (number).
package optional

import "encoding/json"

// Optional represents a JSON field that can be absent, null, or hold a value.
type Optional[T any] struct {
	Value *T
	Set   bool
}

// UnmarshalJSON implements custom decoding so Optional can detect whether
// the field was present in the input. The encoding/json decoder calls this
// only when the field IS present, so any call sets Set=true; absent fields
// leave Set at its zero value (false).
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}
