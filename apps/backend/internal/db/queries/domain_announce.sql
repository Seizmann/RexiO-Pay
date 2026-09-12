-- name: AddDomain :one
INSERT INTO public.domain_whitelist (id, merchant_id, domain, added_by)
VALUES ($1, $2, $3, $4)
ON CONFLICT (merchant_id, domain) DO NOTHING
RETURNING *;

-- name: RemoveDomain :exec
DELETE FROM public.domain_whitelist WHERE id = $1 AND merchant_id = $2;

-- name: ListDomains :many
SELECT * FROM public.domain_whitelist WHERE merchant_id = $1 ORDER BY created_at ASC;

-- name: CheckDomainAllowed :one
SELECT id FROM public.domain_whitelist
WHERE merchant_id = $1 AND domain = $2
LIMIT 1;

-- name: ListActiveAnnouncements :many
SELECT * FROM public.announcements
WHERE expires_at IS NULL OR expires_at > now()
ORDER BY created_at DESC;

-- name: ListAnnouncementsAdmin :many
SELECT * FROM public.announcements
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateAnnouncement :one
INSERT INTO public.announcements (id, title, body, severity, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateAnnouncement :one
UPDATE public.announcements
SET title = $2,
    body = $3,
    severity = $4,
    expires_at = $5
WHERE id = $1
RETURNING *;

-- name: DeleteAnnouncement :exec
DELETE FROM public.announcements WHERE id = $1;
