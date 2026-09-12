"use client";

import { useCallback, useEffect, useRef, useState, use } from "react";
import Image from "next/image";
import type { CheckoutClaimRequest, CheckoutSessionPublic } from "@rexio-pay/shared-types";
import { Button } from "@/components/ui/button";
import { useI18n } from "@/lib/i18n";
import { formatTaka } from "@/lib/money";
import { Countdown } from "./countdown";
import { PayPanel } from "./pay-panel";

/**
 * Public checkout (REQUIREMENT §15). Mobile-first, no login.
 * Polls GET /v1/checkout/{id} every 3 seconds; the claim goes through
 * POST /v1/checkout/{id}/claim. Merchant-disabled state (plan_status) is
 * rendered when the backend adds the field (gap flag #2); until then the
 * maintenance branch exists but is not reachable.
 */

const POLL_MS = 3000;

type Phase = "loading" | "ready" | "error";

export default function CheckoutPage({
  params,
}: {
  params: Promise<{ session_id: string }>;
}) {
  const { dict } = useI18n();
  const { session_id } = use(params);

  const [session, setSession] = useState<CheckoutSessionPublic | null>(null);
  const [phase, setPhase] = useState<Phase>("loading");
  const [notFound, setNotFound] = useState(false);
  const [claiming, setClaiming] = useState(false);
  const [claimError, setClaimError] = useState<string | null>(null);
  const pollTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const submittedRef = useRef(false);

  const stopPolling = useCallback(() => {
    if (pollTimer.current) {
      clearTimeout(pollTimer.current);
      pollTimer.current = null;
    }
  }, []);

  const applySession = useCallback(
    (next: CheckoutSessionPublic) => {
      setSession(next);
      setPhase("ready");
      const terminal =
        next.status === "succeeded" ||
        next.status === "expired" ||
        next.status === "canceled";
      if (terminal) stopPolling();

      // Success: prefer the merchant's return_url when the backend provides it.
      if (next.status === "succeeded" && !submittedRef.current) {
        submittedRef.current = true;
        if (next.return_url) {
          const url = new URL(next.return_url);
          url.searchParams.set("session_id", next.id);
          url.searchParams.set("status", "succeeded");
          window.location.href = url.toString();
        }
      }
    },
    [stopPolling],
  );

  const fetchSession = useCallback(async () => {
    try {
      const res = await fetch(`/api/backend/checkout/${session_id}`, {
        cache: "no-store",
      });
      if (res.status === 404) {
        setNotFound(true);
        setPhase("error");
        stopPolling();
        return;
      }
      if (!res.ok) throw new Error(String(res.status));
      const body = (await res.json()) as CheckoutSessionPublic;
      applySession(body);
    } catch {
      // Transient network errors keep polling; a hard failure only replaces
      // the panel if we never loaded a session at all.
      setPhase((current) => (current === "loading" ? "error" : current));
    }
  }, [session_id, applySession, stopPolling]);

  useEffect(() => {
    void fetchSession();
    pollTimer.current = setInterval(() => void fetchSession(), POLL_MS) as unknown as ReturnType<typeof setTimeout>;
    return stopPolling;
  }, [fetchSession, stopPolling]);

  async function submitClaim(args: { senderNumber: string; trxId: string }) {
    setClaiming(true);
    setClaimError(null);
    try {
      const body: CheckoutClaimRequest = {
        sender_number: args.senderNumber,
        trx_id: args.trxId || undefined,
        confirmed: true,
      };
      const res = await fetch(`/api/backend/checkout/${session_id}/claim`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const err = (await res.json().catch(() => null)) as
          | { error?: { message?: string } }
          | null;
        throw new Error(err?.error?.message ?? dict.common.errorGeneric);
      }
      const next = (await res.json()) as CheckoutSessionPublic;
      applySession(next);
    } catch (err) {
      setClaimError(err instanceof Error ? err.message : dict.common.errorGeneric);
    } finally {
      setClaiming(false);
    }
  }

  // ── State matrix ──────────────────────────────────────────────────────

  if (phase === "loading") {
    return (
      <CheckoutShell>
        <p className="text-body-md text-ink-mute">{dict.common.loading}</p>
      </CheckoutShell>
    );
  }

  if (notFound) {
    return (
      <CheckoutShell>
        <StateBlock
          title={dict.checkout.notFound}
          body={dict.checkout.notFoundHelp}
        />
      </CheckoutShell>
    );
  }

  if (phase === "error" || !session) {
    return (
      <CheckoutShell>
        <StateBlock
          title={dict.common.errorTitle}
          body={dict.checkout.sessionFailed}
          action={
            <Button variant="secondary" onClick={() => window.location.reload()}>
              {dict.common.retry}
            </Button>
          }
        />
      </CheckoutShell>
    );
  }

  if (session.plan_status === "disabled") {
    return (
      <CheckoutShell session={session}>
        <StateBlock
          title={dict.checkout.maintenanceTitle}
          body={dict.checkout.maintenanceBody}
        />
      </CheckoutShell>
    );
  }

  if (session.status === "canceled") {
    // REQUIREMENT §15: canceled sessions redirect to cancel_url when present.
    if (session.cancel_url && !submittedRef.current) {
      submittedRef.current = true;
      window.location.href = session.cancel_url;
    }
    return (
      <CheckoutShell session={session}>
        <StateBlock title={dict.checkout.canceled} body="" />
      </CheckoutShell>
    );
  }

  if (session.status === "expired") {
    return (
      <CheckoutShell session={session}>
        <StateBlock
          title={dict.checkout.expired}
          body={dict.checkout.expiredHelp}
          support={session}
        />
      </CheckoutShell>
    );
  }

  if (session.status === "succeeded") {
    return (
      <CheckoutShell session={session}>
        <StateBlock
          title={dict.checkout.successTitle}
          body={
            session.return_url
              ? formatTemplateDict(dict.checkout.redirecting, session.merchant_name)
              : dict.checkout.successBody
          }
          success
          support={session}
        />
      </CheckoutShell>
    );
  }

  // pending (incl. needs_review)
  const inReview = session.needs_review && session.review_reason !== "needs_trxid";
  return (
    <CheckoutShell session={session}>
      {inReview ? (
        <div className="rounded-lg border border-hairline bg-canvas-soft p-6">
          <h1 className="text-heading-lg text-ink">{dict.checkout.reviewTitle}</h1>
          <p className="mt-2 text-body-md text-ink-secondary">
            {dict.checkout.reviewBody}
          </p>
        </div>
      ) : (
        <>
          <header className="flex flex-col gap-1">
            <span className="text-micro-cap uppercase text-ink-mute">
              {dict.checkout.amount}
            </span>
            <span className="tnum text-display-lg text-ink">
              {formatTaka(session.amount)}
            </span>
            <Countdown
              expiresAt={session.expires_at}
              onExpired={() => void fetchSession()}
            />
          </header>
          <PayPanel
            session={session}
            onSubmitClaim={submitClaim}
            claiming={claiming}
            claimError={claimError}
          />
        </>
      )}
    </CheckoutShell>
  );
}

