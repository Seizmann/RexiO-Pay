"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import type { CheckoutSession, PaymentView } from "@/lib/supabase-views";
import { createClient } from "@/lib/supabase/client";
import { useI18n } from "@/lib/i18n";
import { formatTaka } from "@/lib/money";
import { StatusBadge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

/**
 * Transactions (REQUIREMENT §14.2): checkout sessions joined with payments
 * via Supabase RLS reads. There is no backend CSV endpoint (gap flag #4),
 * so export generates the CSV client-side from the loaded rows.
 */
export default function TransactionsPage() {
  const { dict, lang } = useI18n();
  const m = dict.modules;
  const supabase = createClient();

  const [rows, setRows] = useState<
    { session: CheckoutSession; payment: PaymentView | null }[]
  >([]);
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [provider, setProvider] = useState("");
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    let query = supabase
      .from("checkout_sessions")
      .select(
        "*, payments(id, provider, account_type, trx_id, sender_number, verification_source)",
      )
      .order("created_at", { ascending: false })
      .limit(200);
    if (from) query = query.gte("created_at", new Date(from).toISOString());
    if (to) query = query.lte("created_at", new Date(`${to}T23:59:59`).toISOString());
    const { data } = await query;
    setRows(
      ((data ?? []) as (CheckoutSession & { payments: PaymentView[] | null })[]).map(
        (s) => ({ session: s, payment: s.payments?.[0] ?? null }),
      ),
    );
    setLoading(false);
  }, [supabase, from, to]);

  useEffect(() => {
    void load();
  }, [load]);

  const filtered = useMemo(
    () => (provider ? rows.filter((r) => r.payment?.provider === provider) : rows),
    [rows, provider],
  );

  function exportCsv() {
    const header = ["session_id", "date", "amount_bdt", "customer", "provider", "trx_id", "status", "source"];
    const lines = filtered.map((r) =>
      [
        r.session.id,
        r.session.created_at,
        r.session.amount,
        nameOf(r.session.customer),
        r.payment?.provider ?? "",
        r.payment?.trx_id ?? "",
        r.session.status,
        r.session.source,
      ]
        .map((cell) => `"${String(cell).replace(/"/g, '""')}"`)
        .join(","),
    );
    const blob = new Blob([[header.join(","), ...lines].join("\n")], {
      type: "text/csv;charset=utf-8",
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `rexio-pay-transactions-${new Date().toISOString().slice(0, 10)}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-4">
        <h1 className="text-display-lg text-ink">{m.txTitle}</h1>
        <Button variant="secondary" size="sm" onClick={exportCsv} disabled={filtered.length === 0}>
          {m.txExport}
        </Button>
      </div>

      <div className="mt-6 flex flex-wrap items-end gap-4">
        <Input id="f-from" type="date" label={`${m.txDate} (from)`} value={from} onChange={(e) => setFrom(e.target.value)} className="w-44" />
        <Input id="f-to" type="date" label={`${m.txDate} (to)`} value={to} onChange={(e) => setTo(e.target.value)} className="w-44" />
        <label className="flex flex-col gap-1.5">
          <span className="text-caption text-ink-secondary">{m.txProvider}</span>
          <select
            value={provider}
            onChange={(e) => setProvider(e.target.value)}
            className="h-10 rounded-sm border border-hairline-input bg-canvas px-3 text-body-md"
          >
            <option value="">{dict.common.all}</option>
            <option value="bkash">bKash</option>
            <option value="nagad">Nagad</option>
          </select>
        </label>
      </div>

      {loading ? (
        <p className="mt-8 text-body-md text-ink-mute">{dict.common.loading}</p>
      ) : filtered.length === 0 ? (
        <p className="mt-8 text-body-md text-ink-mute">{m.txEmpty}</p>
      ) : (
        <div className="mt-6 overflow-x-auto rounded-lg border border-hairline bg-canvas">
          <table className="w-full min-w-3xl text-body-tabular">
            <thead>
              <tr className="border-b border-hairline bg-canvas-soft text-left text-ink-mute">
                <th className="px-4 py-3 font-normal">{m.txSession}</th>
                <th className="px-4 py-3 font-normal">{m.txDate}</th>
                <th className="px-4 py-3 font-normal">{m.txAmount}</th>
                <th className="px-4 py-3 font-normal">{m.txCustomer}</th>
                <th className="px-4 py-3 font-normal">{m.txProvider}</th>
                <th className="px-4 py-3 font-normal">{m.txTrxid}</th>
                <th className="px-4 py-3 font-normal">{m.txStatus}</th>
                <th className="px-4 py-3 font-normal">{m.txSource}</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((r) => (
                <tr key={r.session.id} className="border-b border-hairline last:border-0">
                  <td className="px-4 py-3 text-ink-mute">{r.session.id}</td>
                  <td className="px-4 py-3 text-ink-secondary">
                    {new Date(r.session.created_at).toLocaleString(lang === "bn" ? "bn-BD" : "en-GB")}
                  </td>
                  <td className="tnum px-4 py-3 text-ink">{formatTaka(r.session.amount, lang)}</td>
                  <td className="px-4 py-3">{nameOf(r.session.customer) || "—"}</td>
                  <td className="px-4 py-3 capitalize">{r.payment?.provider ?? "—"}</td>
                  <td className="tnum px-4 py-3">{r.payment?.trx_id ?? "—"}</td>
                  <td className="px-4 py-3">
                    <StatusBadge
                      tone={
                        r.session.status === "succeeded"
                          ? "succeeded"
                          : r.session.status === "pending"
                            ? "pending"
                            : "muted"
                      }
                      label={dict.status[r.session.status as keyof typeof dict.status] ?? r.session.status}
                    />
                  </td>
                  <td className="px-4 py-3 text-ink-mute">{r.session.source}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function nameOf(customer: Record<string, unknown> | null): string {
  if (customer && "name" in customer) return String(customer.name ?? "");
  return "";
}
