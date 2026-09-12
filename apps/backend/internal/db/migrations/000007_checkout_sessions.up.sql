CREATE TABLE IF NOT EXISTS public.checkout_sessions (
    id                      text        PRIMARY KEY,
    merchant_id             text        NOT NULL REFERENCES public.merchants(id) ON DELETE CASCADE,
    payment_profile_id      text        NOT NULL REFERENCES public.payment_profiles(id),
    source                  text        NOT NULL DEFAULT 'api',
    amount                  bigint      NOT NULL,
    currency                text        NOT NULL DEFAULT 'BDT',
    customer                jsonb       NOT NULL DEFAULT '{}'::jsonb,
    metadata                jsonb       NOT NULL DEFAULT '{}'::jsonb,
    sender_number_claim     text,
    trx_id_claim            text,
    claim_submitted_at      timestamptz,
    confirmed_at            timestamptz,
    status                  text        NOT NULL DEFAULT 'pending',
    needs_review            bool        NOT NULL DEFAULT false,
    review_reason           text,
    expires_at              timestamptz NOT NULL,
    return_url              text        NOT NULL DEFAULT '',
    cancel_url              text        NOT NULL DEFAULT '',
    payment_link_id         text,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS checkout_sessions_merchant_id_idx ON public.checkout_sessions(merchant_id);
CREATE INDEX IF NOT EXISTS checkout_sessions_match_idx
    ON public.checkout_sessions(payment_profile_id, status, amount, sender_number_claim);
CREATE INDEX IF NOT EXISTS checkout_sessions_status_idx ON public.checkout_sessions(status);
