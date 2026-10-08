package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
)

type loginRequest struct {
	Email string `json:"email"`
}

func loginHandler(store *otpStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		if !isValidEmail(req.Email) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid email"})
			return
		}

		token, err := generateToken()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		if !store.login(req.Email, token) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "email not verified"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"token": token})
	}
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
