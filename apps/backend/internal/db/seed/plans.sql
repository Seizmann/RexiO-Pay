-- Seed the 3 subscription plans (REQUIREMENT.md §18)
-- Run once after migrate-up: psql $DATABASE_URL -f internal/db/seed/plans.sql

INSERT INTO public.plans (code, price_bdt_monthly, limits) VALUES
(
    'starter',
    0,
    '{
        "payment_profiles": 2,
        "devices": 1,
        "sessions_per_month": 100,
        "webhook_endpoints": 1,
        "team_members": 1
    }'::jsonb
),
(
    'pro',
    1000,
    '{
        "payment_profiles": 10,
        "devices": 3,
        "sessions_per_month": 5000,
        "webhook_endpoints": 5,
        "team_members": 5
    }'::jsonb
),
(
    'business',
    3000,
    '{
        "payment_profiles": 0,
        "devices": 0,
        "sessions_per_month": 0,
        "webhook_endpoints": 0,
        "team_members": 0
    }'::jsonb
)
ON CONFLICT (code) DO UPDATE SET
    price_bdt_monthly = EXCLUDED.price_bdt_monthly,
    limits            = EXCLUDED.limits;
