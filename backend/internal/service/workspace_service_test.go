package service

import (
	"testing"
)

func TestSlugPattern(t *testing.T) {
	tests := []struct {
		slug    string
		isValid bool
		reason  string
	}{
		{"my-project", true, "标准 slug"},
		{"test123", true, "字母数字"},
		{"a-b-c", true, "含多个连字符"},
		{"abc", true, "3 位（下限）"},
		{"a1b", true, "3 位混合"},
		{"a", false, "太短（1位）"},
		{"ab", false, "太短（2位）"},
		{"-abc", false, "以连字符开头"},
		{"abc-", false, "以连字符结尾"},
		{"ABC-DEF", false, "含大写字母"},
		{"abc_def", false, "含下划线"},
		{"abc def", false, "含空格"},
		{"abc.def", false, "含点号"},
		{"", false, "空字符串"},
	}

	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			result := slugPattern.MatchString(tt.slug)
			if result != tt.isValid {
				t.Errorf("slugPattern.MatchString(%q) = %v, want %v (%s)",
					tt.slug, result, tt.isValid, tt.reason)
			}
		})
	}
}

func TestSlugValidExamples(t *testing.T) {
	validSlugs := []string{
		"my-project",
		"personal",
		"team-alpha",
		"project-2024",
		"client-work",
		"a1b2c3",
		"dev",
	}

	for _, slug := range validSlugs {
		if !slugPattern.MatchString(slug) {
			t.Errorf("expected %q to be valid slug", slug)
		}
	}
}

func TestSlugInvalidExamples(t *testing.T) {
	invalidSlugs := []string{
		"",
		"ab",
		"-start",
		"end-",
		"UPPERCASE",
		"has space",
		"special!char",
	}

	for _, slug := range invalidSlugs {
		if slugPattern.MatchString(slug) {
			t.Errorf("expected %q to be invalid slug", slug)
		}
	}
}
