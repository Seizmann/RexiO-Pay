/**
 * Shared API contracts between the web app and the Go backend.
 *
 * SOURCE OF TRUTH: the Go handlers under apps/backend/internal/api/.
 * packages/openapi/openapi.yaml is generated documentation, not the contract.
 * Where the two disagree, the handlers win (see WORKLOGS Session 4).
 */

export type Provider = "bkash" | "nagad";
export type AccountType = "personal" | "agent" | "merchant";
export type SessionStatus = "pending" | "succeeded" | "expired" | "canceled";
export type ProfileStatus = "active" | "disabled" | "pending";
export type DeviceStatus = "pending" | "active" | "offline" | "disabled";
export type VerificationSource =
  | "auto"
  | "claim"
  | "manual_override"
  | "manual_paste";
export type TeamRole = "owner" | "admin" | "staff";
export type PlanCode = "starter" | "pro" | "business";
export type AnnouncementSeverity = "info" | "success" | "warning" | "error";

/** Error envelope returned by every backend route: {"error":{code,message,param?}}. */
export interface ApiErrorBody {
  error: {
    code:
      | "plan_limit_exceeded"
      | "domain_not_whitelisted"
      | "idempotency_conflict"
      | "device_disabled"
      | "unauthorized"
      | "forbidden"
      | "not_found"
      | "user_not_found"
      | "invalid_request"
      | "internal_error"
      | (string & {});
    message: string;
    param?: string;
  };
}

/** Merchant list envelope: limit/offset pagination with has_more, no total count. */
export interface Paginated<T> {
  data: T[];
  has_more: boolean;
}

/** Admin list envelope: echoes limit/offset, no total count. */
export interface AdminPage<T> {
  data: T[];
  limit: number;
  offset: number;
}

// ── Public checkout ────────────────────────────────────────────────────────

/**
 * GET /v1/checkout/{id} response. NOTE: the backend does not yet return the
 * receiving profile's provider/account_type/mfs_number, nor return_url /
 * cancel_url / plan_status. The optional fields below document the expected
 * shape once the backend adds them (Session 4 gap flag #2).
 */
export interface CheckoutSessionPublic {
  id: string;
  amount: number; // integer taka
  currency: string; // "BDT"
  status: SessionStatus;
  needs_review: boolean;
  review_reason?: string | null; // "needs_trxid" | "late_match" | ...
  expires_at: string;
  created_at: string;
  sender_number_claim?: string | null;
  trx_id_claim?: string | null;
  confirmed_at?: string | null;
  merchant_name: string;
  merchant_logo_url?: string;
  merchant_brand_color?: string;
  merchant_support_email?: string;
  merchant_support_phone?: string;
  payment_profile_id: string;
  // Not yet returned by the backend — planned fields:
  provider?: Provider;
  account_type?: AccountType;
  mfs_number?: string;
  return_url?: string;
  cancel_url?: string;
  plan_status?: string;
}

/** POST /v1/checkout/{id}/claim body. */
export interface CheckoutClaimRequest {
  sender_number: string; // canonical 01XXXXXXXXX enforced server-side
  trx_id?: string; // ≤ 100 chars, trimmed
  confirmed?: boolean; // sets confirmed_at
}

// ── Merchant API ───────────────────────────────────────────────────────────

export interface Session {
  id: string;
  amount: number;
  currency: string; // "BDT"
  status: SessionStatus;
  source: "api" | "payment_link" | "invoice";
  payment_profile_id: string;
  customer: Record<string, unknown>;
  metadata: Record<string, unknown>;
  needs_review: boolean;
  review_reason?: string | null;
  expires_at: string;
  return_url?: string | null;
  cancel_url?: string | null;
  payment_link_id?: string | null;
  checkout_url: string;
  created_at: string;
  updated_at: string;
}

export interface Payment {
  id: string;
  session_id: string;
  sms_id?: string | null;
  provider: Provider;
  account_type: AccountType;
  sender_number: string;
  trx_id: string;
  amount: number;
  balance_after?: number | null;
  verified_at: string;
  verification_source: VerificationSource;
  is_refund_flagged: boolean;
  refund_note?: string | null;
  created_at: string;
}

export interface PaymentProfile {
  id: string;
  merchant_id: string;
  provider: Provider;
  account_type: AccountType;
  mfs_number: string; // canonical 01XXXXXXXXX
  display_name: string;
  device_id?: string | null;
  sim_slot?: number | null; // 0..3
  tracked_balance?: number | null;
  balance_verification: boolean;
  balance_last_synced_at?: string | null;
  status: ProfileStatus;
  is_otp_verified: boolean;
  created_at: string;
}

export interface CreateProfileRequest {
  provider: string;
  account_type: string;
  mfs_number: string;
  display_name?: string;
  balance_verification?: boolean;
}

/** PATCH /v1/payment-profiles/{id}. device_id + tracked balance sync have no endpoint (gap flag #8). */
export interface UpdateProfileRequest {
  display_name?: string;
  sim_slot?: number; // 0..3
  status?: ProfileStatus;
}

export interface Device {
  id: string;
  name: string;
  model: string;
  android_version: string;
  app_version: string;
  status: DeviceStatus;
  failed_sig_count: number;
  last_heartbeat_at?: string | null;
  last_sms_synced_at?: string | null;
  pairing_token?: string; // returned once on create/repair
  pairing_expires_at?: string | null;
  created_at: string;
}

