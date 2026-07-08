// Package postgres: JSON helpers + shared scanner interface.
package postgres

import "encoding/json"

// jsonUnmarshal wraps encoding/json.Unmarshal.
// Centralized so we can swap implementations if needed.
func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// jsonMarshal wraps encoding/json.Marshal.
func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}
