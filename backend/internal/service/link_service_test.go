package service

import (
	"strings"
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
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
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
		// 公式注入防护
		{"=1+1", "'=1+1"},
		{"+8613800000000", "'+8613800000000"},
		{"@SUM(A1)", "'@SUM(A1)"},
		{"-2+3", "'-2+3"},
		{"=HYPERLINK(\"http://evil\")", `"'=HYPERLINK(""http://evil"")"`},
		// 用户可控的 domain 字段含逗号/引号时正确包裹
		{`evil.com,"x`, `"evil.com,""x"`},
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
	url := svc.BuildShortURL("kada.click", "abc123")
	expected := "https://kada.click/r/abc123"
	if url != expected {
		t.Errorf("BuildShortURL = %q, want %q", url, expected)
	}

	// 自定义域名
	url2 := svc.BuildShortURL("custom.domain.com", "xyz789")
	expected2 := "https://custom.domain.com/r/xyz789"
	if url2 != expected2 {
		t.Errorf("BuildShortURL = %q, want %q", url2, expected2)
	}
}

func TestHashPassword(t *testing.T) {
	pwd := "test-password-123"
	hash1 := hashPassword(pwd)
	hash2 := hashPassword(pwd)

	// bcrypt 加盐：相同密码产生不同哈希
	if hash1 == "" || hash2 == "" {
		t.Fatal("hashPassword returned empty hash")
	}
	if hash1 == hash2 {
		t.Error("bcrypt hashes should differ due to salt")
	}
	if !strings.HasPrefix(hash1, "$2") {
		t.Errorf("expected bcrypt hash, got %q", hash1)
	}

	// 验证
	if !checkPasswordHash(pwd, hash1) {
		t.Error("checkPasswordHash should return true for correct password")
	}
	if checkPasswordHash("wrong-password", hash1) {
		t.Error("checkPasswordHash should return false for wrong password")
	}
}

func TestHashPassword_LongPassword(t *testing.T) {
	// bcrypt 上限 72 字节，超长密码应自动截断并仍可验证
	pwd := strings.Repeat("a", 100)
	hash := hashPassword(pwd)
	if hash == "" {
		t.Fatal("hashPassword returned empty hash for long password")
	}
	if !checkPasswordHash(pwd, hash) {
		t.Error("checkPasswordHash should accept truncated long password")
	}
}

func TestCheckPasswordHash_LegacySHA256(t *testing.T) {
	// 存量数据为未加盐 SHA-256 十六进制，升级后仍应可验证
	legacy := sha256Hex("legacy-pass-123")
	if !checkPasswordHash("legacy-pass-123", legacy) {
		t.Error("legacy SHA-256 hash should still verify")
	}
	if checkPasswordHash("wrong", legacy) {
		t.Error("legacy SHA-256 hash should reject wrong password")
	}
	if checkPasswordHash("legacy-pass-123", "") {
		t.Error("empty hash should never verify")
	}
}
