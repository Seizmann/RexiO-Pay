"use client";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { formatTaka } from "@/lib/money";
import { canonicalizePhone, isValidPhone } from "@/lib/phone";
import { useCopy } from "@/lib/use-copy";
import { formatTemplate } from "@/lib/i18n/format";
import { useI18n } from "@/lib/i18n";
import { useState } from "react";
import type { CheckoutSessionPublic } from "@rexio-pay/shared-types";

/**
 * Pay instructions + claim controls.
 *
 * The public checkout endpoint does not yet return the receiving profile's
 * provider/account_type/mfs_number (Session 4 gap flag #2), so when those
 * fields are absent the page falls back to a plain send-money explanation
 * with the amount copy button and the merchant's support contacts.
 */
export function PayPanel({
  session,
  onSubmitClaim,
  claiming,
  claimError,
}: {
  session: CheckoutSessionPublic;
  onSubmitClaim: (args: { senderNumber: string; trxId: string }) => void;
  claiming: boolean;
  claimError: string | null;
}) {
  const { dict, lang } = useI18n();
  const { copied, copy } = useCopy();

  const [sender, setSender] = useState(session.sender_number_claim ?? "");
  const [trxId, setTrxId] = useState("");
  const senderCanonical = canonicalizePhone(sender);
  const senderValid = isValidPhone(sender);
  const needsTrxid = session.needs_review && session.review_reason === "needs_trxid";

  const providerName =
    session.provider === "nagad"
      ? "Nagad"
      : session.provider === "bkash"
        ? "bKash"
        : "bKash/Nagad";
  const accountTypeLabel =
    session.account_type === "agent"
      ? dict.checkout.providerAgent
      : session.account_type === "merchant"
        ? dict.checkout.providerMerchant
        : dict.checkout.providerPersonal;
  const amount = formatTaka(session.amount, lang);

  const instructionsUnlocked = senderValid;

  return (
    <section className="flex flex-col gap-6">
      {/* Sender number gate */}
      <Input
        id="sender_number"
        type="tel"
        inputMode="numeric"
        label={dict.checkout.senderNumber}
        hint={senderValid ? undefined : dict.checkout.senderNumberHint}
        error={
          sender.length > 0 && !senderValid ? dict.checkout.senderNumberInvalid : undefined
        }
        value={sender}
        onChange={(e) => setSender(e.target.value)}
        autoComplete="tel-national"
        placeholder="01XXXXXXXXX"
      />

      {/* Pay-to block when the backend returns profile details */}
      {session.mfs_number ? (
        <div className="rounded-lg border border-hairline bg-canvas-soft p-4">
          <p className="text-caption text-ink-mute">{dict.checkout.payTo}</p>
          <div className="mt-1 flex items-center justify-between gap-3">
            <div>
              <p className="tnum text-heading-md text-ink">{session.mfs_number}</p>
              <p className="text-caption text-ink-secondary">
                {providerName} · {accountTypeLabel}
              </p>
            </div>
            <Button variant="secondary" size="sm" onClick={() => copy(session.mfs_number!, "num")}>
              {copied === "num" ? dict.common.copied : dict.checkout.copyNumber}
            </Button>
          </div>
        </div>
      ) : null}

      {/* Instructions */}
      <div
        className={`rounded-lg border border-hairline p-5 transition-opacity ${
          instructionsUnlocked ? "" : "pointer-events-none opacity-40"
        }`}
        aria-hidden={!instructionsUnlocked}
      >
        <h2 className="text-heading-md text-ink">{dict.checkout.instructionsTitle}</h2>
        {session.mfs_number ? (
          <ol className="mt-3 flex list-decimal flex-col gap-2 pl-5 text-body-md text-ink-secondary">
            <li>{formatTemplate(dict.checkout.step1, { provider: providerName })}</li>
            <li>
              {formatTemplate(dict.checkout.step2, { number: session.mfs_number })}{" "}
              <button
                type="button"
                className="text-primary hover:underline"
                onClick={() => copy(session.mfs_number!, "num")}
              >
                {copied === "num" ? dict.common.copied : dict.checkout.copyNumber}
              </button>
            </li>
            <li>
              {formatTemplate(dict.checkout.step3, { amount })}{" "}
              <button
                type="button"
                className="tnum text-primary hover:underline"
                onClick={() => copy(String(session.amount), "amt")}
              >
                {copied === "amt" ? dict.common.copied : dict.checkout.copyAmount}
              </button>
            </li>
            <li>{dict.checkout.step4}</li>
          </ol>
        ) : (
          <p className="mt-3 text-body-md text-ink-secondary">
            {dict.checkout.fallbackBody}
          </p>
        )}
        <div className="mt-4 flex items-center justify-between gap-3 rounded-md bg-canvas-soft px-4 py-3">
          <span className="tnum text-heading-md text-ink">{amount}</span>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => copy(String(session.amount), "amt")}
          >
            {copied === "amt" ? dict.common.copied : dict.checkout.copyAmount}
          </Button>
        </div>
      </div>

      {/* Claim controls */}
      <div className="flex flex-col gap-4">
        {needsTrxid ? (
          <p className="rounded-md border border-hairline bg-canvas-cream px-4 py-3 text-caption text-ink">
            {dict.checkout.trxidRequired}
          </p>
        ) : null}
        <Input
          id="trx_id"
          label={dict.checkout.trxidLabel}
          hint={needsTrxid ? undefined : formatTemplate(dict.checkout.trxidHint, { provider: providerName })}
          error={claimError ?? undefined}
          value={trxId}
          onChange={(e) => setTrxId(e.target.value)}
          maxLength={100}
          className={needsTrxid ? "border-ruby" : undefined}
        />
        <div>
          <p className="text-heading-sm text-ink">{dict.checkout.paidQuestion}</p>
          <p className="mt-1 text-caption text-ink-mute">{dict.checkout.paidInfo}</p>
        </div>
        <Button
          disabled={!senderValid || claiming}
          onClick={() => onSubmitClaim({ senderNumber: senderCanonical, trxId: trxId.trim() })}
        >
          {claiming ? dict.common.loading : dict.checkout.paidButton}
        </Button>
      </div>
    </section>
  );
}
