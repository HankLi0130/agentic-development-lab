package main

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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
		{"empty body", ``, http.StatusBadRequest, `{"error":"invalid request body"}`},
		{"missing email", `{}`, http.StatusBadRequest, `{"error":"invalid email"}`},
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

func loginToken(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp.Token
}

func TestLoginSuccess(t *testing.T) {
	store := newOTPStore()
	store.verified["user@example.com"] = true
	rec := post(t, loginHandler(store), "/login", `{"email":"user@example.com"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	token := loginToken(t, rec)
	if b, err := hex.DecodeString(token); err != nil || len(b) != 32 {
		t.Fatalf("token %q is not 32 bytes of hex", token)
	}
	if store.tokens[token] != "user@example.com" {
		t.Fatal("token not stored for email")
	}
	if store.verified["user@example.com"] {
		t.Fatal("verified flag not consumed")
	}
}

func TestLoginFailureStoresNoToken(t *testing.T) {
	store := newOTPStore()
	store.verified["user@example.com"] = true
	if rec := post(t, loginHandler(store), "/login", `{"email":"other@example.com"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if len(store.tokens) != 0 {
		t.Fatalf("tokens stored = %d, want 0", len(store.tokens))
	}
	if !store.verified["user@example.com"] {
		t.Fatal("other user's verified flag consumed")
	}
}

func TestLoginIsSingleUse(t *testing.T) {
	store := newOTPStore()
	store.verified["user@example.com"] = true
	body := `{"email":"user@example.com"}`
	if rec := post(t, loginHandler(store), "/login", body); rec.Code != http.StatusOK {
		t.Fatalf("first login status = %d", rec.Code)
	}
	if rec := post(t, loginHandler(store), "/login", body); rec.Code != http.StatusUnauthorized {
		t.Fatalf("second login status = %d, want 401", rec.Code)
	}
	if len(store.tokens) != 1 {
		t.Fatalf("tokens stored = %d, want 1", len(store.tokens))
	}
}

func TestLoginTokensUnique(t *testing.T) {
	store := newOTPStore()
	body := `{"email":"user@example.com"}`
	store.verified["user@example.com"] = true
	first := loginToken(t, post(t, loginHandler(store), "/login", body))
	store.verified["user@example.com"] = true
	second := loginToken(t, post(t, loginHandler(store), "/login", body))
	if first == second {
		t.Fatal("tokens are identical")
	}
	if len(store.tokens) != 2 {
		t.Fatalf("tokens stored = %d, want 2", len(store.tokens))
	}
}

func TestLoginConcurrentSingleUse(t *testing.T) {
	store := newOTPStore()
	store.verified["user@example.com"] = true
	body := `{"email":"user@example.com"}`
	const n = 50
	codes := make(chan int, n)
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes <- post(t, loginHandler(store), "/login", body).Code
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
		t.Fatalf("successful logins = %d, want 1", ok)
	}
	if len(store.tokens) != 1 {
		t.Fatalf("tokens stored = %d, want 1", len(store.tokens))
	}
}

func TestReplayedOTPCannotLoginAgain(t *testing.T) {
	store := newOTPStore()
	email := "user@example.com"
	if rec := post(t, sendOTPHandler(store), "/send-otp", `{"email":"`+email+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("send-otp status = %d", rec.Code)
	}
	verifyBody := `{"email":"` + email + `","otp":"` + store.otps[email] + `"}`
	if rec := post(t, verifyOTPHandler(store), "/verify-otp", verifyBody); rec.Code != http.StatusOK {
		t.Fatalf("verify-otp status = %d", rec.Code)
	}
	if rec := post(t, loginHandler(store), "/login", `{"email":"`+email+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("login status = %d", rec.Code)
	}
	if rec := post(t, verifyOTPHandler(store), "/verify-otp", verifyBody); rec.Code != http.StatusUnauthorized {
		t.Fatalf("replayed verify-otp status = %d, want 401", rec.Code)
	}
	if rec := post(t, loginHandler(store), "/login", `{"email":"`+email+`"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("login after replay status = %d, want 401", rec.Code)
	}
	if len(store.tokens) != 1 {
		t.Fatalf("tokens = %d, want 1", len(store.tokens))
	}
}
