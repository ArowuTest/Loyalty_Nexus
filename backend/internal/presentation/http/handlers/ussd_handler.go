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
	text := r.FormValue("text") 

	response := ""
	parts := strings.Split(text, "*")

	if text == "" {
		response = "CON Nexus Rewards\n1. My Balance\n2. Redeem Points\n3. Spin Wheel\n4. Regional Wars"
	} else if parts[0] == "1" {
		// My Balance
		response = "END Your Balance:\nPulse Points: 150\nSpin Credits: 2"
	} else if parts[0] == "2" {
		// Redeem Points (REQ-3.4)
		if len(parts) == 1 {
			response = "CON Select Reward:\n1. 100MB Data (50 pts)\n2. N100 Airtime (100 pts)\n3. N500 Airtime (450 pts)"
		} else {
			response = "END Request Accepted. You will receive an SMS confirmation shortly."
		}
	} else if parts[0] == "3" {
		// Spin Wheel
		response = "END You won N200 Airtime! Credited to your line."
	} else if parts[0] == "4" {
		// Regional Wars
		response = "END Tournament Leader: LAGOS\nYour Region: LAGOS (2.0X Multiplier Active)"
	} else {
		response = "END Invalid Option"
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, response)
}
