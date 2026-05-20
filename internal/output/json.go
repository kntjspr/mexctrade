package output

import (
	"encoding/json"
	"io"
)

// WriteJSON encodes v as a single JSON object on its own line.
func WriteJSON(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}

// WriteJSONError emits a structured error envelope: {"error":...,"code":...,"exit":N}.
func WriteJSONError(w io.Writer, code, message string, exit int) error {
	return json.NewEncoder(w).Encode(map[string]any{
		"error": message,
		"code":  code,
		"exit":  exit,
	})
}
