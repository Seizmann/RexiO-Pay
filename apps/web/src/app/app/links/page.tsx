"use client";

import { useCallback, useEffect, useState } from "react";
import { api, ApiError } from "@/lib/api";
import type { PaymentLink, PaymentProfile } from "@rexio-pay/shared-types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { StatusBadge } from "@/components/ui/badge";
import { useCopy } from "@/lib/use-copy";
import { useI18n } from "@/lib/i18n";

/**
 * Payment Links (REQUIREMENT §14.6): create + list + copy.
 * No enable/disable endpoint exists (gap flag #7) and opening a link URL
 * needs a backend session-on-open endpoint; both are noted in the UI.
 */
export default function LinksPage() {
  const { dict } = useI18n();
  const m = dict.modules;
  const { copied, copy } = useCopy();
  const [rows, setRows] = useState<PaymentLink[]>([]);
  const [profiles, setProfiles] = useState<PaymentProfile[]>([]);
  const [form, setForm] = useState({
    payment_profile_id: "",
    title: "",
    amount: "",
    reusable: true,
  });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    try {
      const [linksRes, profilesRes] = await Promise.all([
        api.get<{ data: PaymentLink[] }>("/payment-links"),
        api.get<{ data: PaymentProfile[] }>("/payment-profiles"),
      ]);
      setRows(linksRes.data);
      setProfiles(profilesRes.data.filter((p) => p.status === "active"));
      if (!form.payment_profile_id && profilesRes.data.length > 0) {
        setForm((f) => ({ ...f, payment_profile_id: profilesRes.data[0].id }));
      }
    } catch {
      // shell handles disconnect
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await api.post("/payment-links", {
        payment_profile_id: form.payment_profile_id,
        title: form.title,
        amount: form.amount === "" ? null : Number(form.amount),
        reusable: form.reusable,
      });
      setForm((f) => ({ ...f, title: "", amount: "" }));
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <h1 className="text-display-lg text-ink">{m.linksTitle}</h1>

      {profiles.length === 0 && !loading ? (
        <p className="mt-4 rounded-md bg-canvas-cream px-4 py-3 text-caption text-ink">
          {m.profilesEmpty}
        </p>
      ) : (
        <form onSubmit={create} className="mt-6 flex flex-wrap items-end gap-4 rounded-lg border border-hairline bg-canvas p-6">
          <label className="flex flex-col gap-1.5">
            <span className="text-caption text-ink-secondary">{m.profilesTitle}</span>
            <select
              value={form.payment_profile_id}
              onChange={(e) => setForm({ ...form, payment_profile_id: e.target.value })}
              className="h-10 rounded-sm border border-hairline-input bg-canvas px-3 text-body-md"
            >
              {profiles.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.display_name || p.mfs_number} ({p.provider})
                </option>
              ))}
            </select>
          </label>
          <Input id="pl-title" label={m.linksTitleField} value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} required maxLength={200} />
          <Input
            id="pl-amount"
            type="number"
            min={1}
            label={m.linksAmount}
            value={form.amount}
            onChange={(e) => setForm({ ...form, amount: e.target.value })}
            className="w-44"
          />
          <label className="flex items-center gap-2 pb-2 text-body-md text-ink-secondary">
            <input
              type="checkbox"
              checked={form.reusable}
              onChange={(e) => setForm({ ...form, reusable: e.target.checked })}
              className="h-4 w-4 accent-[var(--color-primary)]"
            />
            {m.linksReusable}
          </label>
          <Button type="submit" disabled={busy || !form.title.trim() || !form.payment_profile_id}>
            {busy ? dict.common.saving : m.linksCreate}
          </Button>
          {error ? <p className="text-caption text-ruby">{error}</p> : null}
        </form>
      )}

      {loading ? (
        <p className="mt-8 text-body-md text-ink-mute">{dict.common.loading}</p>
      ) : rows.length === 0 ? (
        <p className="mt-8 text-body-md text-ink-mute">{m.linksEmpty}</p>
      ) : (
        <div className="mt-6 overflow-x-auto rounded-lg border border-hairline bg-canvas">
          <table className="w-full min-w-2xl text-body-tabular">
            <thead>
              <tr className="border-b border-hairline bg-canvas-soft text-left text-ink-mute">
                <th className="px-4 py-3 font-normal">{m.linksTitleField}</th>
                <th className="px-4 py-3 font-normal">{m.txAmount}</th>
                <th className="px-4 py-3 font-normal">{m.linksReusable}</th>
                <th className="px-4 py-3 font-normal">{m.txStatus}</th>
                <th className="px-4 py-3 font-normal">{m.profilesActions}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((l) => (
                <tr key={l.id} className="border-b border-hairline last:border-0">
                  <td className="px-4 py-3 text-ink">{l.title}</td>
                  <td className="tnum px-4 py-3">
                    {l.amount === null ? (
                      <span className="text-ink-mute">{m.linksAmount.split(" ")[0]}</span>
                    ) : (
                      `৳${l.amount}`
                    )}
                  </td>
                  <td className="px-4 py-3">{l.reusable ? dict.common.yes : dict.common.no}</td>
                  <td className="px-4 py-3">
                    <StatusBadge
                      tone={l.active ? "succeeded" : "muted"}
                      label={l.active ? dict.status.active : dict.status.disabled}
                    />
                  </td>
                  <td className="px-4 py-3">
                    <Button variant="ghost" size="sm" onClick={() => copy(l.url, l.id)}>
                      {copied === l.id ? dict.common.copied : m.linksCopy}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <p className="mt-4 text-caption text-ink-mute">
        {m.linksToggleNote} {m.linksOpenNote}
      </p>
    </div>
  );
}
