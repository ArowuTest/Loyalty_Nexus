package services

import (
	"sync"
	"time"
)

// otpLockout is an in-process, per-phone failed-verify limiter that closes the
// OTP brute-force / account-takeover vector (review finding S11): without it,
// VerifyOTP accepts UNLIMITED guesses against a 6-digit (900,000-value) code for
// the full 5-minute OTP lifetime, so a target account can be taken over by
// exhausting the code space. After maxFailures wrong codes within failWindow the
// phone is locked for banDuration; a correct code (Reset) clears the counter.
//
// It is in-process (per API instance) BY DESIGN: it adds no Redis/DB dependency
// to the auth hot path, so a datastore blip can never lock users out. With N
// instances the effective cap is N*maxFailures — still a decisive reduction from
// "unlimited" to a handful, which makes a 1-in-900,000 brute force infeasible.
// The key is the (already E.164-normalized) phone number, not the client IP, so
// it defends the targeted account even against a distributed / proxy-rotating
// attacker that would bypass a per-IP limit.
//
// Tradeoff: an attacker can grief a victim by burning maxFailures wrong guesses
// to lock them out for banDuration. banDuration is kept short so that window is
// small; the alternative (no lockout → account takeover) is far worse.
type otpLockout struct {
	mu          sync.Mutex
	entries     map[string]*otpLockEntry
	maxFailures int
	failWindow  time.Duration
	banDuration time.Duration
}

type otpLockEntry struct {
	failures     int
	windowStart  time.Time
	blockedUntil time.Time
}

func newOTPLockout(maxFailures int, failWindow, banDuration time.Duration) *otpLockout {
	l := &otpLockout{
		entries:     make(map[string]*otpLockEntry),
		maxFailures: maxFailures,
		failWindow:  failWindow,
		banDuration: banDuration,
	}
	go func() {
		for range time.Tick(10 * time.Minute) {
			l.gc()
		}
	}()
	return l
}

// Blocked reports whether phone is currently locked out.
func (l *otpLockout) Blocked(phone string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e := l.entries[phone]
	return e != nil && time.Now().Before(e.blockedUntil)
}

// Fail records one wrong-code attempt for phone and locks it once maxFailures
// wrong codes accumulate within failWindow.
func (l *otpLockout) Fail(phone string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	e := l.entries[phone]
	if e == nil || now.Sub(e.windowStart) > l.failWindow {
		e = &otpLockEntry{windowStart: now}
		l.entries[phone] = e
	}
	e.failures++
	if e.failures >= l.maxFailures {
		e.blockedUntil = now.Add(l.banDuration)
	}
}

// Reset clears any failure state for phone (call on a successful verify).
func (l *otpLockout) Reset(phone string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, phone)
}

func (l *otpLockout) gc() {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	for k, e := range l.entries {
		if now.After(e.blockedUntil) && now.Sub(e.windowStart) > l.failWindow {
			delete(l.entries, k)
		}
	}
}
