package services

import (
	"testing"
	"time"
)

// These tests defend the OTP brute-force lockout (S11). Each is written so it
// FAILS if the specific guarantee it names regresses.

func TestOTPLockout_BlocksExactlyAtThreshold(t *testing.T) {
	l := newOTPLockout(5, 10*time.Minute, 15*time.Minute)
	phone := "+2348011111111"
	for i := 1; i <= 4; i++ {
		l.Fail(phone)
		if l.Blocked(phone) {
			t.Fatalf("locked after %d failures; must not lock before the 5th", i)
		}
	}
	l.Fail(phone) // 5th wrong code
	if !l.Blocked(phone) {
		t.Fatal("not locked after 5 failures — brute-force protection is off")
	}
}

func TestOTPLockout_ResetClearsOnSuccess(t *testing.T) {
	l := newOTPLockout(3, 10*time.Minute, 15*time.Minute)
	phone := "+2348011111111"
	l.Fail(phone)
	l.Fail(phone)
	l.Fail(phone)
	if !l.Blocked(phone) {
		t.Fatal("want locked after 3 failures")
	}
	l.Reset(phone) // a correct code was entered
	if l.Blocked(phone) {
		t.Fatal("still locked after Reset — a valid code must clear the lock")
	}
}

func TestOTPLockout_BanExpires(t *testing.T) {
	l := newOTPLockout(1, 10*time.Minute, 30*time.Millisecond)
	phone := "+2348011111111"
	l.Fail(phone)
	if !l.Blocked(phone) {
		t.Fatal("want locked immediately at max=1")
	}
	time.Sleep(50 * time.Millisecond)
	if l.Blocked(phone) {
		t.Fatal("still locked after banDuration elapsed — lock must be time-bounded")
	}
}

func TestOTPLockout_FailuresAgeOutOfWindow(t *testing.T) {
	l := newOTPLockout(3, 30*time.Millisecond, 15*time.Minute)
	phone := "+2348011111111"
	l.Fail(phone)
	l.Fail(phone) // 2 within the window
	time.Sleep(50 * time.Millisecond)
	l.Fail(phone) // window elapsed → fresh window, count resets to 1
	if l.Blocked(phone) {
		t.Fatal("locked, but the two earlier failures should have aged out of the window")
	}
}

func TestOTPLockout_IsolatedPerPhone(t *testing.T) {
	l := newOTPLockout(2, 10*time.Minute, 15*time.Minute)
	victim, other := "+2348011111111", "+2348022222222"
	l.Fail(victim)
	l.Fail(victim)
	if !l.Blocked(victim) {
		t.Fatal("victim should be locked")
	}
	if l.Blocked(other) {
		t.Fatal("an unrelated phone must not be locked by another phone's failures")
	}
}