function formatTemplateDict(template: string, merchant: string): string {
  return template.replace("{merchant}", merchant);
}

// ── Layout helpers ──────────────────────────────────────────────────────

function CheckoutShell({
  session,
  children,
}: {
  session?: CheckoutSessionPublic;
  children: React.ReactNode;
}) {
  const { dict } = useI18n();
  return (
    <main className="mx-auto flex min-h-dvh w-full max-w-md flex-col px-5 py-8">
      <header className="flex items-center gap-3">
        {session?.merchant_logo_url ? (
          <Image
            src={session.merchant_logo_url}
            alt={session.merchant_name}
            width={40}
            height={40}
            className="h-10 w-10 rounded-md object-contain"
            unoptimized
          />
        ) : null}
        <div>
          <p className="text-heading-sm text-ink">
            {session?.merchant_name ?? "RexiO Pay"}
          </p>
        </div>
      </header>
      <div className="mt-8 flex flex-1 flex-col gap-8">{children}</div>
      <footer className="mt-12 border-t border-hairline pt-4">
        <p className="text-center text-micro text-ink-mute">
          {dict.checkout.poweredBy}
        </p>
      </footer>
    </main>
  );
}

function StateBlock({
  title,
  body,
  action,
  support,
  success,
}: {
  title: string;
  body: string;
  action?: React.ReactNode;
  support?: CheckoutSessionPublic;
  success?: boolean;
}) {
  const { dict } = useI18n();
  return (
    <section className="flex flex-col gap-4 rounded-lg border border-hairline bg-canvas p-6 shadow-level-1">
      <h1 className={`text-heading-lg ${success ? "text-primary-deep" : "text-ink"}`}>
        {title}
      </h1>
      {body ? <p className="text-body-md text-ink-secondary">{body}</p> : null}
      {support && (support.merchant_support_phone || support.merchant_support_email) ? (
        <p className="text-caption text-ink-mute">
          {formatTemplateDict(dict.checkout.support, support.merchant_name)}{" "}
          {support.merchant_support_phone ? (
            <span className="tnum">{support.merchant_support_phone}</span>
          ) : null}
          {support.merchant_support_phone && support.merchant_support_email ? " · " : null}
          {support.merchant_support_email}
        </p>
      ) : null}
      {action}
    </section>
  );
}
