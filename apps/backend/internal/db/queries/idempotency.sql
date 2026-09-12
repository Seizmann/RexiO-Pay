-- name: UpsertIdempotencyKey :exec
INSERT INTO public.idempotency_keys (key, merchant_id, request_hash, response_body, status_code)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (key) DO NOTHING;

-- name: GetIdempotencyKey :one
SELECT * FROM public.idempotency_keys
WHERE key         = $1
  AND merchant_id = $2
  AND expires_at  > now()
LIMIT 1;

-- name: DeleteExpiredIdempotencyKeys :exec
DELETE FROM public.idempotency_keys WHERE expires_at < now();
