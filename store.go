package main

import "sync"

type otpStore struct {
	mu   sync.Mutex
	otps map[string]string
}

func newOTPStore() *otpStore {
	return &otpStore{otps: make(map[string]string)}
}

func (s *otpStore) save(email, otp string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.otps[email] = otp
}
