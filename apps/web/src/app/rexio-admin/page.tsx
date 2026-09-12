"use client";

import { useCallback, useEffect, useState } from "react";
import { api, ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { StatusBadge } from "@/components/ui/badge";
import { useI18n } from "@/lib/i18n";

/**
 * Admin merchants list + detail (REQUIREMENT §17): plan override, disable,
 * and the §18 manual billing confirmation (admin marks the monthly bKash
 * payment as received by overriding plan + renew date).
 * The admin endpoints return flat merchant rows joined with counts.
 */
interface AdminMerchantFlat {
  id: string;
  name: string;
  plan_id: string;
  plan_status: string;
  device_count: number;
  total_payments: number;
  total_sessions?: number;
  session_count_current_period: number;
}

export default function AdminMerchantsPage() {
  const { dict } = useI18n();
  const a = dict.admin;
  const [rows, setRows] = useState<AdminMerchantFlat[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [detail, setDetail] = useState<AdminMerchantFlat | null>(null);
  const [newPlan, setNewPlan] = useState("pro");
  const [renewDate, setRenewDate] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    try {
      const list = await api.get<{ data: AdminMerchantFlat[] }>("/rexio-admin/merchants");
      setRows(list.data);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    } finally {
      setLoading(false);
    }
  }, [dict]);

  useEffect(() => {
    void load();
  }, [load]);

  const openDetail = useCallback(
    async (id: string) => {
      setSelected(id);
      setDetail(null);
      try {
        const d = await api.get<AdminMerchantFlat>(`/rexio-admin/merchants/${id}`);
        setDetail(d);
      } catch {
        setDetail(null);
      }
    },
    [],
  );

  async function overridePlan(id: string) {
    setError(null);
    try {
      const body: Record<string, unknown> = { plan_id: newPlan };
      if (renewDate) body.plan_renews_at = new Date(renewDate).toISOString();
      await api.post(`/rexio-admin/merchants/${id}/plan`, body);
      await load();
      if (selected === id) await openDetail(id);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    }
  }

  async function disable(id: string) {
    setError(null);
    try {
      await api.post(`/rexio-admin/merchants/${id}/disable`);
      await load();
      if (selected === id) await openDetail(id);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    }
  }

  return (
    <div>
      <h1 className="text-display-lg text-ink">{a.merchants}</h1>
      {error ? <p className="mt-3 text-caption text-ruby">{error}</p> : null}

      {loading ? (
        <p className="mt-8 text-body-md text-ink-mute">{dict.common.loading}</p>
      ) : (
        <div className="mt-6 overflow-x-auto rounded-lg border border-hairline bg-canvas">
          <table className="w-full min-w-3xl text-body-tabular">
            <thead>
              <tr className="border-b border-hairline bg-canvas-soft text-left text-ink-mute">
                <th className="px-4 py-3 font-normal">{a.merchantName}</th>
                <th className="px-4 py-3 font-normal">{a.merchantPlan}</th>
                <th className="px-4 py-3 font-normal">{a.merchantStatus}</th>
                <th className="px-4 py-3 font-normal">{a.merchantDevices}</th>
                <th className="px-4 py-3 font-normal">{a.merchantPayments}</th>
                <th className="px-4 py-3 font-normal"></th>
              </tr>
            </thead>
            <tbody>
              {rows.map((m) => (
                <tr key={m.id} className="border-b border-hairline last:border-0">
                  <td className="px-4 py-3 text-ink">{m.name || m.id}</td>
                  <td className="px-4 py-3 capitalize">{m.plan_id}</td>
                  <td className="px-4 py-3">
                    <StatusBadge
                      tone={m.plan_status === "active" ? "succeeded" : "muted"}
                      label={m.plan_status}
                    />
                  </td>
                  <td className="tnum px-4 py-3">{m.device_count}</td>
                  <td className="tnum px-4 py-3">{m.total_payments}</td>
                  <td className="px-4 py-3">
                    <Button variant="ghost" size="sm" onClick={() => openDetail(m.id)}>
                      {a.overridePlan}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {selected ? (
        <section className="mt-8 rounded-lg border border-hairline bg-canvas p-6">
          <h2 className="text-heading-md text-ink">
            {detail?.name ?? selected}
          </h2>
          {detail ? (
            <p className="tnum mt-1 text-caption text-ink-mute">
              {a.merchantSessions}: {detail.total_sessions ?? "—"} ·{" "}
              {dict.settings.sessionsUsed}: {detail.session_count_current_period}
            </p>
          ) : (
            <p className="mt-2 text-caption text-ink-mute">{dict.common.loading}</p>
          )}
          {detail?.plan_status === "disabled" ? (
            <p className="mt-3 rounded-md bg-canvas-cream px-4 py-3 text-caption text-ink">
              {a.disabledNote}
            </p>
          ) : null}

          <div className="mt-6 flex flex-wrap items-end gap-4">
            <label className="flex flex-col gap-1.5">
              <span className="text-caption text-ink-secondary">{a.merchantPlan}</span>
              <select
                value={newPlan}
                onChange={(e) => setNewPlan(e.target.value)}
                className="h-10 rounded-sm border border-hairline-input bg-canvas px-3 text-body-md"
              >
                <option value="starter">starter</option>
                <option value="pro">pro</option>
                <option value="business">business</option>
              </select>
            </label>
            <Input
              id="adm-renew"
              type="date"
              label={dict.settings.renews}
              value={renewDate}
              onChange={(e) => setRenewDate(e.target.value)}
              className="w-44"
            />
            <Button onClick={() => overridePlan(selected)} disabled={!selected}>
              {a.billingConfirm}
            </Button>
            <Button variant="danger" onClick={() => disable(selected)} disabled={!selected}>
              {a.disableMerchant}
            </Button>
          </div>
          <p className="mt-3 text-micro text-ink-mute">
            {dict.settings.billingNote}
          </p>
        </section>
      ) : null}
    </div>
  );
}
