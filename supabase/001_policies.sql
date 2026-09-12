-- RLS SELECT policies for all tenant tables.
-- No INSERT/UPDATE/DELETE policies — all writes go through the Go backend (service role).
-- Platform admin (is_platform_admin=true) can read all rows via the bypass clause.

-- Helper: is platform admin
-- (checking users table; auth.uid() returns the Supabase auth user id)

-- ─── users ────────────────────────────────────────────────────────────────────
CREATE POLICY users_read_own ON public.users
  FOR SELECT USING (
    id = auth.uid()
    OR EXISTS (SELECT 1 FROM public.users u WHERE u.id = auth.uid() AND u.is_platform_admin)
  );

-- ─── merchants ────────────────────────────────────────────────────────────────
CREATE POLICY merchants_read_own ON public.merchants
  FOR SELECT USING (
    owner_user_id = auth.uid()
    OR EXISTS (
      SELECT 1 FROM public.team_members tm
      WHERE tm.merchant_id = merchants.id AND tm.user_id = auth.uid()
    )
    OR EXISTS (SELECT 1 FROM public.users u WHERE u.id = auth.uid() AND u.is_platform_admin)
  );

-- ─── plans (public read) ──────────────────────────────────────────────────────
CREATE POLICY plans_read_all ON public.plans
  FOR SELECT USING (true);

-- ─── team_members ─────────────────────────────────────────────────────────────
CREATE POLICY team_members_read_own ON public.team_members
  FOR SELECT USING (
    merchant_id IN (
      SELECT id FROM public.merchants
      WHERE owner_user_id = auth.uid()
      UNION
      SELECT tm2.merchant_id FROM public.team_members tm2 WHERE tm2.user_id = auth.uid()
    )
    OR EXISTS (SELECT 1 FROM public.users u WHERE u.id = auth.uid() AND u.is_platform_admin)
  );

-- ─── payment_profiles ─────────────────────────────────────────────────────────
CREATE POLICY payment_profiles_read_own ON public.payment_profiles
  FOR SELECT USING (
    merchant_id IN (
      SELECT id FROM public.merchants WHERE owner_user_id = auth.uid()
      UNION
      SELECT tm.merchant_id FROM public.team_members tm WHERE tm.user_id = auth.uid()
    )
    OR EXISTS (SELECT 1 FROM public.users u WHERE u.id = auth.uid() AND u.is_platform_admin)
  );

-- ─── devices ──────────────────────────────────────────────────────────────────
CREATE POLICY devices_read_own ON public.devices
  FOR SELECT USING (
    merchant_id IN (
      SELECT id FROM public.merchants WHERE owner_user_id = auth.uid()
      UNION
      SELECT tm.merchant_id FROM public.team_members tm WHERE tm.user_id = auth.uid()
    )
    OR EXISTS (SELECT 1 FROM public.users u WHERE u.id = auth.uid() AND u.is_platform_admin)
  );

-- ─── checkout_sessions ────────────────────────────────────────────────────────
CREATE POLICY checkout_sessions_read_own ON public.checkout_sessions
  FOR SELECT USING (
    merchant_id IN (
      SELECT id FROM public.merchants WHERE owner_user_id = auth.uid()
      UNION
      SELECT tm.merchant_id FROM public.team_members tm WHERE tm.user_id = auth.uid()
    )
    OR EXISTS (SELECT 1 FROM public.users u WHERE u.id = auth.uid() AND u.is_platform_admin)
  );

-- ─── payments ─────────────────────────────────────────────────────────────────
CREATE POLICY payments_read_own ON public.payments
  FOR SELECT USING (
    merchant_id IN (
      SELECT id FROM public.merchants WHERE owner_user_id = auth.uid()
      UNION
      SELECT tm.merchant_id FROM public.team_members tm WHERE tm.user_id = auth.uid()
    )
    OR EXISTS (SELECT 1 FROM public.users u WHERE u.id = auth.uid() AND u.is_platform_admin)
  );

-- ─── sms_messages ─────────────────────────────────────────────────────────────
CREATE POLICY sms_messages_read_own ON public.sms_messages
  FOR SELECT USING (
    merchant_id IN (
      SELECT id FROM public.merchants WHERE owner_user_id = auth.uid()
      UNION
      SELECT tm.merchant_id FROM public.team_members tm WHERE tm.user_id = auth.uid()
    )
    OR EXISTS (SELECT 1 FROM public.users u WHERE u.id = auth.uid() AND u.is_platform_admin)
  );

