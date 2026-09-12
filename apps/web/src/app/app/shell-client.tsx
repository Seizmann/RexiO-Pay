"use client";

import { useEffect, useState } from "react";
import type { Announcement } from "@rexio-pay/shared-types";
import { createClient } from "@/lib/supabase/client";
import { DashboardSidebar } from "./sidebar";
import { ConnectGate, useMerchant } from "./merchant-context";
import { useI18n } from "@/lib/i18n";

/**
 * Dashboard shell: dark sidebar + announcements banner (announcements are
 * public-read per RLS, so no API key needed) + connect-key gate.
 */
export function DashboardShellClient({
  children,
}: {
  children: React.ReactNode;
}) {
  const { dict } = useI18n();
  const { merchant, state } = useMerchant();
  const supabase = createClient();
  const [announcements, setAnnouncements] = useState<Announcement[]>([]);

  useEffect(() => {
    const now = new Date().toISOString();
    supabase
      .from("announcements")
      .select("id, title, body, severity, expires_at, created_at")
      .or(`expires_at.is.null,expires_at.gt.${now}`)
      .order("created_at", { ascending: false })
      .limit(3)
      .then(({ data }) => setAnnouncements((data ?? []) as Announcement[]));
  }, [supabase]);

  return (
    <div className="flex min-h-dvh flex-col bg-canvas-soft md:flex-row">
      <DashboardSidebar merchant={merchant} />
      <main className="flex-1 px-6 py-8 md:px-10">
        {announcements.map((a) => (
          <div
            key={a.id}
            className="mb-6 rounded-md border border-hairline bg-canvas-cream px-4 py-3"
          >
            <p className="text-micro-cap uppercase text-ink-mute">
              {dict.dash.announcement} · {a.severity}
            </p>
            <p className="text-body-md text-ink">{a.body}</p>
          </div>
        ))}
        {state === "loading" ? (
          <p className="text-body-md text-ink-mute">{dict.common.loading}</p>
        ) : state === "disconnected" ? (
          <ConnectGate />
        ) : (
          children
        )}
      </main>
    </div>
  );
}
