package main

import (
	"bytes"
	"log"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"
)

var otpPattern = regexp.MustCompile(`^\d{6}$`)

func TestSendOTP(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{"success", `{"email":"user@example.com"}`, http.StatusOK, `{"message":"otp sent"}`},
		{"invalid body", `{`, http.StatusBadRequest, `{"error":"invalid request body"}`},
		{"empty body", ``, http.StatusBadRequest, `{"error":"invalid request body"}`},
		{"missing email", `{}`, http.StatusBadRequest, `{"error":"invalid email"}`},
		{"invalid email", `{"email":"bad"}`, http.StatusBadRequest, `{"error":"invalid email"}`},
		{"email with display name", `{"email":"User <user@example.com>"}`, http.StatusBadRequest, `{"error":"invalid email"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newOTPStore()
			rec := post(t, sendOTPHandler(store), "/send-otp", tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := strings.TrimSpace(rec.Body.String()); got != tt.wantBody {
				t.Fatalf("body = %s, want %s", got, tt.wantBody)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", ct)
			}
			if tt.wantStatus != http.StatusOK && len(store.otps) != 0 {
				t.Fatalf("store not empty on failure: %v", store.otps)
			}
		})
	}
}

func TestSendOTPStoresSixDigitOTP(t *testing.T) {
	store := newOTPStore()
	post(t, sendOTPHandler(store), "/send-otp", `{"email":"user@example.com"}`)
	otp, ok := store.otps["user@example.com"]
	if !ok {
		t.Fatal("otp not stored")
	}
	if !otpPattern.MatchString(otp) {
		t.Fatalf("otp = %q, want 6 digits", otp)
	}
}

func TestSendOTPOverwritesPrevious(t *testing.T) {
	store := newOTPStore()
	store.save("user@example.com", "stale!")
	post(t, sendOTPHandler(store), "/send-otp", `{"email":"user@example.com"}`)
	if got := store.otps["user@example.com"]; got == "stale!" {
		t.Fatal("previous otp not overwritten")
	}
}

func TestSendOTPLogsFakeSend(t *testing.T) {
	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(orig) })

	store := newOTPStore()
	post(t, sendOTPHandler(store), "/send-otp", `{"email":"user@example.com"}`)
	want := "fake send otp: email=user@example.com otp=" + store.otps["user@example.com"]
	if !strings.Contains(buf.String(), want) {
		t.Fatalf("log = %q, want contains %q", buf.String(), want)
	}
}

func TestSendOTPThenVerify(t *testing.T) {
	store := newOTPStore()
	post(t, sendOTPHandler(store), "/send-otp", `{"email":"user@example.com"}`)
	otp := store.otps["user@example.com"]
	rec := postVerify(t, store, `{"email":"user@example.com","otp":"`+otp+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify status = %d, want 200", rec.Code)
	}
}

func TestGenerateOTP(t *testing.T) {
	for range 1000 {
		otp, err := generateOTP()
		if err != nil {
			t.Fatal(err)
		}
		if !otpPattern.MatchString(otp) {
			t.Fatalf("otp = %q, want 6 digits", otp)
		}
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email string
		want  bool
	}{
		{"user@example.com", true},
		{"a.b+tag@sub.example.com", true},
		{"", false},
		{"bad", false},
		{"@example.com", false},
		{"User <user@example.com>", false},
		{" user@example.com", false},
	}
	for _, tt := range tests {
		if got := isValidEmail(tt.email); got != tt.want {
			t.Errorf("isValidEmail(%q) = %v, want %v", tt.email, got, tt.want)
		}
	}
}

func TestGenerateOTPIsSixDigits(t *testing.T) {
	for range 1000 {
		otp, err := generateOTP()
		if err != nil {
			t.Fatal(err)
		}
		if !isValidOTP(otp) {
			t.Fatalf("generated otp %q is not 6 digits", otp)
		}
	}
}

func TestSendOTPCooldown(t *testing.T) {
	store := newOTPStore()
	h := sendOTPHandler(store)
	post(t, h, "/send-otp", `{"email":"user@example.com"}`)
	first := store.otps["user@example.com"]

	rec := post(t, h, "/send-otp", `{"email":"user@example.com"}`)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"error":"too many requests"}` {
		t.Fatalf("body = %s", got)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	if got := store.otps["user@example.com"]; got != first {
		t.Fatalf("otp changed during cooldown: %q -> %q", first, got)
	}
}

func TestSendOTPCooldownDoesNotLogFakeSend(t *testing.T) {
	store := newOTPStore()
	h := sendOTPHandler(store)
	post(t, h, "/send-otp", `{"email":"user@example.com"}`)

	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(orig) })

	post(t, h, "/send-otp", `{"email":"user@example.com"}`)
	if strings.Contains(buf.String(), "fake send otp") {
		t.Fatalf("otp sent during cooldown: %q", buf.String())
	}
}

func TestSendOTPCooldownIsPerEmail(t *testing.T) {
	store := newOTPStore()
	h := sendOTPHandler(store)
	post(t, h, "/send-otp", `{"email":"a@example.com"}`)
	if rec := post(t, h, "/send-otp", `{"email":"b@example.com"}`); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestSendOTPAllowedAfterCooldown(t *testing.T) {
	store := newOTPStore()
	h := sendOTPHandler(store)
	post(t, h, "/send-otp", `{"email":"user@example.com"}`)
	store.lastSent["user@example.com"] = time.Now().Add(-otpCooldown)

	if rec := post(t, h, "/send-otp", `{"email":"user@example.com"}`); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestSendOTPFailedRequestDoesNotStartCooldown(t *testing.T) {
	store := newOTPStore()
	h := sendOTPHandler(store)
	post(t, h, "/send-otp", `{"email":"bad"}`)
	if len(store.lastSent) != 0 {
		t.Fatalf("cooldown started on invalid request: %v", store.lastSent)
	}
	if rec := post(t, h, "/send-otp", `{"email":"user@example.com"}`); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
