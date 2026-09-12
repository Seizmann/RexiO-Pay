CREATE TABLE IF NOT EXISTS public.plans (
    code                text    PRIMARY KEY,
    price_bdt_monthly   bigint  NOT NULL DEFAULT 0,
    limits              jsonb   NOT NULL DEFAULT '{}'::jsonb
);
