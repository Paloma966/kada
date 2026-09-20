package sms

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"13812345678", "138****5678"},
		{"1234567", "123****4567"},
		{"123456", "***"},
		{"", "***"},
	}
	for _, tt := range tests {
		if got := maskPhone(tt.in); got != tt.want {
			t.Errorf("maskPhone(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// A failed send must never report a bare "try again later": the Aliyun code is the only thing that
// identifies an unapproved signature/template or a disabled AccessKey.
func TestSMSProviderErrorIncludesProviderCode(t *testing.T) {
	t.Setenv("GIN_MODE", "debug")
	msg := smsProviderError("isv.SMS_SIGNATURE_ILLEGAL", "signature is not approved")

	if !strings.Contains(msg, "isv.SMS_SIGNATURE_ILLEGAL") {
		t.Errorf("expected the provider code in %q", msg)
	}
	if !strings.Contains(msg, "signature is not approved") {
		t.Errorf("expected the provider message in %q", msg)
	}
}

// Outside debug the provider message may embed account details, so only the code is surfaced.
func TestSMSProviderErrorHidesMessageInRelease(t *testing.T) {
	t.Setenv("GIN_MODE", "release")
	msg := smsProviderError("isv.SMS_SIGNATURE_ILLEGAL", "signature is not approved")

	if !strings.Contains(msg, "isv.SMS_SIGNATURE_ILLEGAL") {
		t.Errorf("expected the provider code in %q", msg)
	}
	if strings.Contains(msg, "signature is not approved") {
		t.Errorf("provider message must not leak in release mode: %q", msg)
	}
}

// An empty payload (for example a nil Body on a transport-level failure) degrades to the generic text
// instead of producing a dangling "(provider code: )".
func TestSMSProviderErrorEmptyPayload(t *testing.T) {
	got := smsProviderError("", "")
	if got != "failed to send SMS, please try again later" {
		t.Errorf("unexpected message: %q", got)
	}
}

// A missing signature or template must fail loudly at construction. Defaulting them is how a
// configuration mistake turned into a provider rejection that read like a broken Aliyun account.
func TestNewAliyunSenderRejectsIncompleteConfiguration(t *testing.T) {
	tests := []struct {
		name         string
		accessKeyID  string
		secret       string
		signName     string
		templateCode string
		wantMentions string
	}{
		{"no credentials", "", "", "signature", "SMS_123", "SMS_ACCESS_KEY_ID"},
		{"no secret", "id", "", "signature", "SMS_123", "SMS_ACCESS_KEY_ID"},
		{"no signature", "id", "secret", "", "SMS_123", "SMS_SIGN_NAME"},
		{"no template", "id", "secret", "signature", "", "SMS_TEMPLATE_CODE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender, err := NewAliyunSender(tt.accessKeyID, tt.secret, tt.signName, tt.templateCode)
			if err == nil {
				t.Fatalf("expected an error naming the missing setting, got a sender: %+v", sender)
			}
			if sender != nil {
				t.Errorf("expected no sender alongside the error")
			}
			if !strings.Contains(err.Error(), tt.wantMentions) {
				t.Errorf("error %q should name %s", err, tt.wantMentions)
			}
		})
	}
}

// A pair that is present but is not a pair is the failure that hides best: it passes every emptiness
// check, construction succeeds, and the provider answers SignatureDoesNotMatch only when a code is
// finally requested. A 12-character secret reached production exactly that way, so the shape is checked
// at startup - as a warning, because refusing to start would also stop short-link redirection.
func TestNewAliyunSenderWarnsAboutImplausibleCredentialLengths(t *testing.T) {
	var logged bytes.Buffer
	previousWriter := log.Writer()
	log.SetOutput(&logged)
	defer log.SetOutput(previousWriter)

	plausibleID := strings.Repeat("L", accessKeyIDLength)

	// A correctly shaped pair must stay quiet; a warning on a working configuration is noise that
	// teaches the reader to ignore the line.
	if _, err := NewAliyunSender(plausibleID, strings.Repeat("s", accessKeySecretLength), "kada", "SMS_000000000"); err != nil {
		t.Fatalf("NewAliyunSender with a plausible pair: %v", err)
	}
	if strings.Contains(logged.String(), "do not look like") {
		t.Errorf("a plausible pair was warned about: %s", logged.String())
	}

	logged.Reset()

	// The production failure: a valid-looking id and a 12-character secret.
	if _, err := NewAliyunSender(plausibleID, "SHORT-SECRET", "kada", "SMS_000000000"); err != nil {
		t.Fatalf("NewAliyunSender with a short secret: %v", err)
	}
	warning := logged.String()
	if !strings.Contains(warning, "do not look like") {
		t.Fatalf("a 12-character secret was accepted silently: %s", warning)
	}
	for _, want := range []string{"SMS_ACCESS_KEY_SECRET", "12", "30", "SignatureDoesNotMatch"} {
		if !strings.Contains(warning, want) {
			t.Errorf("warning should mention %q, got: %s", want, warning)
		}
	}
}
