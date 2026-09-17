package sms

import (
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
