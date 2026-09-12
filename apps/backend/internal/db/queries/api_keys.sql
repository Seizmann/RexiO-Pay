-- name: CreateAPIKey :one
INSERT INTO public.api_keys (
    id, merchant_id, name, key_prefix, key_hash, scopes
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetAPIKeyByHash :one
SELECT * FROM public.api_keys
WHERE key_hash   = $1
  AND revoked_at IS NULL
LIMIT 1;

-- name: ListAPIKeys :many
SELECT id, merchant_id, name, key_prefix, scopes, last_used_at, created_at, revoked_at
FROM public.api_keys
WHERE merchant_id = $1
ORDER BY created_at DESC;

-- name: RevokeAPIKey :exec
UPDATE public.api_keys
SET revoked_at = now()
WHERE id = $1 AND merchant_id = $2;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE public.api_keys SET last_used_at = now() WHERE id = $1;

-- name: CountAPIKeysForMerchant :one
SELECT COUNT(*) FROM public.api_keys
WHERE merchant_id = $1 AND revoked_at IS NULL;
