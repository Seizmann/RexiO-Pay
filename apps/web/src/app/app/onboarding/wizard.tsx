"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";
import type {
  DomainWhitelistRow,
  Merchant,
  OtpVerification,
} from "@rexio-pay/shared-types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { ConnectKeyForm } from "@/components/connect-key-form";
import { useCopy } from "@/lib/use-copy";
import { useI18n } from "@/lib/i18n";
import { canonicalizePhone, isValidPhone } from "@/lib/phone";

/**
 * Onboarding wizard (REQUIREMENT §19):
 * connect key → branding basics → MFS OTP verify → device pairing →
 * domain whitelist (optional) → plan choice. Plan payment is manual (§18).
 */
export function OnboardingWizard() {
  const { dict } = useI18n();
  const router = useRouter();
  const o = dict.onboard;
  const [step, setStep] = useState(0);
  const [merchant, setMerchant] = useState<Merchant | null>(null);

  useEffect(() => {
    api.get<Merchant>("/branding").then(setMerchant).catch(() => undefined);
  }, []);

  const steps = [
    o.stepKey,
    o.stepBranding,
    o.stepOtp,
    o.stepDevice,
    o.stepDomains,
    o.stepPlan,
  ];

  const advance = useCallback(
    (next?: Merchant) => {
      if (next) setMerchant(next);
      setStep((s) => Math.min(s + 1, steps.length - 1));
    },
    [steps.length],
  );

  return (
    <div className="mx-auto w-full max-w-2xl">
      <h1 className="text-display-lg text-ink">{o.title}</h1>
      <ol className="mt-6 flex flex-wrap gap-2">
        {steps.map((label, i) => (
          <li
            key={label}
            className={`rounded-pill px-3 py-1 text-micro-cap uppercase ${
              i === step
                ? "bg-primary text-on-primary"
                : i < step
                  ? "bg-primary-bg-subdued-hover text-primary-deep"
                  : "bg-canvas text-ink-mute"
            }`}
          >
            {i + 1}. {label}
          </li>
        ))}
      </ol>
      <Card className="mt-6">
        {step === 0 && <StepKey onDone={() => advance()} />}
        {step === 1 && <StepBranding merchant={merchant} onDone={advance} />}
        {step === 2 && <StepOtp onDone={() => advance()} />}
        {step === 3 && <StepDevice onDone={() => advance()} />}
        {step === 4 && <StepDomains onDone={() => advance()} />}
        {step === 5 && <StepPlan merchant={merchant} onDone={() => router.push("/app")} />}
      </Card>
    </div>
  );
}

// ── Step 0: connect API key ─────────────────────────────────────────────

function StepKey({ onDone }: { onDone: () => void }) {
  const { dict } = useI18n();
  return (
    <div>
      <h2 className="text-heading-lg text-ink">{dict.connect.title}</h2>
      <p className="mt-2 text-body-md text-ink-secondary">{dict.connect.body}</p>
      <div className="mt-6">
        <ConnectKeyForm onConnected={onDone} />
      </div>
    </div>
  );
}

// ── Step 1: branding basics ─────────────────────────────────────────────

function StepBranding({
  merchant,
  onDone,
}: {
  merchant: Merchant | null;
  onDone: (m: Merchant) => void;
}) {
  const { dict } = useI18n();
  const o = dict.onboard;
  const [name, setName] = useState(merchant?.name ?? "");
  const [email, setEmail] = useState(merchant?.support_email ?? "");
  const [phone, setPhone] = useState(merchant?.support_phone ?? "");
  const [color, setColor] = useState(merchant?.brand_color ?? "#533afd");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const updated = await api.patch<Merchant>("/branding", {
        name,
        brand_color: color,
        support_email: email,
        support_phone: phone,
        logo_url: merchant?.logo_url ?? "",
        favicon_url: merchant?.favicon_url ?? "",
      });
      onDone(updated);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-5">
      <h2 className="text-heading-lg text-ink">{o.stepBranding}</h2>
      <Input id="ob-name" label={o.brandName} value={name} onChange={(e) => setName(e.target.value)} required />
      <Input id="ob-email" type="email" label={o.brandEmail} value={email} onChange={(e) => setEmail(e.target.value)} />
      <Input
        id="ob-phone"
        type="tel"
        label={o.brandPhone}
        hint={o.brandPhoneHint}
        value={phone}
        onChange={(e) => setPhone(e.target.value)}
        placeholder="01XXXXXXXXX"
        required
      />
      <Input
        id="ob-color"
        type="color"
        label={o.brandColor}
        value={color}
        onChange={(e) => setColor(e.target.value)}
        className="h-12 w-24 p-1"
      />
      {error ? <p className="text-caption text-ruby">{error}</p> : null}
      <Button type="submit" disabled={busy || !name || !isValidPhone(phone)}>
        {busy ? dict.common.saving : dict.common.save}
      </Button>
    </form>
  );
}

// ── Step 2: MFS OTP verification (status polling, per §19 step 3) ───────

