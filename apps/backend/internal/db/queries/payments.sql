-- name: InsertPayment :one
INSERT INTO public.payments (
    id, merchant_id, session_id, sms_id, provider, account_type,
    sender_number, trx_id, amount, balance_after, verified_at, verification_source
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
ON CONFLICT (session_id, trx_id) DO NOTHING
RETURNING *;

-- name: GetPayment :one
SELECT * FROM public.payments
WHERE id = $1 AND merchant_id = $2
LIMIT 1;

-- name: ListPayments :many
SELECT * FROM public.payments
WHERE merchant_id = $1
  AND ($2::timestamptz IS NULL OR created_at >= $2)
  AND ($3::timestamptz IS NULL OR created_at <= $3)
  AND ($4::text        IS NULL OR provider   = $4)
ORDER BY created_at DESC
LIMIT $5 OFFSET $6;

-- name: CountPayments :one
SELECT COUNT(*) FROM public.payments
WHERE merchant_id = $1
  AND ($2::timestamptz IS NULL OR created_at >= $2)
  AND ($3::timestamptz IS NULL OR created_at <= $3)
  AND ($4::text        IS NULL OR provider   = $4);
