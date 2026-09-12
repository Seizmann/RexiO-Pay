-- name: InviteTeamMember :exec
INSERT INTO public.team_members (merchant_id, user_id, role, invited_by)
VALUES ($1, $2, $3, $4)
ON CONFLICT (merchant_id, user_id) DO NOTHING;

-- name: GetTeamMember :one
SELECT tm.*, u.email, u.full_name
FROM public.team_members tm
JOIN public.users u ON u.id = tm.user_id
WHERE tm.merchant_id = $1 AND tm.user_id = $2
LIMIT 1;

-- name: ListTeamMembers :many
SELECT tm.*, u.email, u.full_name
FROM public.team_members tm
JOIN public.users u ON u.id = tm.user_id
WHERE tm.merchant_id = $1
ORDER BY tm.created_at ASC;

-- name: UpdateTeamMemberRole :exec
UPDATE public.team_members
SET role = $3
WHERE merchant_id = $1 AND user_id = $2;

-- name: RemoveTeamMember :exec
DELETE FROM public.team_members
WHERE merchant_id = $1 AND user_id = $2;

-- name: CountTeamMembers :one
SELECT COUNT(*) FROM public.team_members WHERE merchant_id = $1;

-- name: GetUserByEmail :one
SELECT * FROM public.users WHERE email = $1 LIMIT 1;