function StepOtp({ onDone }: { onDone: () => void }) {
  const { dict } = useI18n();
  const o = dict.onboard;
  const { copied, copy } = useCopy();
  const [provider, setProvider] = useState<"bkash" | "nagad">("bkash");
  const [accountType, setAccountType] = useState<"personal" | "agent" | "merchant">("personal");
  const [mfsNumber, setMfsNumber] = useState("");
  const [otp, setOtp] = useState<OtpVerification | null>(null);
  const [error, setError] = useState<string | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(
    () => () => {
      if (pollRef.current) clearInterval(pollRef.current);
    },
    [],
  );

  async function start(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      const result = await api.post<OtpVerification>("/onboarding/otp/init", {
        provider,
        account_type: accountType,
        mfs_number: canonicalizePhone(mfsNumber),
      });
      setOtp(result);
      pollRef.current = setInterval(async () => {
        try {
          const status = await api.get<OtpVerification>(
            `/onboarding/otp/${result.id}`,
          );
          setOtp(status);
          if (status.status === "verified" || status.status === "expired") {
            if (pollRef.current) clearInterval(pollRef.current);
          }
        } catch {
          // keep polling
        }
      }, 3000);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    }
  }

  if (otp) {
    if (otp.status === "verified") {
      return (
        <div>
          <h2 className="text-heading-lg text-ink">{o.stepOtp}</h2>
          <p className="mt-3 rounded-md bg-primary-bg-subdued-hover px-4 py-3 text-body-md text-primary-deep">
            {o.otpVerified}
          </p>
          <Button className="mt-6" onClick={onDone}>
            {dict.common.next}
          </Button>
        </div>
      );
    }
    if (otp.status === "expired") {
      return (
        <div>
          <h2 className="text-heading-lg text-ink">{o.stepOtp}</h2>
          <p className="mt-3 text-body-md text-ink-secondary">{o.otpExpired}</p>
          <Button variant="secondary" className="mt-6" onClick={() => setOtp(null)}>
            {dict.common.retry}
          </Button>
        </div>
      );
    }
    return (
      <div>
        <h2 className="text-heading-lg text-ink">{o.stepOtp}</h2>
        <p className="mt-3 text-body-md text-ink-secondary">{o.otpWaiting}</p>
        <div className="mt-4 rounded-lg border border-hairline bg-canvas-soft p-4">
          <p className="text-caption text-ink-mute">{o.otpSendFrom}</p>
          <div className="mt-1 flex items-center justify-between gap-3">
            <p className="tnum text-heading-md text-ink">{otp.verification_number}</p>
            <Button variant="secondary" size="sm" onClick={() => copy(otp.verification_number)}>
              {copied ? dict.common.copied : dict.common.copy}
            </Button>
          </div>
          <p className="tnum mt-2 text-caption text-ink-secondary">
            {dict.checkout.amount}: ৳{otp.amount}
          </p>
        </div>
        <div className="mt-6 flex items-center gap-2 text-body-md text-ink-mute">
          <span className="h-2 w-2 animate-pulse rounded-full bg-primary" />
          {dict.common.loading}
        </div>
      </div>
    );
  }

  return (
    <form onSubmit={start} className="flex flex-col gap-5">
      <h2 className="text-heading-lg text-ink">{o.stepOtp}</h2>
      <p className="text-body-md text-ink-secondary">{o.otpIntro}</p>
      <div className="grid grid-cols-2 gap-4">
        <label className="flex flex-col gap-1.5">
          <span className="text-caption text-ink-secondary">{o.otpProvider}</span>
          <select
            value={provider}
            onChange={(e) => setProvider(e.target.value as "bkash" | "nagad")}
            className="h-10 rounded-sm border border-hairline-input bg-canvas px-3 text-body-md"
          >
            <option value="bkash">{o.providerBkash}</option>
            <option value="nagad">{o.providerNagad}</option>
          </select>
        </label>
        <label className="flex flex-col gap-1.5">
          <span className="text-caption text-ink-secondary">{o.otpAccountType}</span>
          <select
            value={accountType}
            onChange={(e) => setAccountType(e.target.value as typeof accountType)}
            className="h-10 rounded-sm border border-hairline-input bg-canvas px-3 text-body-md"
          >
            <option value="personal">{dict.checkout.providerPersonal}</option>
            <option value="agent">{dict.checkout.providerAgent}</option>
            <option value="merchant">{dict.checkout.providerMerchant}</option>
          </select>
        </label>
      </div>
      <Input
        id="otp-number"
        type="tel"
        label={o.otpNumber}
        value={mfsNumber}
        onChange={(e) => setMfsNumber(e.target.value)}
        placeholder="01XXXXXXXXX"
        required
      />
      {error ? <p className="text-caption text-ruby">{error}</p> : null}
      <Button type="submit" disabled={!isValidPhone(mfsNumber)}>
        {o.otpStart}
      </Button>
      <SkipButton />
    </form>
  );
}

// ── Step 3: device pairing QR ───────────────────────────────────────────

