-- name: CreateProfile :one
INSERT INTO public.payment_profiles (
    id, merchant_id, provider, account_type, mfs_number, display_name,
    balance_verification, is_otp_verified
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetProfile :one
SELECT * FROM public.payment_profiles
WHERE id = $1 AND merchant_id = $2
LIMIT 1;

-- name: GetProfileByMerchantAndID :one
SELECT * FROM public.payment_profiles
WHERE id = $1 AND merchant_id = $2
LIMIT 1;

-- name: ListProfilesForMerchant :many
SELECT pp.*,
       d.name  AS device_name,
       d.status AS device_status
FROM public.payment_profiles pp
LEFT JOIN public.devices d ON d.id = pp.device_id
WHERE pp.merchant_id = $1
ORDER BY pp.created_at ASC;

-- name: UpdateProfile :exec
UPDATE public.payment_profiles
SET display_name = $3,
    sim_slot     = $4
WHERE id = $1 AND merchant_id = $2;

-- name: UpdateProfileStatus :exec
UPDATE public.payment_profiles
SET status = $3
WHERE id = $1 AND merchant_id = $2;

-- name: UpdateTrackedBalance :exec
UPDATE public.payment_profiles
SET tracked_balance        = $2,
    balance_last_synced_at = now()
WHERE id = $1;

-- name: BindDevice :exec
UPDATE public.payment_profiles
SET device_id = $3,
    sim_slot  = $4
WHERE id = $1 AND merchant_id = $2;

-- name: CountProfilesForMerchant :one
SELECT COUNT(*) FROM public.payment_profiles WHERE merchant_id = $1;
