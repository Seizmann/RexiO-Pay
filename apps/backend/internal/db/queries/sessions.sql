-- name: CreateSession :one
INSERT INTO public.checkout_sessions (
    id, merchant_id, payment_profile_id, source, amount, currency,
    customer, metadata, return_url, cancel_url, payment_link_id, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: GetSession :one
SELECT * FROM public.checkout_sessions
WHERE id = $1 AND merchant_id = $2
LIMIT 1;

-- name: GetSessionForPublicCheckout :one
SELECT
    cs.id, cs.merchant_id, cs.payment_profile_id, cs.amount, cs.currency,
    cs.status, cs.needs_review, cs.review_reason, cs.expires_at, cs.created_at,
    cs.sender_number_claim, cs.trx_id_claim, cs.confirmed_at,
    m.name  AS merchant_name,
    m.logo_url AS merchant_logo_url,
    m.brand_color AS merchant_brand_color,
    m.support_email AS merchant_support_email,
    m.support_phone AS merchant_support_phone
FROM public.checkout_sessions cs
JOIN public.merchants m ON m.id = cs.merchant_id
WHERE cs.id = $1
LIMIT 1;

-- name: UpdateSessionStatus :exec
UPDATE public.checkout_sessions
SET status = $2, updated_at = now()
WHERE id = $1;

-- name: UpdateSessionNeedsReview :exec
UPDATE public.checkout_sessions
SET needs_review = $2, review_reason = $3, updated_at = now()
WHERE id = $1;

-- name: UpdateSessionClaim :exec
UPDATE public.checkout_sessions
SET sender_number_claim = $2,
    trx_id_claim        = $3,
    claim_submitted_at  = now(),
    updated_at          = now()
WHERE id = $1;

-- name: UpdateSessionConfirmed :exec
UPDATE public.checkout_sessions
SET confirmed_at = now(), updated_at = now()
WHERE id = $1;

-- name: IncrMerchantSessionCount :exec
UPDATE public.merchants
SET session_count_current_period = session_count_current_period + 1
WHERE id = $1;

-- name: FindMatchCandidates :many
SELECT cs.*
FROM public.checkout_sessions cs
WHERE cs.merchant_id          = $1
  AND cs.payment_profile_id   = $2
  AND cs.amount               BETWEEN $3 AND $4
  AND cs.sender_number_claim  = $5
  AND cs.status               = 'pending'
ORDER BY cs.created_at ASC;

-- name: FindExpiredMatchCandidates :many
SELECT cs.*
FROM public.checkout_sessions cs
WHERE cs.merchant_id         = $1
  AND cs.payment_profile_id  = $2
  AND cs.amount              BETWEEN $3 AND $4
  AND cs.sender_number_claim = $5
  AND cs.status              = 'pending'
  AND cs.expires_at          < now()
  AND cs.expires_at          > now() - interval '24 hours'
ORDER BY cs.created_at ASC;

-- name: ListSessions :many
SELECT * FROM public.checkout_sessions
WHERE merchant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountSessionsCurrentPeriod :one
SELECT session_count_current_period FROM public.merchants WHERE id = $1;

-- name: ResetSessionCount :exec
UPDATE public.merchants SET session_count_current_period = 0 WHERE id = $1;
