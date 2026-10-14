package main

import (
	"crypto/subtle"
	"sync"
	"time"
)

const otpTTL = 60 * time.Second

type otpStore struct {
	mu        sync.Mutex
	otps      map[string]string
	expiresAt map[string]time.Time
	verified  map[string]bool
	tokens    map[string]string
}

func newOTPStore() *otpStore {
	return &otpStore{otps: make(map[string]string), expiresAt: make(map[string]time.Time), verified: make(map[string]bool), tokens: make(map[string]string)}
}

func (s *otpStore) save(email, otp string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.otps[email] = otp
	s.expiresAt[email] = time.Now().Add(otpTTL)
}

func (s *otpStore) verify(email, otp string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, ok := s.otps[email]
	if ok && !time.Now().Before(s.expiresAt[email]) {
		delete(s.otps, email)
		delete(s.expiresAt, email)
		return false
	}
	if !ok || subtle.ConstantTimeCompare([]byte(stored), []byte(otp)) != 1 {
		return false
	}
	delete(s.otps, email)
	delete(s.expiresAt, email)
	s.verified[email] = true
	return true
}

func (s *otpStore) login(email, token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.verified[email] {
		return false
	}
	delete(s.verified, email)
	s.tokens[token] = email
	return true
}
