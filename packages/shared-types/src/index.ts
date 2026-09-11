/**
 * Shared API contracts between the web app and the Go backend.
 * Keep in sync with packages/openapi/openapi.yaml (source of truth).
 */

export type Provider = "bkash" | "nagad";
export type AccountType = "personal" | "agent" | "merchant";
export type SessionStatus =
  | "pending"
  | "succeeded"
  | "expired"
  | "canceled";
export type VerificationSource =
  | "auto"
  | "claim"
  | "manual_override"
  | "manual_paste";

export interface Session {
  id: string;
  merchant_id: string;
  amount: number; // integer taka (BDT)
  currency: "BDT";
  status: SessionStatus;
  provider?: Provider;
  expires_at: string; // ISO 8601
  created_at: string;
}

export interface WebhookEvent {
  id: string;
  type: "payment.succeeded" | "payment.canceled" | "payment.expired";
  data: {
    session_id: string;
    payment_id: string;
    amount: number;
    currency: "BDT";
    provider: Provider;
    account_type: AccountType;
    sender_number: string; // canonical 01XXXXXXXXX
    trx_id: string;
    verified_at: string;
    verification_source: VerificationSource;
    customer?: { name?: string; email?: string; mobile?: string };
    metadata?: Record<string, unknown>;
  };
}

export interface ApiError {
  error: { code: string; message: string; param?: string };
}
