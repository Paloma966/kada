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
		"", "123", "12345678901", // wrong length
		"11812345678", "12812345678", // invalid second digit
		"1381234567a",    // contains a letter
		"138 1234 5678",  // contains a space
		"aaaaaaaaaaa",    // all letters
		"+8613812345678", // with country code
	}
	for _, p := range invalid {
		if phonePattern.MatchString(p) {
			t.Errorf("expected %q to be invalid phone", p)
		}
	}
}
