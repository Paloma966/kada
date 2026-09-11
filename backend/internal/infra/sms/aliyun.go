package sms

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"

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
	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKeyID),
		AccessKeySecret: tea.String(accessKeySecret),
	}
	config.Endpoint = tea.String("dypnsapi.aliyuncs.com")

	client, err := dypnsapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create SMS verification client: %w", err)
	}

	// Fall back to defaults when not explicitly configured so it works out of the box; production should
	// override them through the SMS_SIGN_NAME / SMS_TEMPLATE_CODE environment variables
	if signName == "" {
		signName = "恒创联众"
	}
	if templateCode == "" {
		templateCode = "100001"
	}

	log.Println("✅ Aliyun SMS verification service initialized")
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
		return "", fmt.Errorf("failed to send verification code: %w", err)
	}

	if *response.Body.Code != "OK" {
		return "", fmt.Errorf("failed to send verification code [%s]: %s",
			*response.Body.Code, *response.Body.Message)
	}

	// Security: never log the verification code in plaintext; mask the phone number
	log.Printf("📱 verification code sent to %s", maskPhone(phone))
	return code, nil
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

	if *response.Body.Code != "OK" {
		return false, nil
	}

	return *response.Body.Model.VerifyResult == "PASS", nil
}

func generateCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n.Int64())
}
