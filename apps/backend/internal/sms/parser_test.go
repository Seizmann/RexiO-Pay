package sms

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type fixture struct {
	Name     string    `json:"name"`
	Provider string    `json:"provider"`
	Account  string    `json:"account_type"`
	Raw      string    `json:"raw"`
	Expected ParsedSMS `json:"expected"`
}

func TestParserFixtures(t *testing.T) {
	parser := New()
	paths, err := filepath.Glob(filepath.Join("..", "..", "sms-fixtures", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no SMS fixtures found")
	}
	for _, path := range paths {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var fixtures []fixture
			if err := json.Unmarshal(data, &fixtures); err != nil {
				t.Fatal(err)
			}
			for _, tc := range fixtures {
				t.Run(tc.Name, func(t *testing.T) {
					got, err := parser.Parse(tc.Provider, tc.Raw)
					if err != nil {
						t.Fatalf("Parse() error = %v", err)
					}
					if !reflect.DeepEqual(got, tc.Expected) {
						t.Errorf("Parse() = %#v, want %#v", got, tc.Expected)
					}
				})
			}
		})
	}
}

func TestParserErrors(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		raw      string
		want     error
	}{
		{"unsupported provider", "rocket", "anything", ErrUnsupportedProvider},
		{"empty message", "bkash", "", ErrNoMatch},
		{"unmatched message", "nagad", "Your account was updated", ErrNoMatch},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.provider, tc.raw)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Parse() error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestParserVersion(t *testing.T) {
	if got := New().Version(); got != ParserVersion {
		t.Fatalf("Version() = %q, want %q", got, ParserVersion)
	}
}
