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
	client *dypnsapi.Client
}

func NewAliyunSender(accessKeyID, accessKeySecret string) (*AliyunSender, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKeyID),
		AccessKeySecret: tea.String(accessKeySecret),
	}
	config.Endpoint = tea.String("dypnsapi.aliyuncs.com")

	client, err := dypnsapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("创建短信认证客户端失败: %w", err)
	}

	log.Println("✅ 阿里云短信认证服务已初始化")
	return &AliyunSender{client: client}, nil
}

func (s *AliyunSender) SendVerificationCode(phone string) (code string, err error) {
	code = generateCode()

	request := &dypnsapi.SendSmsVerifyCodeRequest{
		PhoneNumber:   tea.String(phone),
		SchemeName:    tea.String("SMS"),
		SignName:      tea.String("恒创联众"),
		TemplateCode:  tea.String("100001"),
		TemplateParam: tea.String(fmt.Sprintf(`{"code":"%s","min":"5"}`, code)),
	}

	response, err := s.client.SendSmsVerifyCode(request)
	if err != nil {
		return "", fmt.Errorf("验证码发送失败: %w", err)
	}

	if *response.Body.Code != "OK" {
		return "", fmt.Errorf("验证码发送失败 [%s]: %s",
			*response.Body.Code, *response.Body.Message)
	}

	log.Printf("📱 验证码已发送: %s, Code: %s", phone, code)
	return code, nil
}

func (s *AliyunSender) CheckVerificationCode(phone, code string) (bool, error) {
	request := &dypnsapi.CheckSmsVerifyCodeRequest{
		PhoneNumber: tea.String(phone),
		VerifyCode:  tea.String(code),
		SchemeName:  tea.String("SMS"),
	}

	response, err := s.client.CheckSmsVerifyCode(request)
	if err != nil {
		return false, fmt.Errorf("验证码校验失败: %w", err)
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
