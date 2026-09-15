package sms

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
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

func NewAliyunSender(accessKeyID, accessKeySecret, signName, templateCode string) (*AliyunSender, error) {
	if accessKeyID == "" || accessKeySecret == "" {
		return nil, errors.New("Aliyun SMS credentials are empty; set SMS_ACCESS_KEY_ID and SMS_ACCESS_KEY_SECRET")
	}

	// The signature and the template are not defaulted. This service (dypnsapi, "SMS verification") ships
	// one system-granted signature and one system-granted template per account, and their names only exist
	// in the console - they cannot be guessed or created by hand. Substituting a value here, as this used
	// to do with "恒创联众" / "100001", converts "nobody configured this" into a provider rejection that
	// looks like a broken account, which is how it went undiagnosed.
	if signName == "" {
		return nil, errors.New("SMS_SIGN_NAME is empty; copy the system-granted signature name from the PNVS console")
	}
	if templateCode == "" {
		return nil, errors.New("SMS_TEMPLATE_CODE is empty; copy the system-granted template code from the PNVS console")
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

func (s *AliyunSender) SendVerificationCode(phone string) (code string, err error) {
	code = generateCode()

	request := &dypnsapi.SendSmsVerifyCodeRequest{
		PhoneNumber:   tea.String(phone),
		SchemeName:    tea.String("SMS"),
		SignName:      tea.String(s.signName),
		TemplateCode:  tea.String(s.templateCode),
		TemplateParam: tea.String(fmt.Sprintf(`{"code":"%s","min":"5"}`, code)),
	}

	response, err := s.client.SendSmsVerifyCode(request)
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

func generateCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n.Int64())
}
