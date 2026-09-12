CREATE TABLE IF NOT EXISTS public.payment_links (
    id                  text        PRIMARY KEY,
    merchant_id         text        NOT NULL REFERENCES public.merchants(id) ON DELETE CASCADE,
    payment_profile_id  text        NOT NULL REFERENCES public.payment_profiles(id),
    title               text        NOT NULL DEFAULT '',
    amount              bigint,
    reusable            bool        NOT NULL DEFAULT false,
    active              bool        NOT NULL DEFAULT true,
    expires_at          timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS payment_links_merchant_id_idx ON public.payment_links(merchant_id);
