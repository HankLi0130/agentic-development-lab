package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net/http"
)

func sendOTPHandler(store *otpStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req emailRequest
		if !decodeRequest(w, r, &req) {
			return
		}

		otp, err := generateOTP()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		if !store.issue(req.Email, otp) {
			writeError(w, http.StatusTooManyRequests, "too many requests")
			return
		}

		fakeSendOTP(req.Email, otp)

		writeJSON(w, http.StatusOK, map[string]string{"message": "otp sent"})
	}
}

func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func fakeSendOTP(email, otp string) {
	log.Printf("fake send otp: email=%s otp=%s", email, otp)
}
