package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/mail"
)

type sendOTPRequest struct {
	Email string `json:"email"`
}

func sendOTPHandler(store *otpStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req sendOTPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		if !isValidEmail(req.Email) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid email"})
			return
		}

		otp, err := generateOTP()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		store.save(req.Email, otp)

		fakeSendOTP(req.Email, otp)

		writeJSON(w, http.StatusOK, map[string]string{"message": "otp sent"})
	}
}

func isValidEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
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

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
