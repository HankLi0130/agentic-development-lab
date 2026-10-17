package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
)

func loginHandler(store *otpStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req emailRequest
		if !decodeRequest(w, r, &req) {
			return
		}

		token, err := store.login(req.Email)
		if errors.Is(err, errNotVerified) {
			writeError(w, http.StatusUnauthorized, "email not verified")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
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
