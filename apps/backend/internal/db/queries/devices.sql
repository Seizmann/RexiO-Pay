-- name: CreateDevice :one
INSERT INTO public.devices (
    id, merchant_id, name, pairing_token, pairing_expires_at
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetDevice :one
SELECT * FROM public.devices WHERE id = $1 LIMIT 1;

-- name: GetDeviceByPairingToken :one
SELECT * FROM public.devices
WHERE pairing_token = $1 AND pairing_expires_at > now()
LIMIT 1;

-- name: StorePairedDevice :exec
UPDATE public.devices
SET secret_ciphertext = $2,
    secret_nonce      = $3,
    pairing_token     = NULL,
    pairing_expires_at = NULL,
    status            = 'active',
    failed_sig_count  = 0
WHERE id = $1;

-- name: UpdateDeviceStatus :exec
UPDATE public.devices SET status = $2 WHERE id = $1;

-- name: UpdateHeartbeat :exec
UPDATE public.devices
SET last_heartbeat_at = now(),
    app_version       = $2
WHERE id = $1;

-- name: UpdateLastSMSSynced :exec
UPDATE public.devices SET last_sms_synced_at = now() WHERE id = $1;

-- name: IncrFailedSigCount :one
UPDATE public.devices
SET failed_sig_count = failed_sig_count + 1
WHERE id = $1
RETURNING failed_sig_count;

-- name: ResetFailedSigCount :exec
UPDATE public.devices SET failed_sig_count = 0 WHERE id = $1;

-- name: DisableDevice :exec
UPDATE public.devices SET status = 'disabled', failed_sig_count = 0 WHERE id = $1;

-- name: ListDevicesForMerchant :many
SELECT * FROM public.devices WHERE merchant_id = $1 ORDER BY created_at DESC;

-- name: RegeneratePairingToken :exec
UPDATE public.devices
SET pairing_token      = $2,
    pairing_expires_at = $3
WHERE id = $1;

-- name: ListStaleDevices :many
SELECT * FROM public.devices
WHERE status           = 'active'
  AND last_heartbeat_at < now() - interval '3 minutes';

-- name: ListAllDevicesForAdmin :many
SELECT d.*, m.name AS merchant_name
FROM public.devices d
JOIN public.merchants m ON m.id = d.merchant_id
ORDER BY d.last_heartbeat_at DESC NULLS LAST
LIMIT $1 OFFSET $2;
