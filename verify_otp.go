package main

import (
	"encoding/json"
	"net/http"
)

type verifyOTPRequest struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

func verifyOTPHandler(store *otpStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req verifyOTPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		if !isValidEmail(req.Email) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid email"})
			return
		}

		if !isValidOTP(req.OTP) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid otp"})
			return
		}

		if !store.verify(req.Email, req.OTP) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid otp"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]bool{"verified": true})
	}
}

func isValidOTP(otp string) bool {
	if len(otp) != 6 {
		return false
	}
	for i := range len(otp) {
		if otp[i] < '0' || otp[i] > '9' {
			return false
		}
	}
	return true
}
