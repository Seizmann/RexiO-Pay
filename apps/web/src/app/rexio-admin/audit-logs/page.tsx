"use client";

import { useCallback, useEffect, useState } from "react";
import { api, buildQuery } from "@/lib/api";
import type { AuditLogRow } from "@rexio-pay/shared-types";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { useI18n } from "@/lib/i18n";

/** Audit log viewer with merchant/action/entity/date filters (REQUIREMENT §17). */
export default function AdminAuditLogsPage() {
  const { dict, lang } = useI18n();
  const a = dict.admin;
  const [rows, setRows] = useState<AuditLogRow[]>([]);
  const [merchantId, setMerchantId] = useState("");
  const [action, setAction] = useState("");
  const [from, setFrom] = useState("");
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const qs = buildQuery({
        merchant_id: merchantId || undefined,
        action: action || undefined,
        from: from ? new Date(from).toISOString() : undefined,
        limit: 100,
      });
      const res = await api.get<{ data: AuditLogRow[] }>(`/rexio-admin/audit-logs${qs}`);
      setRows(res.data);
    } catch {
      // show empty
    } finally {
      setLoading(false);
    }
  }, [merchantId, action, from]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div>
      <h1 className="text-display-lg text-ink">{a.auditLogs}</h1>

      <div className="mt-6 flex flex-wrap items-end gap-4">
        <Input
          id="al-merchant"
          label={a.auditMerchant}
          placeholder="mer_..."
          value={merchantId}
          onChange={(e) => setMerchantId(e.target.value)}
          className="w-48"
        />
        <Input
          id="al-action"
          label={a.auditAction}
          placeholder="payment_verified"
          value={action}
          onChange={(e) => setAction(e.target.value)}
          className="w-48"
        />
        <Input id="al-from" type="date" label={dict.modules.txDate} value={from} onChange={(e) => setFrom(e.target.value)} className="w-44" />
        <Button variant="secondary" onClick={() => void load()}>
          {dict.common.refresh}
        </Button>
      </div>

      {loading ? (
        <p className="mt-8 text-body-md text-ink-mute">{dict.common.loading}</p>
      ) : rows.length === 0 ? (
        <p className="mt-8 text-body-md text-ink-mute">{dict.common.empty}</p>
      ) : (
        <div className="mt-6 overflow-x-auto rounded-lg border border-hairline bg-canvas">
          <table className="w-full min-w-3xl text-body-tabular">
            <thead>
              <tr className="border-b border-hairline bg-canvas-soft text-left text-ink-mute">
                <th className="px-4 py-3 font-normal">{dict.modules.txDate}</th>
                <th className="px-4 py-3 font-normal">{a.auditMerchant}</th>
                <th className="px-4 py-3 font-normal">{a.auditAction}</th>
                <th className="px-4 py-3 font-normal">{a.auditEntity}</th>
                <th className="px-4 py-3 font-normal">{a.auditActor}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.id} className="border-b border-hairline last:border-0">
                  <td className="px-4 py-3 text-ink-secondary">
                    {new Date(r.created_at).toLocaleString(lang === "bn" ? "bn-BD" : "en-GB")}
                  </td>
                  <td className="px-4 py-3 text-ink-mute">{r.merchant_id ?? "—"}</td>
                  <td className="px-4 py-3 text-ink">{r.action}</td>
                  <td className="px-4 py-3 text-ink-secondary">
                    {r.entity}
                    {r.entity_id ? ` (${r.entity_id})` : ""}
                  </td>
                  <td className="px-4 py-3 text-ink-mute">
                    {r.actor_user_id ?? r.actor_device_id ?? "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
