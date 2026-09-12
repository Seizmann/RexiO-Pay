// Package sms parses incoming bKash and Nagad payment notifications.
package sms

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

var (
	ErrUnsupportedProvider = errors.New("sms: unsupported provider")
	ErrNoMatch             = errors.New("sms: no matching template")
	ErrInvalidAmount       = errors.New("sms: invalid amount")
	ErrInvalidSender       = errors.New("sms: invalid sender number")
)

// ParsedSMS is the normalized data used by the matching engine. Optional
// values are omitted from JSON so a template with no balance or fee cannot be
// confused with a zero value.
type ParsedSMS struct {
	Provider     string `json:"provider"`
	AccountType  string `json:"account_type"`
	Amount       string `json:"amount"`
	SenderNumber string `json:"sender_number"`
	TrxID        string `json:"trx_id"`
	Balance      string `json:"balance,omitempty"`
	Fee          string `json:"fee,omitempty"`
	Reference    string `json:"reference,omitempty"`
	SMSTime      string `json:"sms_time,omitempty"`
}

// Result is retained as an alias for callers that use the parser's shorter
// result name.
type Result = ParsedSMS

// Parser matches provider-specific templates in descending priority order.
type Parser struct {
	version   string
	templates []Template
}

// New returns a parser using the built-in, versioned template corpus.
func New() *Parser {
	templates := append([]Template(nil), Patterns...)
	sort.SliceStable(templates, func(i, j int) bool {
		if templates[i].Provider != templates[j].Provider {
			return templates[i].Provider < templates[j].Provider
		}
		return templates[i].Priority > templates[j].Priority
	})
	return &Parser{version: ParserVersion, templates: templates}
}

// NewParser is an explicit alias for New.
func NewParser() *Parser { return New() }

// Version reports the parser version advertised to devices.
func (p *Parser) Version() string {
	if p == nil || p.version == "" {
		return ParserVersion
	}
	return p.version
}

// Parse parses raw using provider ("bkash" or "nagad"). The raw text is not
// changed or stored here; the ingest layer must persist it before calling the
// parser so a failed parse can still be reviewed.
func Parse(provider, raw string) (ParsedSMS, error) {
	return New().Parse(provider, raw)
}

// ParseSMS is a descriptive alias for Parse.
func ParseSMS(provider, raw string) (ParsedSMS, error) { return Parse(provider, raw) }

// Parse applies the priority-ordered patterns for provider.
func (p *Parser) Parse(provider, raw string) (ParsedSMS, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider != "bkash" && provider != "nagad" {
		return ParsedSMS{}, fmt.Errorf("%w: %q", ErrUnsupportedProvider, provider)
	}
	if strings.TrimSpace(raw) == "" {
		return ParsedSMS{}, ErrNoMatch
	}

	for _, template := range p.templates {
		if template.Provider != provider {
			continue
		}
		matches := template.re.FindStringSubmatch(normalizeWhitespace(raw))
		if matches == nil {
			continue
		}
		result, err := resultFromMatch(template, matches)
		if err != nil {
			return ParsedSMS{}, fmt.Errorf("%s: %w", template.Name, err)
		}
		return result, nil
	}
	return ParsedSMS{}, fmt.Errorf("%w for %s", ErrNoMatch, provider)
}

func resultFromMatch(template Template, matches []string) (ParsedSMS, error) {
	values := make(map[string]string, len(template.Fields))
	for i, field := range template.Fields {
		if i+1 < len(matches) {
			values[field] = strings.TrimSpace(matches[i+1])
		}
	}

	amount, err := sanitizeMoney(values["amount"])
	if err != nil {
		return ParsedSMS{}, err
	}
	sender := canonicalSender(values["sender_number"])
	if sender == "" {
		return ParsedSMS{}, ErrInvalidSender
	}

	result := ParsedSMS{
		Provider:     template.Provider,
		AccountType:  template.AccountType,
		Amount:       amount,
		SenderNumber: sender,
		TrxID:        strings.ToUpper(values["trx_id"]),
		Balance:      optionalMoney(values["balance"]),
		Fee:          optionalMoney(values["fee"]),
		Reference:    values["reference"],
		SMSTime:      values["sms_time"],
	}
	return result, nil
}

func normalizeWhitespace(s string) string {
	var normalized strings.Builder
	for _, r := range s {
		switch {
		case r >= '০' && r <= '৯':
			normalized.WriteRune('0' + (r - '০'))
		default:
			normalized.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(strings.TrimSpace(normalized.String())), " ")
}

func sanitizeMoney(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	if value == "" {
		return "", ErrInvalidAmount
	}
	digits := 0
	dots := 0
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == '.':
			dots++
		default:
			return "", ErrInvalidAmount
		}
	}
	if digits == 0 || dots > 1 {
		return "", ErrInvalidAmount
	}
	return value, nil
}

func optionalMoney(value string) string {
	if value == "" {
		return ""
	}
	clean, err := sanitizeMoney(value)
	if err != nil {
		return ""
	}
	return clean
}

func canonicalSender(value string) string {
	var digits strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		} else if unicode.IsSpace(r) || r == '+' || r == '-' || r == '(' || r == ')' {
			continue
		} else {
			return ""
		}
	}
	number := digits.String()
	switch {
	case strings.HasPrefix(number, "880"):
		number = "0" + number[3:]
	case strings.HasPrefix(number, "88"):
		number = "0" + number[2:]
	}
	if len(number) != 11 || !strings.HasPrefix(number, "01") {
		return ""
	}
	return number
}