function StepDevice({ onDone }: { onDone: () => void }) {
  const { dict } = useI18n();
  const o = dict.onboard;
  const [name, setName] = useState("My phone");
  const [device, setDevice] = useState<{
    id: string;
    pairing_token?: string;
    pairing_expires_at?: string;
  } | null>(null);
  const [qrDataUrl, setQrDataUrl] = useState<string | null>(null);
  const [merchantId, setMerchantId] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .get<Merchant>("/branding")
      .then((m) => setMerchantId(m.id))
      .catch(() => undefined);
  }, []);

  async function createDevice() {
    setBusy(true);
    setError(null);
    try {
      const created = await api.post<typeof device & Record<string, unknown>>(
        "/devices",
        { name },
      );
      setDevice(created as NonNullable<typeof device>);
      const payload = {
        server_url: process.env.NEXT_PUBLIC_API_BASE_URL ?? "https://api.pay.rexio.pro",
        merchant_id: merchantId ?? "",
        pairing_token: (created as { pairing_token?: string }).pairing_token ?? "",
      };
      // QR payload contract from the Android app (Session 3): canonical JSON.
      const QRCode = (await import("qrcode")).default;
      setQrDataUrl(await QRCode.toDataURL(JSON.stringify(payload), { width: 256, margin: 1 }));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <h2 className="text-heading-lg text-ink">{o.stepDevice}</h2>
      <p className="mt-2 text-body-md text-ink-secondary">{o.deviceIntro}</p>
      {!device ? (
        <div className="mt-6 flex flex-col gap-5">
          <Input id="dev-name" label={o.deviceName} value={name} onChange={(e) => setName(e.target.value)} />
          {error ? <p className="text-caption text-ruby">{error}</p> : null}
          <Button onClick={createDevice} disabled={busy || !merchantId}>
            {busy ? dict.common.loading : o.deviceCreate}
          </Button>
        </div>
      ) : (
        <div className="mt-6 flex flex-col items-start gap-4">
          {qrDataUrl ? (
            <>
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={qrDataUrl} alt={o.deviceScanWith} width={256} height={256} className="rounded-lg border border-hairline" />
              <p className="text-caption text-ink-mute">{o.deviceScanWith}</p>
            </>
          ) : null}
          <Button variant="secondary" onClick={createDevice} disabled={busy}>
            {o.deviceRegenerate}
          </Button>
        </div>
      )}
      <Button className="mt-6" variant="ghost" onClick={onDone}>
        {dict.common.skip}
      </Button>
    </div>
  );
}

// ── Step 4: domain whitelist (optional) ─────────────────────────────────

function StepDomains({ onDone }: { onDone: () => void }) {
  const { dict } = useI18n();
  const o = dict.onboard;
  const [domain, setDomain] = useState("");
  const [rows, setRows] = useState<DomainWhitelistRow[]>([]);
  const [error, setError] = useState<string | null>(null);

  async function add(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      await api.post("/domain-whitelist", { domain: domain.trim() });
      setDomain("");
      const list = await api.get<{ data: DomainWhitelistRow[] }>("/domain-whitelist");
      setRows(list.data);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    }
  }

  return (
    <div>
      <h2 className="text-heading-lg text-ink">{o.stepDomains}</h2>
      <p className="mt-2 text-body-md text-ink-secondary">{o.domainsIntro}</p>
      <form onSubmit={add} className="mt-6 flex items-end gap-3">
        <Input
          id="dom"
          label={o.domainsPlaceholder}
          placeholder={o.domainsPlaceholder}
          value={domain}
          onChange={(e) => setDomain(e.target.value)}
        />
        <Button type="submit" disabled={!domain.trim()}>
          {o.domainsAdd}
        </Button>
      </form>
      {error ? <p className="mt-3 text-caption text-ruby">{error}</p> : null}
      {rows.length > 0 ? (
        <ul className="mt-4 flex flex-col gap-1 text-body-md text-ink">
          {rows.map((r) => (
            <li key={r.id} className="tnum">{r.domain}</li>
          ))}
        </ul>
      ) : null}
      <Button className="mt-6" onClick={onDone}>
        {dict.common.next}
      </Button>
    </div>
  );
}

// ── Step 5: plan choice (manual billing per §18) ────────────────────────

function StepPlan({
  merchant,
  onDone,
}: {
  merchant: Merchant | null;
  onDone: () => void;
}) {
  const { dict } = useI18n();
  const o = dict.onboard;
  return (
    <div>
      <h2 className="text-heading-lg text-ink">{o.stepPlan}</h2>
      <p className="mt-2 text-body-md text-ink-secondary">{o.planIntro}</p>
      {merchant ? (
        <p className="mt-4 text-caption text-ink-mute">
          {o.planCurrent}: <span className="capitalize">{merchant.plan_id}</span>
        </p>
      ) : null}
      <p className="mt-4 rounded-md bg-canvas-soft px-4 py-3 text-caption text-ink-secondary">
        {dict.settings.billingNote}
      </p>
      <Button className="mt-6" onClick={onDone}>
        {o.finish}
      </Button>
    </div>
  );
}

function SkipButton() {
  const { dict } = useI18n();
  return (
    <p className="text-caption text-ink-mute">
      ({dict.common.skip}: {dict.connect.howTo.split(".")[0]})
    </p>
  );
}
