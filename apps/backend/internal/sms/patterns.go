package sms

import "regexp"

// ParserVersion identifies the template set understood by this package. It is
// sent to devices so a server can require an app update when templates change.
const ParserVersion = "sms-v1"

// Account types supported by the MVP parser.
const (
	AccountPersonal = "personal"
	AccountAgent    = "agent"
	AccountMerchant = "merchant"
)

// Template describes one priority-ordered provider/account-type SMS format.
// Fields contains the names of the capture groups, in regexp capture order.
type Template struct {
	Provider    string
	AccountType string
	Priority    int
	Name        string
	Pattern     string
	Fields      []string
	re          *regexp.Regexp
}

func newTemplate(provider, accountType string, priority int, name, pattern string, fields ...string) Template {
	return Template{
		Provider: provider, AccountType: accountType, Priority: priority,
		Name: name, Pattern: pattern, Fields: fields,
		re: regexp.MustCompile(pattern),
	}
}

// Patterns is the complete parser corpus. Entries are ordered by priority
// within each provider; the parser also sorts a copy, making additions safe.
// The English formats mirror PipraPay's pp-functions.php templates. Bengali
// formats are intentionally kept as separate templates so each can be tested
// against a fixture before being changed.
var Patterns = []Template{
	// bKash
	newTemplate("bkash", AccountPersonal, 100, "bkash-personal-received",
		`(?i)You have received Tk ([0-9,.]+) from ([+0-9][0-9+ .-]*)\.\s*(?:Ref[: -]?\s*(\S+)\s*)?Fee Tk ([0-9,.]+)\.\s*Balance Tk ([0-9,.]+)\.\s*TrxID[: ]+([A-Za-z0-9]+)(?:\s+at\s+(.+?))?\s*$`,
		"amount", "sender_number", "reference", "fee", "balance", "trx_id", "sms_time"),
	newTemplate("bkash", AccountMerchant, 90, "bkash-merchant-received-payment",
		`(?i)You have received payment Tk ([0-9,.]+) from ([+0-9][0-9+ .-]*)\.\s*(?:Ref[: -]?\s*(\S+)\s*)?Fee Tk ([0-9,.]+)\.\s*Balance Tk ([0-9,.]+)\.\s*TrxID[: ]+([A-Za-z0-9]+)(?:\s+at\s+(.+?))?\s*$`,
		"amount", "sender_number", "reference", "fee", "balance", "trx_id", "sms_time"),
	newTemplate("bkash", AccountAgent, 80, "bkash-agent-cash-in",
		`(?i)Cash In Tk ([0-9,.]+) from ([+0-9][0-9+ .-]*) successful\.\s*(?:Fee Tk ([0-9,.]+)\.\s*)?Balance Tk ([0-9,.]+)\.\s*TrxID[: ]+([A-Za-z0-9]+)(?:\s+at\s+(.+?))?\s*$`,
		"amount", "sender_number", "fee", "balance", "trx_id", "sms_time"),
	newTemplate("bkash", AccountPersonal, 70, "bkash-personal-bengali",
		`আপনি ([0-9,.]+) টাকা পেয়েছেন ([+0-9][0-9+ .-]*) থেকে।\s*(?:রেফ[: -]?\s*(\S+)\s*)?ফি ([0-9,.]+) টাকা।\s*ব্যালেন্স ([0-9,.]+) টাকা।\s*ট্রানজেকশন আইডি[: ]+([A-Za-z0-9]+)(?:\s+সময়\s+(.+?))?\s*$`,
		"amount", "sender_number", "reference", "fee", "balance", "trx_id", "sms_time"),
	newTemplate("bkash", AccountMerchant, 60, "bkash-merchant-bengali",
		`আপনি ([0-9,.]+) টাকা পেমেন্ট পেয়েছেন ([+0-9][0-9+ .-]*) থেকে।\s*(?:রেফ[: -]?\s*(\S+)\s*)?ফি ([0-9,.]+) টাকা।\s*ব্যালেন্স ([0-9,.]+) টাকা।\s*ট্রানজেকশন আইডি[: ]+([A-Za-z0-9]+)(?:\s+সময়\s+(.+?))?\s*$`,
		"amount", "sender_number", "reference", "fee", "balance", "trx_id", "sms_time"),
	newTemplate("bkash", AccountAgent, 50, "bkash-agent-bengali",
		`ক্যাশ ইন ([0-9,.]+) টাকা ([+0-9][0-9+ .-]*) থেকে সফল।\s*ব্যালেন্স ([0-9,.]+) টাকা।\s*ট্রানজেকশন আইডি[: ]+([A-Za-z0-9]+)(?:\s+সময়\s+(.+?))?\s*$`,
		"amount", "sender_number", "balance", "trx_id", "sms_time"),

	// Nagad
	newTemplate("nagad", AccountPersonal, 100, "nagad-personal-money-received",
		`(?i)Money Received\.\s*Amount: Tk ([0-9,.]+)\s*Sender: ([+0-9][0-9+ .-]*)\s*(?:Ref[: -]?\s*(\S+)\s*)?TxnID: ([A-Za-z0-9]+)\s*Balance: Tk ([0-9,.]+)(?:\s+(.+?))?\s*$`,
		"amount", "sender_number", "reference", "trx_id", "balance", "sms_time"),
	newTemplate("nagad", AccountAgent, 90, "nagad-agent-cash-in",
		`(?i)Cash In Received\.\s*Amount: Tk ([0-9,.]+)\s*Uddokta: ([+0-9][0-9+ .-]*)\s*TxnID: ([A-Za-z0-9]+)\s*Balance: ?(?:Tk )?([0-9,.]+)(?:\s+(.+?))?\s*$`,
		"amount", "sender_number", "trx_id", "balance", "sms_time"),
	newTemplate("nagad", AccountMerchant, 80, "nagad-merchant-payment",
		`(?i)(?:You have )?received a payment of Tk ([0-9,.]+) from ([+0-9][0-9+ .-]*)\.\s*TrxID:?[ ]+([A-Za-z0-9]+)(?:\s+at\s+(.+?))?\s*$`,
		"amount", "sender_number", "trx_id", "sms_time"),
	newTemplate("nagad", AccountPersonal, 70, "nagad-personal-bengali",
		`আপনি ([0-9,.]+) টাকা পেয়েছেন।\s*প্রেরক: ([+0-9][0-9+ .-]*)\s*(?:রেফ[: -]?\s*(\S+)\s*)?লেনদেন আইডি[: ]+([A-Za-z0-9]+)\s*ব্যালেন্স: ?([0-9,.]+)(?:\s+(.+?))?\s*$`,
		"amount", "sender_number", "reference", "trx_id", "balance", "sms_time"),
	newTemplate("nagad", AccountAgent, 60, "nagad-agent-bengali",
		`ক্যাশ ইন গ্রহণ।\s*পরিমাণ: ([0-9,.]+) টাকা\s*উদ্যোক্তা: ([+0-9][0-9+ .-]*)\s*লেনদেন আইডি: ([A-Za-z0-9]+)\s*ব্যালেন্স: ?([0-9,.]+)(?:\s+(.+?))?\s*$`,
		"amount", "sender_number", "trx_id", "balance", "sms_time"),
	newTemplate("nagad", AccountMerchant, 50, "nagad-merchant-bengali",
		`আপনি ([0-9,.]+) টাকা পেমেন্ট পেয়েছেন ([+0-9][0-9+ .-]*) থেকে।\s*লেনদেন আইডি: ([A-Za-z0-9]+)(?:\s+সময়\s+(.+?))?\s*$`,
		"amount", "sender_number", "trx_id", "sms_time"),
}
