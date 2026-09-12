CREATE TABLE IF NOT EXISTS public.audit_logs (
    id              text        PRIMARY KEY,
    merchant_id     text        REFERENCES public.merchants(id) ON DELETE SET NULL,
    actor_user_id   text        REFERENCES public.users(id) ON DELETE SET NULL,
    actor_device_id text        REFERENCES public.devices(id) ON DELETE SET NULL,
    action          text        NOT NULL,
    entity          text        NOT NULL DEFAULT '',
    entity_id       text        NOT NULL DEFAULT '',
    details         jsonb       NOT NULL DEFAULT '{}'::jsonb,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS audit_logs_merchant_id_idx ON public.audit_logs(merchant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_logs_entity_idx      ON public.audit_logs(entity, entity_id);

CREATE TABLE IF NOT EXISTS public.domain_whitelist (
    id          text        PRIMARY KEY,
    merchant_id text        NOT NULL REFERENCES public.merchants(id) ON DELETE CASCADE,
    domain      text        NOT NULL,
    added_by    text        REFERENCES public.users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (merchant_id, domain)
);

CREATE INDEX IF NOT EXISTS domain_whitelist_merchant_id_idx ON public.domain_whitelist(merchant_id);

CREATE TABLE IF NOT EXISTS public.announcements (
    id          text        PRIMARY KEY,
    title       text        NOT NULL,
    body        text        NOT NULL,
    severity    text        NOT NULL DEFAULT 'info',
    expires_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);
