/**
 * Row shapes for Supabase RLS reads. These mirror the DB schema
 * (REQUIREMENT §8) as read by the browser client; the API-key contract
 * types in @rexio-pay/shared-types cover the Go API side.
 */
export interface CheckoutSession {
  id: string;
  status: "pending" | "succeeded" | "expired" | "canceled";
  amount: number;
  currency: string;
  customer: Record<string, unknown> | null;
  metadata: Record<string, unknown> | null;
  source: string;
  needs_review: boolean;
  review_reason: string | null;
  return_url: string | null;
  cancel_url: string | null;
  created_at: string;
  expires_at: string;
}

export interface Device {
  id: string;
  name: string;
  model: string;
  status: string;
  last_heartbeat_at: string | null;
  last_sms_synced_at: string | null;
}

export interface SmsRow {
  id: string;
  provider: string | null;
  match_status: string | null;
  parse_status: string | null;
  raw_text: string;
  received_at: string;
  matched_session_id: string | null;
}

export interface WebhookEndpointRow {
  id: string;
  url: string;
  events: string[] | null;
  status: string;
  created_at: string;
}

export interface WebhookDeliveryRow {
  id: string;
  endpoint_id: string;
  event_type: string;
  attempt_count: number;
  status: string;
  last_response_code: number | null;
  created_at: string;
}

export interface PaymentView {
  id: string;
  provider: string;
  account_type: string;
  trx_id: string;
  sender_number: string;
  verification_source: string;
}
