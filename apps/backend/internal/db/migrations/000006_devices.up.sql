CREATE TABLE IF NOT EXISTS public.devices (
    id                  text        PRIMARY KEY,
    merchant_id         text        NOT NULL REFERENCES public.merchants(id) ON DELETE CASCADE,
    name                text        NOT NULL DEFAULT '',
    model               text        NOT NULL DEFAULT '',
    android_version     text        NOT NULL DEFAULT '',
    app_version         text        NOT NULL DEFAULT '',
    -- AES-256-GCM encrypted device_secret. Server decrypts in-memory to verify HMAC.
    -- Never stored as plaintext or as a raw SHA-256 hash (that would allow sig forgery on DB leak).
    secret_ciphertext   bytea,
    secret_nonce        bytea,
    status              text        NOT NULL DEFAULT 'pending',
    failed_sig_count    int         NOT NULL DEFAULT 0,
    last_heartbeat_at   timestamptz,
    last_sms_synced_at  timestamptz,
    pairing_token       text        UNIQUE,
    pairing_expires_at  timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS devices_merchant_id_idx    ON public.devices(merchant_id);
CREATE INDEX IF NOT EXISTS devices_pairing_token_idx  ON public.devices(pairing_token) WHERE pairing_token IS NOT NULL;
