CREATE TABLE IF NOT EXISTS public.payment_profiles (
    id                      text        PRIMARY KEY,
    merchant_id             text        NOT NULL REFERENCES public.merchants(id) ON DELETE CASCADE,
    provider                text        NOT NULL,
    account_type            text        NOT NULL,
    mfs_number              text        NOT NULL,
    display_name            text        NOT NULL DEFAULT '',
    device_id               text,
    sim_slot                int,
    tracked_balance         bigint,
    balance_verification    bool        NOT NULL DEFAULT true,
    balance_last_synced_at  timestamptz,
    status                  text        NOT NULL DEFAULT 'active',
    is_otp_verified         bool        NOT NULL DEFAULT false,
    created_at              timestamptz NOT NULL DEFAULT now(),
    UNIQUE (merchant_id, mfs_number)
);

CREATE INDEX IF NOT EXISTS payment_profiles_merchant_id_idx ON public.payment_profiles(merchant_id);
