package main

import (
	"crypto/subtle"
	"errors"
	"sync"
	"time"
)

const (
	otpTTL         = 60 * time.Second
	otpCooldown    = 30 * time.Second
	maxOTPFailures = 5
)

var errNotVerified = errors.New("email not verified")

type otpStore struct {
	mu        sync.Mutex
	otps      map[string]string
	expiresAt map[string]time.Time
	lastSent  map[string]time.Time
	failures  map[string]int
	verified  map[string]bool
	tokens    map[string]string
}

func newOTPStore() *otpStore {
	return &otpStore{
		otps:      make(map[string]string),
		expiresAt: make(map[string]time.Time),
		lastSent:  make(map[string]time.Time),
		failures:  make(map[string]int),
		verified:  make(map[string]bool),
		tokens:    make(map[string]string),
	}
}

func (s *otpStore) issue(email, otp string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if last, ok := s.lastSent[email]; ok && now.Sub(last) < otpCooldown {
		return false
	}
	s.lastSent[email] = now
	s.saveLocked(email, otp, now)
	return true
}

func (s *otpStore) saveLocked(email, otp string, now time.Time) {
	s.otps[email] = otp
	s.expiresAt[email] = now.Add(otpTTL)
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
		n := s.failures[email] + 1
		s.failures[email] = n
		if n >= maxOTPFailures {
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

func (s *otpStore) login(email string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.verified[email] {
		return "", errNotVerified
	}
	token, err := generateToken()
	if err != nil {
		return "", err
	}
	delete(s.verified, email)
	s.tokens[token] = email
	return token, nil
}
