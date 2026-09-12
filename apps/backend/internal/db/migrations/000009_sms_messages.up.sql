CREATE TABLE IF NOT EXISTS public.sms_messages (
    id                  text        PRIMARY KEY,
    merchant_id         text        NOT NULL REFERENCES public.merchants(id) ON DELETE CASCADE,
    device_id           text        REFERENCES public.devices(id),
    payment_profile_id  text        REFERENCES public.payment_profiles(id),
    provider            text        NOT NULL DEFAULT '',
    account_type        text        NOT NULL DEFAULT '',
    raw_text            text        NOT NULL,
    parsed              jsonb,
    parse_status        text        NOT NULL DEFAULT 'pending',
    match_status        text        NOT NULL DEFAULT 'unmatched',
    matched_session_id  text        REFERENCES public.checkout_sessions(id),
    source              text        NOT NULL DEFAULT 'app',
    sim_slot            int,
    received_at         timestamptz NOT NULL DEFAULT now(),
    created_at          timestamptz NOT NULL DEFAULT now()
);

-- Dedupe index: only one parsed row per (provider, trx_id)
CREATE UNIQUE INDEX IF NOT EXISTS sms_messages_trx_id_unique_idx
    ON public.sms_messages(provider, (parsed->>'trx_id'))
    WHERE parse_status = 'parsed' AND parsed->>'trx_id' IS NOT NULL;

CREATE INDEX IF NOT EXISTS sms_messages_merchant_id_idx      ON public.sms_messages(merchant_id);
CREATE INDEX IF NOT EXISTS sms_messages_match_status_idx     ON public.sms_messages(merchant_id, match_status);
CREATE INDEX IF NOT EXISTS sms_messages_payment_profile_idx  ON public.sms_messages(payment_profile_id);
