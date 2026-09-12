"use client";

import { useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import type { SmsStatRow } from "@rexio-pay/shared-types";
import { useI18n } from "@/lib/i18n";

/**
 * SMS/parser stats (REQUIREMENT §14 admin bullet). The endpoint aggregates
 * provider x parse_status per day (not per template) over 30 days.
 */
export default function AdminSmsStatsPage() {
  const { dict } = useI18n();
  const a = dict.admin;
  const [rows, setRows] = useState<SmsStatRow[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api
      .get<{ data: SmsStatRow[] }>("/rexio-admin/sms/stats")
      .then((r) => setRows(r.data))
      .catch(() => undefined)
      .finally(() => setLoading(false));
  }, []);

  const summary = useMemo(() => {
    const map = new Map<string, { parsed: number; failed: number }>();
    for (const r of rows) {
      const key = r.provider || "unknown";
      const entry = map.get(key) ?? { parsed: 0, failed: 0 };
      if (r.parse_status === "parsed") entry.parsed += r.cnt;
      else entry.failed += r.cnt;
      map.set(key, entry);
    }
    return [...map.entries()];
  }, [rows]);

  return (
    <div>
      <h1 className="text-display-lg text-ink">{a.smsStats}</h1>
      <p className="mt-3 text-caption text-ink-mute">{a.smsStatsNote}</p>
      {loading ? (
        <p className="mt-8 text-body-md text-ink-mute">{dict.common.loading}</p>
      ) : summary.length === 0 ? (
        <p className="mt-8 text-body-md text-ink-mute">{dict.common.empty}</p>
      ) : (
        <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {summary.map(([provider, s]) => {
            const total = s.parsed + s.failed;
            const rate = total > 0 ? Math.round((s.parsed / total) * 100) : null;
            return (
              <div key={provider} className="rounded-lg border border-hairline bg-canvas p-5">
                <p className="text-heading-sm capitalize text-ink">{provider}</p>
                <p className="tnum mt-3 text-display-md text-ink">
                  {rate === null ? "—" : `${rate}%`}
                </p>
                <p className="tnum mt-1 text-caption text-ink-mute">
                  {a.parsed}: {s.parsed} · {a.failed}: {s.failed}
                </p>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
