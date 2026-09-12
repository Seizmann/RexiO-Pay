-- name: InsertAuditLog :exec
INSERT INTO public.audit_logs (
    id, merchant_id, actor_user_id, actor_device_id, action, entity, entity_id, details
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);

-- name: ListAuditLogs :many
SELECT * FROM public.audit_logs
WHERE ($1::text IS NULL OR merchant_id = $1)
  AND ($2::text IS NULL OR action      = $2)
  AND ($3::text IS NULL OR entity      = $3)
  AND ($4::timestamptz IS NULL OR created_at >= $4)
  AND ($5::timestamptz IS NULL OR created_at <= $5)
ORDER BY created_at DESC
LIMIT $6 OFFSET $7;