/** QR payload the dashboard must generate for the Android app (Session 3 contract). */
export interface PairingQrPayload {
  server_url: string;
  merchant_id: string;
  pairing_token: string;
}

export interface ApiKeyRow {
  id: string;
  name: string;
  key_prefix: string;
  scopes: string; // "full" | "restricted"
  last_used_at?: string | null;
  created_at: string;
  revoked_at?: string | null;
  key?: string; // raw key, returned once on create only
}

export interface TeamMember {
  merchant_id: string;
  user_id: string; // also the {id} in PATCH/DELETE paths
  role: TeamRole;
  invited_by?: string | null;
  created_at: string;
}

export interface PaymentLink {
  id: string;
  payment_profile_id: string;
  title: string;
  amount?: number | null; // null = customer-entered
  reusable: boolean;
  active: boolean;
  expires_at?: string | null;
  url: string;
  created_at: string;
}

export interface WebhookEndpoint {
  id: string;
  merchant_id: string;
  url: string;
  events: string[];
  status: "active" | "disabled";
  created_at: string;
}

export interface WebhookDelivery {
  id: string;
  endpoint_id: string;
  merchant_id: string;
  event_type: string;
  payload: Record<string, unknown>;
  attempt_count: number;
  next_retry_at?: string | null;
  status: "pending" | "delivered" | "failed";
  last_response_code?: number | null;
  created_at: string;
  delivered_at?: string | null;
}

/** Actual payment.succeeded payload (engine.go:424). Note: no currency/customer/metadata. */
export interface WebhookEvent {
  id: string;
  type: "payment.succeeded" | "payment.canceled" | "payment.expired";
  data: {
    session_id: string;
    payment_id: string;
    amount: number;
    provider: Provider;
    account_type: AccountType;
    sender_number: string;
    trx_id: string;
    verified_at: string;
    verification_source: VerificationSource;
  };
}

export interface MerchantSettings {
  claim_window_minutes: number;
  session_ttl_minutes: number; // stored; session creation reads env only (gap flag #11)
  amount_tolerance: number;
  auto_accept_late_match: boolean;
  balance_verification: boolean;
}

/** Full merchant row; returned by GET /v1/branding (the de-facto get-me). */
export interface Merchant {
  id: string;
  owner_user_id: string;
  name: string;
  slug: string;
  logo_url: string;
  favicon_url: string;
  brand_color: string; // #rrggbb
  support_email: string;
  support_phone: string;
  plan_id: PlanCode;
  plan_status: string;
  plan_renews_at?: string | null;
  session_count_current_period: number;
  settings: MerchantSettings;
  created_at: string;
}

/** PATCH /v1/branding body. name + brand_color + valid support_phone required every call. */
export interface UpdateBrandingRequest {
  name: string;
  logo_url?: string;
  favicon_url?: string;
  brand_color: string;
  support_email?: string;
  support_phone: string;
}

export interface DomainWhitelistRow {
  id: string;
  merchant_id: string;
  domain: string;
  added_by?: string | null;
  created_at: string;
}

/** POST /v1/assets/presign response. */
export interface PresignResponse {
  key: string;
  upload_url: string;
  public_url: string;
  expires_in: number;
}

export interface OtpVerification {
  id: string;
  merchant_id: string;
  mfs_number: string;
  provider: Provider;
  account_type: AccountType;
  status: "pending" | "verified" | "expired";
  created_at: string;
  expires_at: string;
  verification_number: string;
  amount: number;
}

export interface SmsMessage {
  id: string;
  merchant_id: string;
  device_id?: string | null;
  payment_profile_id?: string | null;
  provider: string;
  account_type: string;
  raw_text: string;
  parsed: Record<string, unknown> | null;
  parse_status: "parsed" | "failed";
  match_status: "matched" | "unmatched" | "duplicate" | "suspect" | "held";
  matched_session_id?: string | null;
  source: "app" | "manual_paste";
  sim_slot?: number | null;
  received_at: string;
  created_at: string;
}

// ── Admin API ──────────────────────────────────────────────────────────────

/**
 * Admin merchant list row: flat merchant fields joined with device_count /
 * total_payments (list) or total_sessions (detail), per admin.go.
 */
export interface AdminMerchantRow {
  id: string;
  name: string;
  plan_id: PlanCode;
  plan_status: string;
  device_count: number;
  total_payments: number;
  total_sessions?: number;
  session_count_current_period?: number;
}

export interface SmsStatRow {
  provider: string;
  parse_status: "parsed" | "failed";
  day: string;
  cnt: number;
}

export interface AuditLogRow {
  id: string;
  merchant_id?: string | null;
  actor_user_id?: string | null;
  actor_device_id?: string | null;
  action: string;
  entity: string;
  entity_id: string;
  details: Record<string, unknown> | null;
  created_at: string;
}

export interface Announcement {
  id: string;
  title: string;
  body: string;
  severity: AnnouncementSeverity;
  expires_at?: string | null;
  created_at: string;
}

/** Plan limits table from REQUIREMENT §18 (0 = unlimited). Mirrors backend seed data. */
export interface PlanInfo {
  code: PlanCode;
  price_bdt_monthly: number;
  limits: {
    payment_profiles: number;
    devices: number;
    sessions_per_month: number;
    webhook_endpoints: number;
    team_members: number;
  };
}
