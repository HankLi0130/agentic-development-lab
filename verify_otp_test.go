package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func postVerify(t *testing.T, store *otpStore, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/verify-otp", strings.NewReader(body))
	rec := httptest.NewRecorder()
	verifyOTPHandler(store).ServeHTTP(rec, req)
	return rec
}

func TestVerifyOTP(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{"success", `{"email":"user@example.com","otp":"123456"}`, http.StatusOK, `{"verified":true}`},
		{"wrong otp", `{"email":"user@example.com","otp":"000000"}`, http.StatusUnauthorized, `{"error":"invalid otp"}`},
		{"unknown email", `{"email":"other@example.com","otp":"123456"}`, http.StatusUnauthorized, `{"error":"invalid otp"}`},
		{"invalid email", `{"email":"bad","otp":"123456"}`, http.StatusBadRequest, `{"error":"invalid email"}`},
		{"invalid body", `{`, http.StatusBadRequest, `{"error":"invalid request body"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newOTPStore()
			store.save("user@example.com", "123456")
			rec := postVerify(t, store, tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := strings.TrimSpace(rec.Body.String()); got != tt.wantBody {
				t.Fatalf("body = %s, want %s", got, tt.wantBody)
			}
		})
	}
}

func TestVerifyOTPIsSingleUse(t *testing.T) {
	store := newOTPStore()
	store.save("user@example.com", "123456")
	body := `{"email":"user@example.com","otp":"123456"}`
	if rec := postVerify(t, store, body); rec.Code != http.StatusOK {
		t.Fatalf("first verify status = %d", rec.Code)
	}
	if rec := postVerify(t, store, body); rec.Code != http.StatusUnauthorized {
		t.Fatalf("second verify status = %d, want 401", rec.Code)
	}
	if !store.verified["user@example.com"] {
		t.Fatal("user not marked verified")
	}
}
