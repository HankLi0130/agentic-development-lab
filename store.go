package main

import (
	"crypto/subtle"
	"sync"
)

type otpStore struct {
	mu       sync.Mutex
	otps     map[string]string
	verified map[string]bool
}

func newOTPStore() *otpStore {
	return &otpStore{otps: make(map[string]string), verified: make(map[string]bool)}
}

func (s *otpStore) save(email, otp string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.otps[email] = otp
}

func (s *otpStore) verify(email, otp string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, ok := s.otps[email]
	if !ok || subtle.ConstantTimeCompare([]byte(stored), []byte(otp)) != 1 {
		return false
	}
	delete(s.otps, email)
	s.verified[email] = true
	return true
}
