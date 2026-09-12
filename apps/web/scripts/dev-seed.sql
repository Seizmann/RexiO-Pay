-- RexiO Pay — development seed for the web dashboard (manual, run in Supabase SQL editor).
-- Session 4 gap flag #1: the backend has no signup endpoint and no first-API-key
-- bootstrap, so a merchant row + first key must be provisioned by hand for now.
--
-- HOW TO USE
-- 1. Sign up a user through the web app's Supabase Auth (apps/web /login or /signup).
-- 2. Find the user's UUID:  select id from auth.users order by created_at desc limit 5;
-- 3. Generate an API key locally, e.g.:
--      KEY="rk_live_$(openssl rand -hex 32)"; echo "$KEY"
--    then compute its hash (must match the backend middleware):
--      printf '%s' "$KEY" | sha256sum
-- 4. Replace the placeholders below and run.
--
-- The merchant will be able to use the dashboard with that key after the
-- "Connect API key" step in the web app.

BEGIN;

INSERT INTO public.users (id, email, full_name, is_platform_admin, locale)
VALUES (
  '<USER_UUID_FROM_AUTH_USERS>',
  '<USER_EMAIL>',
  'Test Merchant Owner',
  true,               -- set true to also unlock /rexio-admin for this user's merchant
  'en'
)
ON CONFLICT (id) DO UPDATE SET is_platform_admin = EXCLUDED.is_platform_admin;

INSERT INTO public.merchants (id, owner_user_id, name, slug)
VALUES (
  'mer_test0001',
  '<USER_UUID_FROM_AUTH_USERS>',
  'Test Merchant',
  'test-merchant'
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO public.api_keys (id, merchant_id, name, key_prefix, key_hash, scopes)
VALUES (
  'apk_test0001',
  'mer_test0001',
  'dev key',
  'rk_live_test',
  '<SHA256_HEX_OF_THE_RAW_KEY>',
  'full'
)
ON CONFLICT (id) DO NOTHING;

COMMIT;
