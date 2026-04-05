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
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// jsonOK writes a 200 JSON response (alias for writeJSON with http.StatusOK).
func jsonOK(w http.ResponseWriter, payload interface{}) {
	writeJSON(w, http.StatusOK, payload)
}

// jsonError writes an error JSON response with the given status code.
func jsonError(w http.ResponseWriter, message string, code int) {
	writeJSON(w, code, map[string]string{"error": message})
}
