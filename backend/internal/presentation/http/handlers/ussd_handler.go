package handlers

import (
	"fmt"
	"net/http"
	"strings"
)

type USSDHandler struct {
	// dependencies for balance checking, etc.
}

func (h *USSDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	msisdn := r.FormValue("phoneNumber")
	text := r.FormValue("text") // USSD input chain (e.g. "", "1", "1*500")

	response := ""
	if text == "" {
		response = "CON Welcome to Loyalty Nexus\n1. Check Balance\n2. Regional Wars\n3. Spin Wheel"
	} else if text == "1" {
		// Mock balance retrieval
		response = "END Your Balance: 150 Pulse Points | 2 Spin Credits"
	} else if text == "2" {
		response = "END Current Leader: LAGOS (2.0X Bonus Active)"
	} else {
		response = "END Invalid Option"
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, response)
}
