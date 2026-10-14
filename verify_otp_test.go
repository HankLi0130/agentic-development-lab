package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
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
		{"empty body", ``, http.StatusBadRequest, `{"error":"invalid request body"}`},
		{"missing otp", `{"email":"user@example.com"}`, http.StatusBadRequest, `{"error":"invalid otp"}`},
		{"empty otp", `{"email":"user@example.com","otp":""}`, http.StatusBadRequest, `{"error":"invalid otp"}`},
		{"5 digits", `{"email":"user@example.com","otp":"12345"}`, http.StatusBadRequest, `{"error":"invalid otp"}`},
		{"7 digits", `{"email":"user@example.com","otp":"1234567"}`, http.StatusBadRequest, `{"error":"invalid otp"}`},
		{"letter", `{"email":"user@example.com","otp":"12345a"}`, http.StatusBadRequest, `{"error":"invalid otp"}`},
		{"surrounding space", `{"email":"user@example.com","otp":" 123456"}`, http.StatusBadRequest, `{"error":"invalid otp"}`},
		{"negative sign", `{"email":"user@example.com","otp":"-12345"}`, http.StatusBadRequest, `{"error":"invalid otp"}`},
		{"non-ascii digits", `{"email":"user@example.com","otp":"١٢٣٤٥٦"}`, http.StatusBadRequest, `{"error":"invalid otp"}`},
		{"otp not a string", `{"email":"user@example.com","otp":123456}`, http.StatusBadRequest, `{"error":"invalid request body"}`},
		{"malformed otp on unknown email", `{"email":"other@example.com","otp":"12"}`, http.StatusBadRequest, `{"error":"invalid otp"}`},
		{"extra fields ignored", `{"email":"user@example.com","otp":"123456","foo":"bar"}`, http.StatusOK, `{"verified":true}`},
		{"email checked before otp", `{"email":"bad","otp":"000000"}`, http.StatusBadRequest, `{"error":"invalid email"}`},
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

func TestMalformedOTPKeepsStoredOTP(t *testing.T) {
	store := newOTPStore()
	store.save("user@example.com", "123456")
	if rec := postVerify(t, store, `{"email":"user@example.com","otp":"12345"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed otp status = %d, want 400", rec.Code)
	}
	if rec := postVerify(t, store, `{"email":"user@example.com","otp":"123456"}`); rec.Code != http.StatusOK {
		t.Fatalf("correct otp status = %d, want 200", rec.Code)
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

func TestVerifyOTPFailureKeepsOTP(t *testing.T) {
	store := newOTPStore()
	store.save("user@example.com", "123456")
	if rec := postVerify(t, store, `{"email":"user@example.com","otp":"000000"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong otp status = %d, want 401", rec.Code)
	}
	if store.verified["user@example.com"] {
		t.Fatal("user marked verified after wrong otp")
	}
	if rec := postVerify(t, store, `{"email":"user@example.com","otp":"123456"}`); rec.Code != http.StatusOK {
		t.Fatalf("correct otp status = %d, want 200", rec.Code)
	}
}

func TestVerifyOTPConcurrentSingleUse(t *testing.T) {
	store := newOTPStore()
	store.save("user@example.com", "123456")
	body := `{"email":"user@example.com","otp":"123456"}`
	const n = 50
	codes := make(chan int, n)
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes <- postVerify(t, store, body).Code
		}()
	}
	wg.Wait()
	close(codes)
	ok := 0
	for c := range codes {
		if c == http.StatusOK {
			ok++
		}
	}
	if ok != 1 {
		t.Fatalf("successful verifies = %d, want 1", ok)
	}
}

func TestVerifyOTPExpiry(t *testing.T) {
	body := `{"email":"user@example.com","otp":"123456"}`
	tests := []struct {
		name       string
		wait       time.Duration
		wantStatus int
	}{
		{"before expiry", otpTTL - time.Second, http.StatusOK},
		{"at expiry", otpTTL, http.StatusUnauthorized},
		{"after expiry", otpTTL + time.Second, http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				store := newOTPStore()
				store.save("user@example.com", "123456")
				time.Sleep(tt.wait)
				if rec := postVerify(t, store, body); rec.Code != tt.wantStatus {
					t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
				}
			})
		})
	}
}

func TestExpiredOTPIsRemoved(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := newOTPStore()
		store.save("user@example.com", "123456")
		time.Sleep(otpTTL)
		rec := postVerify(t, store, `{"email":"user@example.com","otp":"123456"}`)
		if got := strings.TrimSpace(rec.Body.String()); got != `{"error":"invalid otp"}` {
			t.Fatalf("body = %s", got)
		}
		if _, ok := store.otps["user@example.com"]; ok {
			t.Fatal("expired otp still stored")
		}
		if _, ok := store.expiresAt["user@example.com"]; ok {
			t.Fatal("expired expiry still stored")
		}
		if store.verified["user@example.com"] {
			t.Fatal("user marked verified with expired otp")
		}
	})
}

func TestResendOTPResetsExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := newOTPStore()
		store.save("user@example.com", "111111")
		time.Sleep(50 * time.Second)
		store.save("user@example.com", "123456")
		time.Sleep(50 * time.Second)
		if rec := postVerify(t, store, `{"email":"user@example.com","otp":"123456"}`); rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})
}

func TestVerifiedOTPIsRemoved(t *testing.T) {
	store := newOTPStore()
	store.save("user@example.com", "123456")
	if rec := postVerify(t, store, `{"email":"user@example.com","otp":"123456"}`); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if _, ok := store.otps["user@example.com"]; ok {
		t.Fatal("used otp still stored")
	}
	if _, ok := store.expiresAt["user@example.com"]; ok {
		t.Fatal("used otp expiry still stored")
	}
	rec := postVerify(t, store, `{"email":"user@example.com","otp":"123456"}`)
	if got := strings.TrimSpace(rec.Body.String()); rec.Code != http.StatusUnauthorized || got != `{"error":"invalid otp"}` {
		t.Fatalf("reuse status = %d, body = %s", rec.Code, got)
	}
}
