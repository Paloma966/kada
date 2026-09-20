package service

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/chun/kada-backend/internal/domain"
)

func TestPhonePattern(t *testing.T) {
	valid := []string{"13812345678", "19912345678", "15800000000"}
	for _, p := range valid {
		if !phonePattern.MatchString(p) {
			t.Errorf("expected %q to be valid phone", p)
		}
	}
	invalid := []string{
		"", "123", "12345678901", // wrong length
		"11812345678", "12812345678", // invalid second digit
		"1381234567a",    // contains a letter
		"138 1234 5678",  // contains a space
		"aaaaaaaaaaa",    // all letters
		"+8613812345678", // with country code
	}
	for _, p := range invalid {
		if phonePattern.MatchString(p) {
			t.Errorf("expected %q to be invalid phone", p)
		}
	}
}

// recordingSender counts the calls that would have cost money.
type recordingSender struct {
	calls int
}

func (s *recordingSender) SendVerificationCode(string) (string, error) {
	s.calls++
	return "123456", nil
}

func (s *recordingSender) CheckVerificationCode(string, string) (bool, error) { return true, nil }

// A malformed number must be rejected before anything is read or written. A nil *gorm.DB is the sharpest
// way to say so: if the check moved below the first query, this test would panic instead of passing.
func TestSendSMSCodeRejectsMalformedPhoneBeforeTouchingTheDatabase(t *testing.T) {
	sender := &recordingSender{}
	svc := NewAuthService(nil, "secret", "1h", sender)

	err := svc.SendSMSCode(context.Background(), "12345", "203.0.113.9", "captcha-id", "A3B7")
	if err == nil {
		t.Fatal("expected an error for a malformed phone number")
	}
	if sender.calls != 0 {
		t.Errorf("an SMS was sent for a malformed number (calls=%d)", sender.calls)
	}
}

// The captcha is checked first, and the same nil-DB trick proves the ordering: an unanswered challenge
// must be refused before any query, and therefore long before any provider call.
func TestSendSMSCodeRequiresASolvedCaptcha(t *testing.T) {
	sender := &recordingSender{}
	svc := NewAuthService(nil, "secret", "1h", sender)

	for _, c := range []struct{ id, answer string }{{"", ""}, {"", "A3B7"}, {"captcha-id", ""}} {
		err := svc.SendSMSCode(context.Background(), "13800138000", "203.0.113.9", c.id, c.answer)
		if err == nil {
			t.Errorf("expected an error for captcha id=%q answer=%q", c.id, c.answer)
		}
		var limited *domain.RateLimitError
		if errors.As(err, &limited) {
			t.Errorf("a missing captcha is bad input, not a quota: %v", err)
		}
	}
	if sender.calls != 0 {
		t.Errorf("an SMS was sent without a solved captcha (calls=%d)", sender.calls)
	}
}

// With no matching row, the atomic consume cannot confirm anything, so the send has to stop there. The
// dry-run database renders the SQL without executing it and returns zero rows, which is exactly the
// "nothing matched" case a wrong or expired challenge produces in production.
func TestSendSMSCodeStopsWhenTheCaptchaCannotBeConfirmed(t *testing.T) {
	sender := &recordingSender{}
	svc := NewAuthService(newDryRunDB(t), "secret", "1h", sender)

	err := svc.SendSMSCode(context.Background(), "13800138000", "203.0.113.9", "captcha-id", "WRONG")
	if err == nil {
		t.Fatal("expected the send to be refused when no captcha row matched")
	}
	if sender.calls != 0 {
		t.Errorf("an SMS was sent despite an unconfirmed captcha (calls=%d)", sender.calls)
	}
}

// The quotas are the thing standing between one script and an SMS bill, so their ordering matters and is
// easy to break by editing a constant: the hourly figure must be smaller than the daily one, and the
// per-IP allowance must be larger than the per-phone one (a NAT legitimately shares an address).
func TestQuotaConstantsAreConsistent(t *testing.T) {
	if smsIPHourlyMax >= smsIPDailyMax {
		t.Errorf("the per-IP hourly quota (%d) must be below the daily one (%d)", smsIPHourlyMax, smsIPDailyMax)
	}
	if smsPhoneDailyMax > smsIPDailyMax {
		t.Errorf("a single phone may not consume more than the whole network's daily allowance (%d > %d)",
			smsPhoneDailyMax, smsIPDailyMax)
	}
	if smsPhoneCooldown < 30*time.Second {
		t.Errorf("a 60s cooldown is the floor that makes retrying pointless; got %v", smsPhoneCooldown)
	}
}

func TestRandomIDIsHexAndNotPredictable(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := randomID()
		if err != nil {
			t.Fatalf("randomID() failed: %v", err)
		}
		if len(id) != 32 {
			t.Fatalf("randomID() returned %d characters (%q), want 32", len(id), id)
		}
		if _, err := hex.DecodeString(id); err != nil {
			t.Fatalf("randomID() returned %q, which is not hex: %v", id, err)
		}
		if seen[id] {
			t.Fatalf("randomID() repeated %q within 100 calls", id)
		}
		seen[id] = true
	}
}

// An unknown address is stored as NULL, so it cannot be counted as one shared bucket by the per-IP quotas.
func TestStringPtrTreatsEmptyAsAbsent(t *testing.T) {
	if got := stringPtr(""); got != nil {
		t.Errorf("stringPtr(\"\") = %v, want nil", got)
	}
	if got := stringPtr("203.0.113.9"); got == nil || *got != "203.0.113.9" {
		t.Errorf("stringPtr(\"203.0.113.9\") = %v, want a pointer to that value", got)
	}
}
