CREATE TABLE IF NOT EXISTS public.team_members (
    merchant_id text        NOT NULL REFERENCES public.merchants(id) ON DELETE CASCADE,
    user_id     text        NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    role        text        NOT NULL DEFAULT 'staff',
    invited_by  text        REFERENCES public.users(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (merchant_id, user_id)
);

CREATE INDEX IF NOT EXISTS team_members_merchant_id_idx ON public.team_members(merchant_id);
CREATE INDEX IF NOT EXISTS team_members_user_id_idx     ON public.team_members(user_id);
