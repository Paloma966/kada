package service

import "testing"

func TestPhonePattern(t *testing.T) {
	valid := []string{"13812345678", "19912345678", "15800000000"}
	for _, p := range valid {
		if !phonePattern.MatchString(p) {
			t.Errorf("expected %q to be valid phone", p)
		}
	}
	invalid := []string{
		"", "123", "12345678901", // 长度错误
		"11812345678", "12812345678", // 第二位非法
		"1381234567a",    // 含字母
		"138 1234 5678",  // 含空格
		"aaaaaaaaaaa",    // 全字母
		"+8613812345678", // 带国家码
	}
	for _, p := range invalid {
		if phonePattern.MatchString(p) {
			t.Errorf("expected %q to be invalid phone", p)
		}
	}
}
