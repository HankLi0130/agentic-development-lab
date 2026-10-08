package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func post(t *testing.T, h http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{"not verified", `{"email":"other@example.com"}`, http.StatusUnauthorized, `{"error":"email not verified"}`},
		{"invalid email", `{"email":"bad"}`, http.StatusBadRequest, `{"error":"invalid email"}`},
		{"invalid body", `{`, http.StatusBadRequest, `{"error":"invalid request body"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newOTPStore()
			store.verified["user@example.com"] = true
			rec := post(t, loginHandler(store), "/login", tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := strings.TrimSpace(rec.Body.String()); got != tt.wantBody {
				t.Fatalf("body = %s, want %s", got, tt.wantBody)
			}
		})
	}
}

func TestFullFlow(t *testing.T) {
	store := newOTPStore()
	email := "user@example.com"

	if rec := post(t, loginHandler(store), "/login", `{"email":"`+email+`"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("login before verify status = %d, want 401", rec.Code)
	}

	if rec := post(t, sendOTPHandler(store), "/send-otp", `{"email":"`+email+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("send-otp status = %d", rec.Code)
	}
	otp := store.otps[email]

	if rec := post(t, verifyOTPHandler(store), "/verify-otp", `{"email":"`+email+`","otp":"`+otp+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("verify-otp status = %d", rec.Code)
	}

	rec := post(t, loginHandler(store), "/login", `{"email":"`+email+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200", rec.Code)
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil || len(resp.Token) != 64 {
		t.Fatalf("bad token response: %v %q", err, resp.Token)
	}
	if store.tokens[resp.Token] != email {
		t.Fatal("token not stored")
	}

	if rec := post(t, loginHandler(store), "/login", `{"email":"`+email+`"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("second login status = %d, want 401", rec.Code)
	}
}
