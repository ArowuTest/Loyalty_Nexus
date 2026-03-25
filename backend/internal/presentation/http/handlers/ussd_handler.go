package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"loyalty-nexus/internal/domain/repositories"
)

type USSDHandler struct {
	userRepo repositories.UserRepository
}

func NewUSSDHandler(ur repositories.UserRepository) *USSDHandler {
	return &USSDHandler{userRepo: ur}
}

func (h *USSDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// ... (rest of the logic)
	msisdn := r.FormValue("phoneNumber")
	text := r.FormValue("text") 

	parts := strings.Split(text, "*")
	response := ""

	if text == "" {
		response = "CON Welcome to Loyalty Nexus\n1. My Balance\n2. Regional Wars\n3. Spin Wheel\n4. Link MoMo"
	} else {
		switch parts[0] {
		case "1":
			// My Balance
			response = "END Your Balance:\nPulse Points: 150\nSpin Credits: 2\nTier: PLATINUM"
		case "2":
			// Regional Wars (Innovation 4)
			response = "END Tournament Leader: LAGOS\nYour Region: LAGOS (2.0X Multiplier Active)"
		case "3":
			// Spin Wheel
			response = "END You won 500MB Data! Credited to your line."
		case "4":
			// Link MoMo (REQ-1.3)
			if len(parts) == 1 {
				response = "CON Enter MTN MoMo Number:"
			} else {
				response = "END Request Accepted. You will receive an SMS confirmation."
			}
		default:
			response = "END Invalid option."
		}
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, response)
}
