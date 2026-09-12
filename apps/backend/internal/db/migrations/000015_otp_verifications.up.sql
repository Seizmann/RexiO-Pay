CREATE TABLE IF NOT EXISTS public.mfs_otp_verifications (
    id              text        PRIMARY KEY,
    merchant_id     text        NOT NULL REFERENCES public.merchants(id) ON DELETE CASCADE,
    mfs_number      text        NOT NULL,
    provider        text        NOT NULL,
    account_type    text        NOT NULL,
    status          text        NOT NULL DEFAULT 'pending',
    created_at      timestamptz NOT NULL DEFAULT now(),
    expires_at      timestamptz NOT NULL DEFAULT now() + interval '30 minutes'
);

CREATE INDEX IF NOT EXISTS mfs_otp_verifications_merchant_id_idx ON public.mfs_otp_verifications(merchant_id, status);
CREATE INDEX IF NOT EXISTS mfs_otp_verifications_number_idx      ON public.mfs_otp_verifications(mfs_number, provider, status);
