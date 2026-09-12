-- RLS: enable row-level security on all tenant tables.
-- Applied after migrations (M1). Backend connects with service role
-- (bypasses RLS); these policies protect direct Supabase client reads.

ALTER TABLE public.users                  ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.merchants              ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.team_members           ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.plans                  ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.payment_profiles       ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.devices                ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.checkout_sessions      ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.payments               ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.sms_messages           ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.webhook_endpoints      ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.webhook_deliveries     ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.api_keys               ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.payment_links          ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.audit_logs             ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.domain_whitelist       ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.announcements          ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.idempotency_keys       ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.mfs_otp_verifications  ENABLE ROW LEVEL SECURITY;
