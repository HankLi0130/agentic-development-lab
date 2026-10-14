package main

import (
	"crypto/subtle"
	"sync"
	"time"
)

const (
	otpTTL         = 60 * time.Second
	maxOTPFailures = 5
)

type otpStore struct {
	mu        sync.Mutex
	otps      map[string]string
	expiresAt map[string]time.Time
	failures  map[string]int
	verified  map[string]bool
	tokens    map[string]string
}

func newOTPStore() *otpStore {
	return &otpStore{otps: make(map[string]string), expiresAt: make(map[string]time.Time), failures: make(map[string]int), verified: make(map[string]bool), tokens: make(map[string]string)}
}

func (s *otpStore) save(email, otp string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.otps[email] = otp
	s.expiresAt[email] = time.Now().Add(otpTTL)
	delete(s.failures, email)
}

func (s *otpStore) verify(email, otp string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, ok := s.otps[email]
	if !ok {
		return false
	}
	if !time.Now().Before(s.expiresAt[email]) {
		s.discard(email)
		return false
	}
	if subtle.ConstantTimeCompare([]byte(stored), []byte(otp)) != 1 {
		s.failures[email]++
		if s.failures[email] >= maxOTPFailures {
			s.discard(email)
		}
		return false
	}
	s.discard(email)
	s.verified[email] = true
	return true
}

func (s *otpStore) discard(email string) {
	delete(s.otps, email)
	delete(s.expiresAt, email)
	delete(s.failures, email)
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
