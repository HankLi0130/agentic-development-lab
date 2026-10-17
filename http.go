package main

import (
	"encoding/json"
	"net/http"
	"net/mail"
)

type emailRequest struct {
	Email string `json:"email"`
}

func (r emailRequest) email() string { return r.Email }

func decodeRequest(w http.ResponseWriter, r *http.Request, req interface{ email() string }) bool {
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	if !isValidEmail(req.email()) {
		writeError(w, http.StatusBadRequest, "invalid email")
		return false
	}
	return true
}

func isValidEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
