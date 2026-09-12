-- name: GetMerchant :one
SELECT * FROM public.merchants WHERE id = $1 LIMIT 1;

-- name: GetMerchantByOwner :one
SELECT * FROM public.merchants WHERE owner_user_id = $1 LIMIT 1;

-- name: UpdateMerchantPlan :exec
UPDATE public.merchants
SET plan_id        = $2,
    plan_status    = $3,
    plan_renews_at = $4
WHERE id = $1;

-- name: UpdateMerchantBranding :exec
UPDATE public.merchants
SET name          = $2,
    logo_url      = $3,
    favicon_url   = $4,
    brand_color   = $5,
    support_email = $6,
    support_phone = $7
WHERE id = $1;

-- name: UpdateMerchantSettings :exec
UPDATE public.merchants
SET settings = $2
WHERE id = $1;

-- name: ListMerchantsAdmin :many
SELECT m.*,
       (SELECT COUNT(*) FROM public.devices        WHERE merchant_id = m.id) AS device_count,
       (SELECT COUNT(*) FROM public.checkout_sessions WHERE merchant_id = m.id AND status = 'succeeded') AS total_payments
FROM public.merchants m
ORDER BY m.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetMerchantAdmin :one
SELECT m.*,
       (SELECT COUNT(*) FROM public.devices        WHERE merchant_id = m.id) AS device_count,
       (SELECT COUNT(*) FROM public.checkout_sessions WHERE merchant_id = m.id) AS total_sessions
FROM public.merchants m
WHERE m.id = $1;

-- name: GetPlan :one
SELECT * FROM public.plans WHERE code = $1 LIMIT 1;

-- name: DisableMerchant :exec
UPDATE public.merchants
SET plan_status = 'disabled'
WHERE id = $1;
