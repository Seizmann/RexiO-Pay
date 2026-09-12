-- name: CreateOTPVerification :one
INSERT INTO public.mfs_otp_verifications (
    id, merchant_id, mfs_number, provider, account_type
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetPendingOTPByNumber :one
SELECT * FROM public.mfs_otp_verifications
WHERE mfs_number = $1
  AND provider   = $2
  AND status     = 'pending'
  AND expires_at > now()
ORDER BY created_at DESC
LIMIT 1;

-- name: GetOTPVerification :one
SELECT * FROM public.mfs_otp_verifications
WHERE id = $1 AND merchant_id = $2
LIMIT 1;

-- name: MarkOTPVerified :exec
UPDATE public.mfs_otp_verifications
SET status = 'verified'
WHERE id = $1;

-- name: ExpireOldOTPs :exec
UPDATE public.mfs_otp_verifications
SET status = 'expired'
WHERE status = 'pending' AND expires_at < now();
