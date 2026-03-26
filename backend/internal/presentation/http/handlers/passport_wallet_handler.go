package handlers

// passport_wallet_handler.go — Additional PassportHandler methods for wallet pass
// download/save URLs and Apple Wallet web service callbacks.
//
// Routes registered in main.go:
//   GET  /api/v1/passport/wallet-urls
//   POST /api/v1/passport/apple/v1/devices/{deviceID}/registrations/{passTypeID}/{serialNumber}
//   DELETE /api/v1/passport/apple/v1/devices/{deviceID}/registrations/{passTypeID}/{serialNumber}
//   GET  /api/v1/passport/apple/v1/devices/{deviceID}/registrations/{passTypeID}

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// ─── GET /api/v1/passport/wallet-urls ────────────────────────────────────────

// GetWalletPassURLs returns both the Apple .pkpass download URL and the
// Google Wallet "Add to Google Wallet" save URL for the authenticated user.
func (h *PassportHandler) GetWalletPassURLs(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)

	urls, err := h.passportSvc.GetWalletPassURLs(r.Context(), userID)
	if err != nil {
		log.Printf("[Passport] GetWalletPassURLs error for %s: %v", userID, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build wallet URLs"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"apple_pkpass_url":      urls.ApplePKPassURL,
		"google_wallet_save_url": urls.GoogleWalletURL,
	})
}

// ─── Apple Wallet Web Service Callbacks ──────────────────────────────────────
// These endpoints are called by iOS when a user adds or removes the pass from
// Apple Wallet. They are public (no user JWT) but the Apple-generated auth token
// in the Authorization header is validated by the service layer.

// RegisterAppleDevice is called by iOS when a user adds the pass to Apple Wallet.
// POST /api/v1/passport/apple/v1/devices/{deviceID}/registrations/{passTypeID}/{serialNumber}
func (h *PassportHandler) RegisterAppleDevice(w http.ResponseWriter, r *http.Request) {
	deviceID    := r.PathValue("deviceID")
	serialNumber := r.PathValue("serialNumber")

	var body struct {
		PushToken string `json:"pushToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PushToken == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.passportSvc.RegisterAppleDevice(r.Context(), deviceID, body.PushToken, serialNumber); err != nil {
		log.Printf("[Passport] RegisterAppleDevice error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// UnregisterAppleDevice is called by iOS when a user removes the pass from Apple Wallet.
// DELETE /api/v1/passport/apple/v1/devices/{deviceID}/registrations/{passTypeID}/{serialNumber}
func (h *PassportHandler) UnregisterAppleDevice(w http.ResponseWriter, r *http.Request) {
	deviceID     := r.PathValue("deviceID")
	serialNumber := r.PathValue("serialNumber")

	if err := h.passportSvc.UnregisterAppleDevice(r.Context(), deviceID, serialNumber); err != nil {
		log.Printf("[Passport] UnregisterAppleDevice error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetUpdatedSerials is called by iOS to check if a pass has been updated since
// the last sync. Returns the serial numbers of any passes that have changed.
// GET /api/v1/passport/apple/v1/devices/{deviceID}/registrations/{passTypeID}
func (h *PassportHandler) GetUpdatedSerials(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("deviceID")

	// Parse If-Modified-Since header (Apple sends this on subsequent polls)
	var since time.Time
	if ims := r.Header.Get("If-Modified-Since"); ims != "" {
		if t, err := http.ParseTime(ims); err == nil {
			since = t
		}
	}
	if since.IsZero() {
		since = time.Now().Add(-24 * time.Hour) // default: last 24h
	}

	serials, lastUpdated, err := h.passportSvc.GetUpdatedSerials(r.Context(), deviceID, since)
	if err != nil {
		w.WriteHeader(http.StatusNoContent) // 204 = no updates
		return
	}
	if len(serials) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Last-Modified", lastUpdated.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"serialNumbers": serials,
		"lastUpdated":   lastUpdated,
	})
}
