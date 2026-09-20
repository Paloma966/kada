package sms

import (
	"errors"
	"fmt"
	"log"
	"os"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dypnsapi "github.com/alibabacloud-go/dypnsapi-20170525/v3/client"
	"github.com/alibabacloud-go/tea/tea"
)

type AliyunSender struct {
	client       *dypnsapi.Client
	signName     string
	templateCode string
}

// The shapes Aliyun issues: a 24-character AccessKey id (LTAI...) and a 30-character secret.
//
// They are used only to warn. A value that is present, non-empty and the wrong size is the mistake that
// hides best - it passes every emptiness check, the process starts, and the provider answers
// "SignatureDoesNotMatch" only when somebody finally asks for a code, which reads like a broken account
// or a code bug. A 12-character secret did exactly that in production.
const (
	accessKeyIDLength     = 24
	accessKeySecretLength = 30
)

func NewAliyunSender(accessKeyID, accessKeySecret, signName, templateCode string) (*AliyunSender, error) {
	if accessKeyID == "" || accessKeySecret == "" {
		return nil, errors.New("missing Aliyun AccessKey pair: set SMS_ACCESS_KEY_ID and SMS_ACCESS_KEY_SECRET")
	}

	// The signature and the template are not defaulted. This service (dypnsapi, "SMS verification") ships
	// one system-granted signature and one system-granted template per account, and their names only exist
	// in the console - they cannot be guessed or created by hand. Substituting a value here, as this used
	// to do with "恒创联众" / "100001", converts "nobody configured this" into a provider rejection that
	// looks like a broken account, which is how it went undiagnosed.
	if signName == "" {
		return nil, errors.New("missing SMS signature: copy the system-granted sign name from the PNVS console into SMS_SIGN_NAME")
	}
	if templateCode == "" {
		return nil, errors.New("missing SMS template: copy the system-granted template code from the PNVS console into SMS_TEMPLATE_CODE")
	}

	// Warn, do not refuse to start: the misconfiguration only blocks sign-in, while refusing would also
	// take down short-link redirection, which is the part of the product that has to keep working.
	if len(accessKeyID) < accessKeyIDLength || len(accessKeySecret) < accessKeySecretLength {
		log.Printf("⚠️  SMS credentials do not look like an Aliyun AccessKey pair: "+
			"SMS_ACCESS_KEY_ID is %d characters (expected %d) and SMS_ACCESS_KEY_SECRET is %d (expected %d). "+
			"Aliyun will reject every send with SignatureDoesNotMatch, so NOBODY CAN SIGN IN until the real "+
			"secret from the console is set",
			len(accessKeyID), accessKeyIDLength, len(accessKeySecret), accessKeySecretLength)
	}

	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKeyID),
		AccessKeySecret: tea.String(accessKeySecret),
	}
	config.Endpoint = tea.String("dypnsapi.aliyuncs.com")

	client, err := dypnsapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create SMS verification client: %w", err)
	}

	// Logged because the pair is the first thing to check when a send is rejected: the signature and the
	// template have to come from the same account and be used together.
	log.Printf("✅ Aliyun SMS verification service initialized (sign_name=%q template_code=%q)", signName, templateCode)
	return &AliyunSender{client: client, signName: signName, templateCode: templateCode}, nil
}

// buildSendRequest assembles the call. It is split out from the send so the contract it encodes can be
// asserted without a network round trip.
//
// The provider generates the code, not this service: the template variable carries the placeholder
// "##code##" and Aliyun substitutes a code of its own for it. A code of our own in that field is rejected
// with isv.INVALID_PARAMETERS before any message is sent, which is exactly how the first real send
// failed. ReturnVerifyCode asks the provider to hand the generated code back, so verification stays where
// it already is - hashed into the sms_codes table with an expiry and an attempt limit - instead of being
// delegated to Aliyun afterwards.
//
// CodeLength is explicit because its default is 4 while the sign-in form, the stored hash and the input
// field all expect six digits. CodeType 1 is digits only, matching that numeric input. ValidTime repeats
// the five minutes the code row lives for, so both sides agree on the deadline.
func (s *AliyunSender) buildSendRequest(phone string) *dypnsapi.SendSmsVerifyCodeRequest {
	return &dypnsapi.SendSmsVerifyCodeRequest{
		PhoneNumber:      tea.String(phone),
		SchemeName:       tea.String("SMS"),
		SignName:         tea.String(s.signName),
		TemplateCode:     tea.String(s.templateCode),
		TemplateParam:    tea.String(`{"code":"##code##","min":"5"}`),
		CodeLength:       tea.Int64(6),
		CodeType:         tea.Int64(1),
		ValidTime:        tea.Int64(300),
		ReturnVerifyCode: tea.Bool(true),
	}
}

