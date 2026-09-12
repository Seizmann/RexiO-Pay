-- Payments are IMMUTABLE after insert — no UPDATE query exists in sqlc.
-- Corrections are new rows + audit events.
CREATE TABLE IF NOT EXISTS public.payments (
    id                      text        PRIMARY KEY,
    merchant_id             text        NOT NULL REFERENCES public.merchants(id) ON DELETE CASCADE,
    session_id              text        NOT NULL REFERENCES public.checkout_sessions(id),
    sms_id                  text,
    provider                text        NOT NULL,
    account_type            text        NOT NULL,
    sender_number           text        NOT NULL,
    trx_id                  text        NOT NULL,
    amount                  bigint      NOT NULL,
    balance_after           bigint,
    verified_at             timestamptz NOT NULL DEFAULT now(),
    verification_source     text        NOT NULL DEFAULT 'auto',
    is_refund_flagged       bool        NOT NULL DEFAULT false,
    refund_note             text,
    created_at              timestamptz NOT NULL DEFAULT now(),
    UNIQUE (session_id, trx_id)
);

CREATE INDEX IF NOT EXISTS payments_merchant_id_idx   ON public.payments(merchant_id);
CREATE INDEX IF NOT EXISTS payments_session_id_idx    ON public.payments(session_id);
CREATE INDEX IF NOT EXISTS payments_created_at_idx    ON public.payments(merchant_id, created_at DESC);
