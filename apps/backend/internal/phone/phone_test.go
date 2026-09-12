package phone

import "testing"

func TestCanonicalize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"01712345678", "01712345678"},
		{"+8801712345678", "01712345678"},
		{"8801712345678", "01712345678"},
		{"88 01712345678", "01712345678"},
		{"017-1234-5678", "01712345678"},
		{"+880-017-1234-5678", "01712345678"},
		{"880017-1234-5678", "01712345678"},
		{"", ""},
		{"1234567890", ""},
		{"0171234567", ""},   // 10 digits — too short
		{"017123456789", ""}, // 12 digits — too long
		{"02712345678", ""},  // doesn't start with 01
	}
	for _, tt := range tests {
		got := Canonicalize(tt.input)
		if got != tt.want {
			t.Errorf("Canonicalize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsValid(t *testing.T) {
	if !IsValid("01712345678") {
		t.Error("expected 01712345678 to be valid")
	}
	if IsValid("123") {
		t.Error("expected 123 to be invalid")
	}
}
