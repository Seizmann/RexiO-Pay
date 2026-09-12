"use client";

import { useCallback, useEffect, useState } from "react";
import { api, ApiError, isPlanLimitError } from "@/lib/api";
import type { CreateProfileRequest, PaymentProfile } from "@rexio-pay/shared-types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { StatusBadge } from "@/components/ui/badge";
import { useI18n } from "@/lib/i18n";
import { canonicalizePhone, isValidPhone } from "@/lib/phone";

/**
 * Payment Profiles (REQUIREMENT §14.3): CRUD via the merchant API.
 * device_id binding and tracked-balance sync are shown but stubbed;
 * the backend PATCH accepts only display_name/sim_slot/status (gap flag #8).
 */
export default function ProfilesPage() {
  const { dict } = useI18n();
  const m = dict.modules;
  const [rows, setRows] = useState<PaymentProfile[]>([]);
  const [adding, setAdding] = useState(false);
  const [limitHit, setLimitHit] = useState(false);
  const [loading, setLoading] = useState(true);
  const [actionError, setActionError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const list = await api.get<{ data: PaymentProfile[] }>("/payment-profiles");
      setRows(list.data);
      setLimitHit(false);
    } catch {
      // disconnect handled by shell
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function toggleStatus(p: PaymentProfile) {
    setActionError(null);
    try {
      await api.patch(`/payment-profiles/${p.id}`, {
        status: p.status === "active" ? "disabled" : "active",
      });
      await load();
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    }
  }

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-4">
        <h1 className="text-display-lg text-ink">{m.profilesTitle}</h1>
        <Button
          onClick={() => setAdding((v) => !v)}
          disabled={limitHit}
          title={limitHit ? dict.modules.planLimitReached : undefined}
        >
          {m.profilesAdd}
        </Button>
      </div>
      {limitHit ? (
        <p className="mt-3 rounded-md bg-canvas-cream px-4 py-3 text-caption text-ink">
          {dict.common.errorPlanLimit}
        </p>
      ) : null}
      {actionError ? <p className="mt-3 text-caption text-ruby">{actionError}</p> : null}

      {adding ? (
        <div className="mt-6">
          <ProfileForm
            onDone={() => {
              setAdding(false);
              void load();
            }}
            onLimit={() => {
              setAdding(false);
              setLimitHit(true);
            }}
          />
        </div>
      ) : null}

      {loading ? (
        <p className="mt-8 text-body-md text-ink-mute">{dict.common.loading}</p>
      ) : rows.length === 0 ? (
        <p className="mt-8 text-body-md text-ink-mute">{m.profilesEmpty}</p>
      ) : (
        <div className="mt-6 overflow-x-auto rounded-lg border border-hairline bg-canvas">
          <table className="w-full min-w-3xl text-body-tabular">
            <thead>
              <tr className="border-b border-hairline bg-canvas-soft text-left text-ink-mute">
                <th className="px-4 py-3 font-normal">{m.profilesNumber}</th>
                <th className="px-4 py-3 font-normal">{m.profilesName}</th>
                <th className="px-4 py-3 font-normal">{m.txProvider}</th>
                <th className="px-4 py-3 font-normal">{m.profilesSim}</th>
                <th className="px-4 py-3 font-normal">{dict.status.verified}</th>
                <th className="px-4 py-3 font-normal">{m.txStatus}</th>
                <th className="px-4 py-3 font-normal">{m.profilesActions}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((p) => (
                <tr key={p.id} className="border-b border-hairline last:border-0">
                  <td className="tnum px-4 py-3 text-ink">{p.mfs_number}</td>
                  <td className="px-4 py-3">{p.display_name || "—"}</td>
                  <td className="px-4 py-3 capitalize">
                    {p.provider} · {p.account_type}
                  </td>
                  <td className="tnum px-4 py-3">{p.sim_slot ?? "—"}</td>
                  <td className="px-4 py-3">
                    {p.is_otp_verified ? (
                      <StatusBadge tone="succeeded" label={dict.status.verified} />
                    ) : (
                      <span className="text-ink-mute">—</span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge
                      tone={p.status === "active" ? "succeeded" : "muted"}
                      label={p.status === "active" ? dict.status.active : dict.status.disabled}
                    />
                  </td>
                  <td className="px-4 py-3">
                    <Button variant="ghost" size="sm" onClick={() => toggleStatus(p)}>
                      {p.status === "active" ? m.profilesPause : m.profilesActivate}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <p className="mt-4 text-caption text-ink-mute">{m.profileBindNote}</p>
    </div>
  );
}

function ProfileForm({
  onDone,
  onLimit,
}: {
  onDone: () => void;
  onLimit: () => void;
}) {
  const { dict } = useI18n();
  const m = dict.modules;
  const [form, setForm] = useState<CreateProfileRequest>({
    provider: "bkash",
    account_type: "personal",
    mfs_number: "",
    display_name: "",
  });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await api.post("/payment-profiles", {
        ...form,
        mfs_number: canonicalizePhone(form.mfs_number),
      });
      onDone();
    } catch (err) {
      if (isPlanLimitError(err)) {
        onLimit();
        return;
      }
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="flex flex-wrap items-end gap-4 rounded-lg border border-hairline bg-canvas p-6">
      <label className="flex flex-col gap-1.5">
        <span className="text-caption text-ink-secondary">{m.txProvider}</span>
        <select
          value={form.provider}
          onChange={(e) => setForm({ ...form, provider: e.target.value })}
          className="h-10 rounded-sm border border-hairline-input bg-canvas px-3 text-body-md"
        >
          <option value="bkash">bKash</option>
          <option value="nagad">Nagad</option>
        </select>
      </label>
      <label className="flex flex-col gap-1.5">
        <span className="text-caption text-ink-secondary">{dict.onboard.otpAccountType}</span>
        <select
          value={form.account_type}
          onChange={(e) => setForm({ ...form, account_type: e.target.value })}
          className="h-10 rounded-sm border border-hairline-input bg-canvas px-3 text-body-md"
        >
          <option value="personal">{dict.checkout.providerPersonal}</option>
          <option value="agent">{dict.checkout.providerAgent}</option>
          <option value="merchant">{dict.checkout.providerMerchant}</option>
        </select>
      </label>
      <Input
        id="pf-number"
        type="tel"
        label={m.profilesNumber}
        value={form.mfs_number}
        onChange={(e) => setForm({ ...form, mfs_number: e.target.value })}
        placeholder="01XXXXXXXXX"
        required
      />
      <Input
        id="pf-name"
        label={m.profilesName}
        value={form.display_name}
        onChange={(e) => setForm({ ...form, display_name: e.target.value })}
      />
      <Button type="submit" disabled={busy || !isValidPhone(form.mfs_number)}>
        {busy ? dict.common.saving : dict.common.save}
      </Button>
      {error ? <p className="text-caption text-ruby">{error}</p> : null}
    </form>
  );
}
