CREATE TABLE IF NOT EXISTS public.webhook_endpoints (
    id          text        PRIMARY KEY,
    merchant_id text        NOT NULL REFERENCES public.merchants(id) ON DELETE CASCADE,
    url         text        NOT NULL,
    secret      text        NOT NULL,
    events      jsonb       NOT NULL DEFAULT '["payment.succeeded","payment.canceled","payment.expired"]'::jsonb,
    status      text        NOT NULL DEFAULT 'active',
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS webhook_endpoints_merchant_id_idx ON public.webhook_endpoints(merchant_id);

CREATE TABLE IF NOT EXISTS public.webhook_deliveries (
    id                  text        PRIMARY KEY,
    endpoint_id         text        NOT NULL REFERENCES public.webhook_endpoints(id) ON DELETE CASCADE,
    merchant_id         text        NOT NULL REFERENCES public.merchants(id) ON DELETE CASCADE,
    event_type          text        NOT NULL,
    payload             jsonb       NOT NULL DEFAULT '{}'::jsonb,
    attempt_count       int         NOT NULL DEFAULT 0,
    next_retry_at       timestamptz NOT NULL DEFAULT now(),
    status              text        NOT NULL DEFAULT 'pending',
    last_response_code  int,
    created_at          timestamptz NOT NULL DEFAULT now(),
    delivered_at        timestamptz
);

CREATE INDEX IF NOT EXISTS webhook_deliveries_endpoint_id_idx ON public.webhook_deliveries(endpoint_id);
CREATE INDEX IF NOT EXISTS webhook_deliveries_merchant_id_idx ON public.webhook_deliveries(merchant_id);
CREATE INDEX IF NOT EXISTS webhook_deliveries_pending_idx
    ON public.webhook_deliveries(next_retry_at)
    WHERE status = 'pending';
