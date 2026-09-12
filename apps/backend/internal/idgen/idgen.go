// Package idgen generates prefixed, nanoid-based IDs for all entities.
// All IDs are server-generated text values — never client-provided.
package idgen

import gonanoid "github.com/matoous/go-nanoid/v2"

// ID prefixes — one per entity type (REQUIREMENT.md §8).
const (
	PrefixMerchant = "mer_"
	PrefixSession  = "cs_"
	PrefixPayment  = "pay_"
	PrefixSMS      = "sms_"
	PrefixProfile  = "pp_"
	PrefixDevice   = "dev_"
	PrefixWebhook  = "whk_"
	PrefixAPIKey   = "apk_"
	PrefixAudit    = "aud_"
	PrefixDomain   = "dom_"
	PrefixLink     = "lnk_"
	PrefixOTP      = "otp_"
	PrefixAnnounce = "ann_"
	PrefixDelivery = "dlv_"
)

// New returns a new ID with the given prefix followed by a 21-character nanoid.
func New(prefix string) string {
	id, err := gonanoid.New(21)
	if err != nil {
		// gonanoid only errors if the alphabet or size is invalid — both are fixed constants.
		panic("idgen.New: " + err.Error())
	}
	return prefix + id
}
