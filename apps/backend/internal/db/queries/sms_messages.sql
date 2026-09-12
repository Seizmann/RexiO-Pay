-- name: InsertSMSMessage :one
INSERT INTO public.sms_messages (
    id, merchant_id, device_id, payment_profile_id, provider, account_type,
    raw_text, parse_status, match_status, source, sim_slot, received_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: UpdateSMSParsed :exec
UPDATE public.sms_messages
SET parsed       = $2,
    parse_status = 'parsed',
    provider     = $3,
    account_type = $4
WHERE id = $1;

-- name: UpdateSMSMatchStatus :exec
UPDATE public.sms_messages
SET match_status       = $2,
    matched_session_id = $3
WHERE id = $1;

-- name: FindDuplicateTrxID :one
SELECT id FROM public.sms_messages
WHERE provider     = $1
  AND parse_status = 'parsed'
  AND parsed->>'trx_id' = $2
LIMIT 1;

-- name: GetSMSMessage :one
SELECT * FROM public.sms_messages WHERE id = $1 LIMIT 1;

-- name: GetHeldSMSForProfile :many
SELECT * FROM public.sms_messages
WHERE payment_profile_id = $1
  AND match_status       = 'held'
  AND parsed->>'sender_number' = $2
  AND (parsed->>'amount')::bigint = $3
ORDER BY received_at ASC;

-- name: ListUnmatchedSMS :many
SELECT * FROM public.sms_messages
WHERE merchant_id  = $1
  AND match_status IN ('unmatched', 'held', 'suspect', 'failed')
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListSMSStatsByProvider :many
SELECT
    provider,
    parse_status,
    DATE(created_at AT TIME ZONE 'Asia/Dhaka') AS day,
    COUNT(*)::bigint AS cnt
FROM public.sms_messages
WHERE created_at > now() - interval '30 days'
GROUP BY provider, parse_status, day
ORDER BY day DESC, provider;
