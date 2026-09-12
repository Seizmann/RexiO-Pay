CREATE TABLE IF NOT EXISTS public.idempotency_keys (
    key             text        PRIMARY KEY,
    merchant_id     text        NOT NULL REFERENCES public.merchants(id) ON DELETE CASCADE,
    request_hash    text        NOT NULL,
    response_body   bytea       NOT NULL,
    status_code     int         NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    expires_at      timestamptz NOT NULL DEFAULT now() + interval '24 hours'
);

CREATE INDEX IF NOT EXISTS idempotency_keys_expires_at_idx ON public.idempotency_keys(expires_at);
