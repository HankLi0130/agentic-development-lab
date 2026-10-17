package main

import (
	"net/http"
)

type verifyOTPRequest struct {
	emailRequest
	OTP string `json:"otp"`
}

func verifyOTPHandler(store *otpStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req verifyOTPRequest
		if !decodeRequest(w, r, &req) {
			return
		}

		if !isValidOTP(req.OTP) {
			writeError(w, http.StatusBadRequest, "invalid otp")
			return
		}

		if !store.verify(req.Email, req.OTP) {
			writeError(w, http.StatusUnauthorized, "invalid otp")
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
