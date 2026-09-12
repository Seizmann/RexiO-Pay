-- name: CreatePaymentLink :one
INSERT INTO public.payment_links (
    id, merchant_id, payment_profile_id, title, amount, reusable, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetPaymentLink :one
SELECT * FROM public.payment_links
WHERE id = $1 AND merchant_id = $2
LIMIT 1;

-- name: ListPaymentLinks :many
SELECT * FROM public.payment_links
WHERE merchant_id = $1
ORDER BY created_at DESC;

-- name: UpdatePaymentLinkStatus :exec
UPDATE public.payment_links SET active = $3 WHERE id = $1 AND merchant_id = $2;
