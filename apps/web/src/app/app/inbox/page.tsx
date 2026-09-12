"use client";

import { useCallback, useEffect, useState } from "react";
import type { SmsRow } from "@/lib/supabase-views";
import { createClient } from "@/lib/supabase/client";
import { StatusBadge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useCopy } from "@/lib/use-copy";
import { useI18n } from "@/lib/i18n";

/**
 * SMS Inbox (REQUIREMENT §14.5): unmatched/duplicate/suspect rows read via
 * Supabase RLS. Attach-to-session, dismiss, and manual paste need backend
 * endpoints that do not exist yet (gap flag #6); copy raw works locally.
 */
export default function InboxPage() {
  const { dict, lang } = useI18n();
  const m = dict.modules;
  const supabase = createClient();
  const { copied, copy } = useCopy();
  const [rows, setRows] = useState<SmsRow[]>([]);
  const [filter, setFilter] = useState<string>("unmatched");
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    let query = supabase
      .from("sms_messages")
      .select(
        "id, provider, match_status, parse_status, raw_text, received_at, matched_session_id",
      )
      .order("received_at", { ascending: false })
      .limit(100);
    if (filter !== "all") query = query.eq("match_status", filter);
    const { data } = await query;
    setRows((data ?? []) as SmsRow[]);
    setLoading(false);
  }, [supabase, filter]);

  useEffect(() => {
    void load();
  }, [load]);

  const filters = ["unmatched", "duplicate", "suspect", "held", "all"];

  return (
    <div>
      <h1 className="text-display-lg text-ink">{m.inboxTitle}</h1>
      <p className="mt-3 rounded-md bg-canvas-cream px-4 py-3 text-caption text-ink">
        {m.inboxStubNote}
      </p>

      <div className="mt-6 flex flex-wrap gap-2">
        {filters.map((f) => (
          <button
            key={f}
            type="button"
            onClick={() => setFilter(f)}
            className={`rounded-pill px-3 py-1.5 text-button-sm capitalize ${
              filter === f ? "bg-primary text-on-primary" : "bg-canvas text-ink-mute border border-hairline"
            }`}
          >
            {f === "all" ? m.inboxFilterAll : f}
          </button>
        ))}
      </div>

      {loading ? (
        <p className="mt-8 text-body-md text-ink-mute">{dict.common.loading}</p>
      ) : rows.length === 0 ? (
        <p className="mt-8 text-body-md text-ink-mute">{m.inboxEmpty}</p>
      ) : (
        <div className="mt-6 flex flex-col gap-3">
          {rows.map((r) => (
            <div key={r.id} className="rounded-lg border border-hairline bg-canvas p-4">
              <div className="flex flex-wrap items-center gap-3">
                <span className="capitalize text-body-md text-ink">
                  {r.provider ?? "—"}
                </span>
                {r.match_status ? (
                  <StatusBadge
                    tone={
                      r.match_status === "matched"
                        ? "succeeded"
                        : r.match_status === "suspect"
                          ? "danger"
                          : "pending"
                    }
                    label={r.match_status}
                  />
                ) : null}
                {r.parse_status === "failed" ? (
                  <StatusBadge tone="danger" label="parse failed" />
                ) : null}
                <span className="text-caption text-ink-mute">
                  {new Date(r.received_at).toLocaleString(lang === "bn" ? "bn-BD" : "en-GB")}
                </span>
                <span className="flex-1" />
                <Button variant="ghost" size="sm" onClick={() => copy(r.raw_text, r.id)}>
                  {copied === r.id ? dict.common.copied : m.inboxRaw}
                </Button>
              </div>
              <pre className="mt-2 whitespace-pre-wrap rounded-sm bg-canvas-soft px-3 py-2 text-caption text-ink-secondary">
                {r.raw_text}
              </pre>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
