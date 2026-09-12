"use client";

import { useEffect, useState } from "react";
import type { WebhookDeliveryRow, WebhookEndpointRow } from "@/lib/supabase-views";
import { createClient } from "@/lib/supabase/client";
import { StatusBadge } from "@/components/ui/badge";
import { useI18n } from "@/lib/i18n";

/**
 * Webhooks (REQUIREMENT §14.8). The backend has no webhook HTTP CRUD,
 * delivery-log, or redeliver endpoints (gap flag #5) — the dispatcher
 * writes to the DB, so this page shows endpoints + deliveries read-only
 * via Supabase RLS and notes the missing actions.
 */
export default function WebhooksPage() {
  const { dict, lang } = useI18n();
  const m = dict.modules;
  const supabase = createClient();
  const [endpoints, setEndpoints] = useState<WebhookEndpointRow[]>([]);
  const [deliveries, setDeliveries] = useState<WebhookDeliveryRow[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function loadAll() {
      const [ep, dl] = await Promise.all([
        supabase
          .from("webhook_endpoints")
          .select("id, url, events, status, created_at")
          .order("created_at", { ascending: false }),
        supabase
          .from("webhook_deliveries")
          .select(
            "id, endpoint_id, event_type, attempt_count, status, last_response_code, created_at",
          )
          .order("created_at", { ascending: false })
          .limit(50),
      ]);
      setEndpoints((ep.data ?? []) as WebhookEndpointRow[]);
      setDeliveries((dl.data ?? []) as WebhookDeliveryRow[]);
      setLoading(false);
    }
    void loadAll();
  }, [supabase]);

  return (
    <div>
      <h1 className="text-display-lg text-ink">{m.webhooksTitle}</h1>
      <p className="mt-3 rounded-md bg-canvas-cream px-4 py-3 text-caption text-ink">
        {m.webhooksStubNote}
      </p>

      <section className="mt-8">
        <h2 className="text-heading-md text-ink">{m.webhooksTitle}</h2>
        {endpoints.length === 0 ? (
          <p className="mt-3 text-body-md text-ink-mute">{m.webhooksEmpty}</p>
        ) : (
          <div className="mt-3 overflow-x-auto rounded-lg border border-hairline bg-canvas">
            <table className="w-full min-w-2xl text-body-tabular">
              <thead>
                <tr className="border-b border-hairline bg-canvas-soft text-left text-ink-mute">
                  <th className="px-4 py-3 font-normal">{m.webhooksUrl}</th>
                  <th className="px-4 py-3 font-normal">{m.webhooksEvents}</th>
                  <th className="px-4 py-3 font-normal">{m.txStatus}</th>
                </tr>
              </thead>
              <tbody>
                {endpoints.map((e) => (
                  <tr key={e.id} className="border-b border-hairline last:border-0">
                    <td className="px-4 py-3 text-ink">{e.url}</td>
                    <td className="px-4 py-3 text-ink-mute">{(e.events ?? []).join(", ") || "*"}</td>
                    <td className="px-4 py-3">
                      <StatusBadge
                        tone={e.status === "active" ? "succeeded" : "muted"}
                        label={e.status === "active" ? dict.status.active : dict.status.disabled}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="mt-8">
        <h2 className="text-heading-md text-ink">{m.webhooksLog}</h2>
        {loading ? (
          <p className="mt-3 text-body-md text-ink-mute">{dict.common.loading}</p>
        ) : deliveries.length === 0 ? (
          <p className="mt-3 text-body-md text-ink-mute">{dict.common.empty}</p>
        ) : (
          <div className="mt-3 overflow-x-auto rounded-lg border border-hairline bg-canvas">
            <table className="w-full min-w-2xl text-body-tabular">
              <thead>
                <tr className="border-b border-hairline bg-canvas-soft text-left text-ink-mute">
                  <th className="px-4 py-3 font-normal">{m.txDate}</th>
                  <th className="px-4 py-3 font-normal">Event</th>
                  <th className="px-4 py-3 font-normal">{m.webhooksAttempts}</th>
                  <th className="px-4 py-3 font-normal">{m.txStatus}</th>
                  <th className="px-4 py-3 font-normal">{m.webhooksResponse}</th>
                </tr>
              </thead>
              <tbody>
                {deliveries.map((d) => (
                  <tr key={d.id} className="border-b border-hairline last:border-0">
                    <td className="px-4 py-3 text-ink-secondary">
                      {new Date(d.created_at).toLocaleString(lang === "bn" ? "bn-BD" : "en-GB")}
                    </td>
                    <td className="px-4 py-3 text-ink">{d.event_type}</td>
                    <td className="tnum px-4 py-3">{d.attempt_count}</td>
                    <td className="px-4 py-3">
                      <StatusBadge
                        tone={
                          d.status === "delivered"
                            ? "succeeded"
                            : d.status === "failed"
                              ? "danger"
                              : "pending"
                        }
                        label={d.status}
                      />
                    </td>
                    <td className="tnum px-4 py-3">{d.last_response_code ?? "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
