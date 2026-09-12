"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { StatusBadge } from "@/components/ui/badge";
import { useI18n } from "@/lib/i18n";

/** Global device health across all merchants (REQUIREMENT §17). */
export default function AdminDevicesPage() {
  const { dict } = useI18n();
  const a = dict.admin;
  const [rows, setRows] = useState<Record<string, unknown>[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api
      .get<{ data: Record<string, unknown>[] }>("/rexio-admin/devices")
      .then((r) => setRows(r.data))
      .catch(() => undefined)
      .finally(() => setLoading(false));
  }, []);

  return (
    <div>
      <h1 className="text-display-lg text-ink">{a.devicesTitle}</h1>
      {loading ? (
        <p className="mt-8 text-body-md text-ink-mute">{dict.common.loading}</p>
      ) : rows.length === 0 ? (
        <p className="mt-8 text-body-md text-ink-mute">{dict.modules.devicesEmpty}</p>
      ) : (
        <div className="mt-6 overflow-x-auto rounded-lg border border-hairline bg-canvas">
          <table className="w-full min-w-2xl text-body-tabular">
            <thead>
              <tr className="border-b border-hairline bg-canvas-soft text-left text-ink-mute">
                <th className="px-4 py-3 font-normal">{dict.modules.keysName}</th>
                <th className="px-4 py-3 font-normal">{a.deviceMerchant}</th>
                <th className="px-4 py-3 font-normal">{dict.modules.txStatus}</th>
                <th className="px-4 py-3 font-normal">{dict.modules.devicesLastBeat}</th>
                <th className="px-4 py-3 font-normal">{dict.modules.devicesSigFails}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((d) => (
                <tr key={String(d.id)} className="border-b border-hairline last:border-0">
                  <td className="px-4 py-3 text-ink">
                    {String(d.name ?? d.id)}
                    {d.model ? <span className="text-ink-mute"> · {String(d.model)}</span> : null}
                  </td>
                  <td className="px-4 py-3 text-ink-mute">{String(d.merchant_id ?? "—")}</td>
                  <td className="px-4 py-3">
                    <StatusBadge
                      tone={
                        d.status === "active" ? "succeeded" : d.status === "offline" ? "danger" : "muted"
                      }
                      label={String(d.status ?? "")}
                    />
                  </td>
                  <td className="px-4 py-3 text-ink-secondary">
                    {d.last_heartbeat_at
                      ? new Date(String(d.last_heartbeat_at)).toLocaleString()
                      : "—"}
                  </td>
                  <td className="tnum px-4 py-3">{String(d.failed_sig_count ?? 0)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