func (s *AliyunSender) SendVerificationCode(phone string) (code string, err error) {
	response, err := s.client.SendSmsVerifyCode(s.buildSendRequest(phone))
	if err != nil {
		log.Printf("aliyun SMS request failed (sign_name=%q template_code=%q phone=%s): %v",
			s.signName, s.templateCode, maskPhone(phone), err)
		return "", fmt.Errorf("failed to send verification code: %w", err)
	}

	// Security: never log the verification code in plaintext; mask the phone number.
	// Every field below is a pointer in the SDK, so use the tea accessors: when the provider rejects the
	// request the Body/Model fields may be nil and a plain dereference would panic the whole process.
	if response == nil || response.Body == nil {
		log.Printf("aliyun SMS returned no body (sign_name=%q template_code=%q phone=%s)",
			s.signName, s.templateCode, maskPhone(phone))
		return "", errors.New(smsProviderError("", ""))
	}

	providerCode := tea.StringValue(response.Body.Code)
	if providerCode != "OK" {
		providerMessage := tea.StringValue(response.Body.Message)
		log.Printf("aliyun SMS rejected (sign_name=%q template_code=%q phone=%s): code=%q message=%q",
			s.signName, s.templateCode, maskPhone(phone), providerCode, providerMessage)
		return "", errors.New(smsProviderError(providerCode, providerMessage))
	}

	// The message is out and the code in it is the provider's. Returning "" here would store a hash of
	// nothing and leave the recipient holding a code that cannot ever be verified, so refuse loudly
	// instead: the caller reports a failed send and the user can ask for another one.
	if response.Body.Model == nil || tea.StringValue(response.Body.Model.VerifyCode) == "" {
		log.Printf("aliyun SMS accepted the request but returned no verification code "+
			"(sign_name=%q template_code=%q phone=%s); the code it sent cannot be verified here",
			s.signName, s.templateCode, maskPhone(phone))
		return "", errors.New(smsProviderError(providerCode, "the provider did not return the generated code"))
	}
	code = tea.StringValue(response.Body.Model.VerifyCode)

	log.Printf("📱 verification code sent to %s", maskPhone(phone))
	return code, nil
}

// smsProviderError turns an Aliyun error payload into an actionable message.
// The provider code and message are what actually identify the misconfiguration (unapproved signature,
// unapproved template, disabled AccessKey, overdue account, ...); the cause is appended only outside
// release mode so production responses never leak internal provider details.
func smsProviderError(code, message string) string {
	base := "failed to send SMS, please try again later"
	if code == "" && message == "" {
		return base
	}
	if os.Getenv("GIN_MODE") == "release" {
		return fmt.Sprintf("%s (provider code: %s)", base, code)
	}
	return fmt.Sprintf("%s (provider code: %s, message: %s)", base, code, message)
}

// maskPhone masks a phone number: 138****1234
func maskPhone(phone string) string {
	if len(phone) < 7 {
		return "***"
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

func (s *AliyunSender) CheckVerificationCode(phone, code string) (bool, error) {
	request := &dypnsapi.CheckSmsVerifyCodeRequest{
		PhoneNumber: tea.String(phone),
		VerifyCode:  tea.String(code),
		SchemeName:  tea.String("SMS"),
	}

	response, err := s.client.CheckSmsVerifyCode(request)
	if err != nil {
		return false, fmt.Errorf("failed to verify verification code: %w", err)
	}

	if tea.StringValue(response.Body.Code) != "OK" {
		return false, nil
	}

	return tea.StringValue(response.Body.Model.VerifyResult) == "PASS", nil
}