-- ─── webhook_endpoints ────────────────────────────────────────────────────────
CREATE POLICY webhook_endpoints_read_own ON public.webhook_endpoints
  FOR SELECT USING (
    merchant_id IN (
      SELECT id FROM public.merchants WHERE owner_user_id = auth.uid()
      UNION
      SELECT tm.merchant_id FROM public.team_members tm WHERE tm.user_id = auth.uid()
    )
    OR EXISTS (SELECT 1 FROM public.users u WHERE u.id = auth.uid() AND u.is_platform_admin)
  );

-- ─── webhook_deliveries ───────────────────────────────────────────────────────
CREATE POLICY webhook_deliveries_read_own ON public.webhook_deliveries
  FOR SELECT USING (
    merchant_id IN (
      SELECT id FROM public.merchants WHERE owner_user_id = auth.uid()
      UNION
      SELECT tm.merchant_id FROM public.team_members tm WHERE tm.user_id = auth.uid()
    )
    OR EXISTS (SELECT 1 FROM public.users u WHERE u.id = auth.uid() AND u.is_platform_admin)
  );

-- ─── api_keys ─────────────────────────────────────────────────────────────────
-- key_hash never exposed to frontend (column exists but policy only allows owner read)
CREATE POLICY api_keys_read_own ON public.api_keys
  FOR SELECT USING (
    merchant_id IN (
      SELECT id FROM public.merchants WHERE owner_user_id = auth.uid()
    )
    OR EXISTS (SELECT 1 FROM public.users u WHERE u.id = auth.uid() AND u.is_platform_admin)
  );

-- ─── payment_links ────────────────────────────────────────────────────────────
CREATE POLICY payment_links_read_own ON public.payment_links
  FOR SELECT USING (
    merchant_id IN (
      SELECT id FROM public.merchants WHERE owner_user_id = auth.uid()
      UNION
      SELECT tm.merchant_id FROM public.team_members tm WHERE tm.user_id = auth.uid()
    )
    OR EXISTS (SELECT 1 FROM public.users u WHERE u.id = auth.uid() AND u.is_platform_admin)
  );

-- ─── audit_logs ───────────────────────────────────────────────────────────────
CREATE POLICY audit_logs_read_own ON public.audit_logs
  FOR SELECT USING (
    merchant_id IN (
      SELECT id FROM public.merchants WHERE owner_user_id = auth.uid()
      UNION
      SELECT tm.merchant_id FROM public.team_members tm WHERE tm.user_id = auth.uid()
    )
    OR EXISTS (SELECT 1 FROM public.users u WHERE u.id = auth.uid() AND u.is_platform_admin)
  );

-- ─── domain_whitelist ─────────────────────────────────────────────────────────
CREATE POLICY domain_whitelist_read_own ON public.domain_whitelist
  FOR SELECT USING (
    merchant_id IN (
      SELECT id FROM public.merchants WHERE owner_user_id = auth.uid()
      UNION
      SELECT tm.merchant_id FROM public.team_members tm WHERE tm.user_id = auth.uid()
    )
    OR EXISTS (SELECT 1 FROM public.users u WHERE u.id = auth.uid() AND u.is_platform_admin)
  );

-- ─── announcements (public read) ──────────────────────────────────────────────
CREATE POLICY announcements_read_all ON public.announcements
  FOR SELECT USING (true);

-- ─── idempotency_keys ─────────────────────────────────────────────────────────
CREATE POLICY idempotency_keys_read_own ON public.idempotency_keys
  FOR SELECT USING (
    merchant_id IN (
      SELECT id FROM public.merchants WHERE owner_user_id = auth.uid()
    )
    OR EXISTS (SELECT 1 FROM public.users u WHERE u.id = auth.uid() AND u.is_platform_admin)
  );

-- ─── mfs_otp_verifications ────────────────────────────────────────────────────
CREATE POLICY otp_verifications_read_own ON public.mfs_otp_verifications
  FOR SELECT USING (
    merchant_id IN (
      SELECT id FROM public.merchants WHERE owner_user_id = auth.uid()
    )
    OR EXISTS (SELECT 1 FROM public.users u WHERE u.id = auth.uid() AND u.is_platform_admin)
  );
