package handlers

import (
	"encoding/json"
	"net/http"
)

func writeJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload) //nolint:errcheck // response write failures are non-actionable
}

// decodeJSON decodes the request body into v.
func decodeJSON(r *http.Request, v interface{}) error {
	defer func() { _ = r.Body.Close() }() // close error is non-actionable on request bodies
	return json.NewDecoder(r.Body).Decode(v)
}
