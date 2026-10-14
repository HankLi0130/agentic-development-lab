package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

// 500 cases need crypto/rand to fail, which cannot be triggered from a request.
var contractUntestable = map[string]bool{
	"POST /send-otp 500": true,
	"POST /login 500":    true,
}

func TestContract(t *testing.T) {
	ctx := context.Background()
	doc, err := openapi3.NewLoader().LoadFromFile("openapi.yaml")
	if err != nil {
		t.Fatalf("load openapi.yaml: %v", err)
	}
	if err := doc.Validate(ctx); err != nil {
		t.Fatalf("invalid openapi.yaml: %v", err)
	}
	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	const email = "user@example.com"
	tests := []struct {
		path       string
		body       string
		setup      func(*otpStore)
		wantStatus int
	}{
		{"/send-otp", `{"email":"` + email + `"}`, nil, http.StatusOK},
		{"/send-otp", `{"email":"bad"}`, nil, http.StatusBadRequest},
		{"/verify-otp", `{"email":"` + email + `","otp":"123456"}`, func(s *otpStore) { s.save(email, "123456") }, http.StatusOK},
		{"/verify-otp", `{`, nil, http.StatusBadRequest},
		{"/verify-otp", `{"email":"` + email + `","otp":"000000"}`, func(s *otpStore) { s.save(email, "123456") }, http.StatusUnauthorized},
		{"/login", `{"email":"` + email + `"}`, func(s *otpStore) { s.verified[email] = true }, http.StatusOK},
		{"/login", `{}`, nil, http.StatusBadRequest},
		{"/login", `{"email":"` + email + `"}`, nil, http.StatusUnauthorized},
	}

	covered := map[string]bool{}
	for _, tt := range tests {
		key := fmt.Sprintf("POST %s %d", tt.path, tt.wantStatus)
		covered[key] = true
		t.Run(key, func(t *testing.T) {
			store := newOTPStore()
			if tt.setup != nil {
				tt.setup(store)
			}
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			newMux(store).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tt.wantStatus, rec.Body)
			}
			route, pathParams, err := router.FindRoute(req)
			if err != nil {
				t.Fatalf("route not in contract: %v", err)
			}
			input := &openapi3filter.ResponseValidationInput{
				RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req, PathParams: pathParams, Route: route},
				Status:                 rec.Code,
				Header:                 rec.Header(),
				Body:                   io.NopCloser(rec.Body),
				Options:                &openapi3filter.Options{IncludeResponseStatus: true},
			}
			if err := openapi3filter.ValidateResponse(ctx, input); err != nil {
				t.Fatalf("response violates contract: %v", err)
			}
		})
	}

	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			for status := range op.Responses.Map() {
				key := fmt.Sprintf("%s %s %s", method, path, status)
				if !covered[key] && !contractUntestable[key] {
					t.Errorf("contract case not tested: %s", key)
				}
			}
		}
	}
}
