"use client";

import { useCallback, useEffect, useState } from "react";
import type { CheckoutSession, Device } from "@/lib/supabase-views";
import { createClient } from "@/lib/supabase/client";
import { useI18n } from "@/lib/i18n";
import { formatTaka, formatTakaCompact } from "@/lib/money";
import { StatusBadge, type StatusTone } from "@/components/ui/badge";

/**
 * Overview (REQUIREMENT §14.1): today's sessions, volume, match rate,
 * device strip. SSE does not exist in the backend (gap flag #3), so this
 * refreshes by polling Supabase reads every 10 seconds.
 */
export default function OverviewPage() {
  const { dict, lang } = useI18n();
  const d = dict.dash;
  const m = dict.modules;
  const supabase = createClient();

  const [sessions, setSessions] = useState<CheckoutSession[]>([]);
  const [payments, setPayments] = useState<{ amount: number }[]>([]);
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    const startOfDay = new Date();
    startOfDay.setHours(0, 0, 0, 0);

    const [sessionsRes, paymentsRes, devicesRes] = await Promise.all([
      supabase
        .from("checkout_sessions")
        .select("id, status, amount, customer, created_at, expires_at, needs_review")
        .gte("created_at", startOfDay.toISOString())
        .order("created_at", { ascending: false })
        .limit(20),
      supabase
        .from("payments")
        .select("amount")
        .gte("verified_at", startOfDay.toISOString()),
      supabase.from("devices").select("id, name, status, last_heartbeat_at"),
    ]);
    setSessions((sessionsRes.data ?? []) as CheckoutSession[]);
    setPayments((paymentsRes.data ?? []) as { amount: number }[]);
    setDevices((devicesRes.data ?? []) as Device[]);
    setLoading(false);
  }, [supabase]);

  useEffect(() => {
    void load();
    const id = setInterval(() => void load(), 10_000);
    return () => clearInterval(id);
  }, [load]);

  const succeeded = sessions.filter((s) => s.status === "succeeded").length;
  const pending = sessions.filter((s) => s.status === "pending").length;
  const expired = sessions.filter((s) => s.status === "expired").length;
  const volume = payments.reduce((sum, p) => sum + p.amount, 0);
  const matchRate = sessions.length > 0 ? Math.round((succeeded / sessions.length) * 100) : null;
  const online = devices.filter((dev) => dev.status === "active").length;

  return (
    <div>
      <div className="flex items-center justify-between">
        <h1 className="text-display-lg text-ink">{m.overviewTitle}</h1>
        <span className="text-caption text-ink-mute">{d.today}</span>
      </div>

      <div className="mt-8 grid grid-cols-2 gap-4 lg:grid-cols-4">
        <Tile label={d.todaySucceeded} value={String(succeeded)} tone="succeeded" />
        <Tile label={d.todayPending} value={String(pending)} tone="pending" />
        <Tile label={d.todayExpired} value={String(expired)} tone="muted" />
        <Tile label={d.todayVolume} value={formatTakaCompact(volume, lang)} tone="succeeded" />
      </div>

      <div className="mt-4 grid grid-cols-2 gap-4">
        <Tile
          label={d.matchRate}
          value={matchRate === null ? "—" : `${matchRate}%`}
          tone="info"
        />
        <Tile
          label={d.devices}
          value={`${online}/${devices.length}`}
          tone={online > 0 ? "succeeded" : "danger"}
        />
      </div>

      <section className="mt-10">
        <h2 className="text-heading-md text-ink">{d.recentTitle}</h2>
        {loading ? (
          <p className="mt-3 text-body-md text-ink-mute">{dict.common.loading}</p>
        ) : sessions.length === 0 ? (
          <p className="mt-3 text-body-md text-ink-mute">{d.recentEmpty}</p>
        ) : (
          <div className="mt-3 overflow-x-auto rounded-lg border border-hairline bg-canvas">
            <table className="w-full text-body-tabular">
              <tbody>
                {sessions.slice(0, 8).map((s) => {
                  const name =
                    typeof s.customer === "object" && s.customer !== null && "name" in s.customer
                      ? String((s.customer as Record<string, unknown>).name ?? "")
                      : "";
                  return (
                    <tr key={s.id} className="border-b border-hairline last:border-0">
                      <td className="px-4 py-3 text-ink-mute">{s.id}</td>
                      <td className="px-4 py-3">{name || "—"}</td>
                      <td className="tnum px-4 py-3 text-ink">{formatTaka(s.amount, lang)}</td>
                      <td className="px-4 py-3">
                        <StatusBadge
                          tone={s.status === "succeeded" ? "succeeded" : s.status === "pending" ? "pending" : "muted"}
                          label={dict.status[s.status as keyof typeof dict.status] ?? s.status}
                        />
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}

function Tile({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone: StatusTone;
}) {
  return (
    <div className="rounded-lg border border-hairline bg-canvas p-5">
      <p className="text-caption text-ink-mute">{label}</p>
      <p className="tnum mt-2 text-display-md text-ink">{value}</p>
      <div className="mt-2">
        <StatusBadge tone={tone} label="" />
      </div>
    </div>
  );
}
