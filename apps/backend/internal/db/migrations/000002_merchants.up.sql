CREATE TABLE IF NOT EXISTS public.merchants (
    id                          text        PRIMARY KEY,
    owner_user_id               text        NOT NULL REFERENCES public.users(id),
    name                        text        NOT NULL DEFAULT '',
    slug                        text        NOT NULL DEFAULT '',
    logo_url                    text        NOT NULL DEFAULT '',
    favicon_url                 text        NOT NULL DEFAULT '',
    brand_color                 text        NOT NULL DEFAULT '#533afd',
    support_email               text        NOT NULL DEFAULT '',
    support_phone               text        NOT NULL DEFAULT '',
    plan_id                     text        NOT NULL DEFAULT 'starter',
    plan_status                 text        NOT NULL DEFAULT 'active',
    plan_renews_at              timestamptz,
    session_count_current_period bigint     NOT NULL DEFAULT 0,
    settings                    jsonb       NOT NULL DEFAULT '{
        "claim_window_minutes": 10,
        "session_ttl_minutes": 30,
        "amount_tolerance": 0,
        "auto_accept_late_match": false,
        "balance_verification": true
    }'::jsonb,
    created_at                  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS merchants_owner_user_id_idx ON public.merchants(owner_user_id);
