package service

import (
	"testing"
)

func TestGenerateShortCode(t *testing.T) {
	// 生成 100 个短码，确保都是 8 位十六进制且不重复
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code := generateShortCode()
		if len(code) != 8 {
			t.Errorf("expected short code length 8, got %d: %s", len(code), code)
		}
		// 检查是否只包含十六进制字符
		for _, c := range code {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("invalid character in short code: %c in %s", c, code)
			}
		}
		if codes[code] {
			t.Errorf("duplicate short code generated: %s", code)
		}
		codes[code] = true
	}
}

func TestShortCodePattern(t *testing.T) {
	tests := []struct {
		code    string
		isValid bool
		reason  string
	}{
		{"abc123", true, "字母数字组合"},
		{"my-link", true, "含连字符"},
		{"my_link", true, "含下划线"},
		{"a1b2c3d4", true, "8位字母数字"},
		{"AbC123", true, "大小写混合"},
		{"a1b2c3d4e5f6g7h8i9j0", true, "20位长度（上限）"},
		{"abc", false, "太短（3位）"},
		{"a", false, "太短（1位）"},
		{"ab", false, "太短（2位）"},
		{"abcdefghijklmnopqrstu", false, "太长（21位）"},
		{"abc def", false, "含空格"},
		{"abc@def", false, "含特殊字符@"},
		{"abc.def", false, "含点号"},
		{"中文测试", false, "含中文"},
		{"abc/def", false, "含斜杠"},
		{"", false, "空字符串"},
	}

	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			result := shortCodePattern.MatchString(tt.code)
			if result != tt.isValid {
				t.Errorf("shortCodePattern.MatchString(%q) = %v, want %v (%s)",
					tt.code, result, tt.isValid, tt.reason)
			}
		})
	}
}

func TestEscapeCSV(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"hello world", "hello world"},
		{"hello,world", `"hello,world"`},
		{`say "hello"`, `"say ""hello"""`},
		{"line1\nline2", "\"line1\nline2\""},
		{"normal text", "normal text"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeCSV(tt.input)
			if result != tt.expected {
				t.Errorf("escapeCSV(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildShortURL(t *testing.T) {
	svc := &LinkService{baseURL: "https://kada.click"}
	url := svc.buildShortURL("kada.click", "abc123")
	expected := "https://kada.click/r/abc123"
	if url != expected {
		t.Errorf("buildShortURL = %q, want %q", url, expected)
	}

	// 自定义域名
	url2 := svc.buildShortURL("custom.domain.com", "xyz789")
	expected2 := "https://custom.domain.com/r/xyz789"
	if url2 != expected2 {
		t.Errorf("buildShortURL = %q, want %q", url2, expected2)
	}
}

func TestHashPassword(t *testing.T) {
	pwd := "test-password-123"
	hash1 := hashPassword(pwd)
	hash2 := hashPassword(pwd)

	// 相同密码产生相同哈希
	if hash1 != hash2 {
		t.Error("hashPassword should be deterministic for same input")
	}

	// 哈希长度应为 64（SHA256 十六进制）
	if len(hash1) != 64 {
		t.Errorf("expected hash length 64, got %d", len(hash1))
	}

	// 验证
	if !checkPasswordHash(pwd, hash1) {
		t.Error("checkPasswordHash should return true for correct password")
	}
	if checkPasswordHash("wrong-password", hash1) {
		t.Error("checkPasswordHash should return false for wrong password")
	}
}
