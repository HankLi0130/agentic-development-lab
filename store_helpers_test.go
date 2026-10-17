package main

import "time"

func (s *otpStore) save(email, otp string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saveLocked(email, otp, time.Now())
}
