-- name: CreateWebhookEndpoint :one
INSERT INTO public.webhook_endpoints (
    id, merchant_id, url, secret, events
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetWebhookEndpoint :one
SELECT * FROM public.webhook_endpoints
WHERE id = $1 AND merchant_id = $2
LIMIT 1;

-- name: ListWebhookEndpoints :many
SELECT * FROM public.webhook_endpoints
WHERE merchant_id = $1
ORDER BY created_at DESC;

-- name: DeleteWebhookEndpoint :exec
UPDATE public.webhook_endpoints
SET status = 'disabled'
WHERE id = $1 AND merchant_id = $2;

-- name: CountWebhookEndpoints :one
SELECT COUNT(*) FROM public.webhook_endpoints
WHERE merchant_id = $1 AND status = 'active';

-- name: CreateWebhookDelivery :one
INSERT INTO public.webhook_deliveries (
    id, endpoint_id, merchant_id, event_type, payload
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetPendingDeliveries :many
SELECT wd.*, we.url, we.secret, we.status AS endpoint_status
FROM public.webhook_deliveries wd
JOIN public.webhook_endpoints we ON we.id = wd.endpoint_id
WHERE wd.status       = 'pending'
  AND wd.next_retry_at <= now()
  AND we.status        = 'active'
ORDER BY wd.next_retry_at ASC
LIMIT $1;

-- name: UpdateDeliverySuccess :exec
UPDATE public.webhook_deliveries
SET status       = 'delivered',
    delivered_at = now(),
    last_response_code = $2
WHERE id = $1;

-- name: UpdateDeliveryFailed :exec
UPDATE public.webhook_deliveries
SET attempt_count      = attempt_count + 1,
    next_retry_at      = $2,
    status             = $3,
    last_response_code = $4
WHERE id = $1;

-- name: ResetDelivery :exec
UPDATE public.webhook_deliveries
SET attempt_count = 0,
    status        = 'pending',
    next_retry_at = now()
WHERE id = $1 AND merchant_id = $2;

-- name: ListDeliveriesForEndpoint :many
SELECT * FROM public.webhook_deliveries
WHERE endpoint_id = $1 AND merchant_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;
