CREATE TABLE IF NOT EXISTS public.api_keys (
    id          text        PRIMARY KEY,
    merchant_id text        NOT NULL REFERENCES public.merchants(id) ON DELETE CASCADE,
    name        text        NOT NULL DEFAULT '',
    key_prefix  text        NOT NULL,
    key_hash    text        NOT NULL UNIQUE,
    scopes      text        NOT NULL DEFAULT 'full',
    last_used_at timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    revoked_at  timestamptz
);

CREATE INDEX IF NOT EXISTS api_keys_merchant_id_idx ON public.api_keys(merchant_id);
CREATE INDEX IF NOT EXISTS api_keys_key_hash_idx    ON public.api_keys(key_hash);
