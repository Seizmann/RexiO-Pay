"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { createClient } from "@/lib/supabase/client";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/button";

/**
 * Admin layout: separate chrome from /app, gated on users.is_platform_admin
 * (read via Supabase RLS users_read_own/admin-bypass policy).
 */
export default function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { dict } = useI18n();
  const a = dict.admin;
  const pathname = usePathname();
  const supabase = createClient();
  const [state, setState] = useState<"loading" | "admin" | "forbidden">(
    "loading",
  );

  useEffect(() => {
    supabase.auth.getUser().then(async ({ data }) => {
      if (!data.user) {
        setState("forbidden");
        return;
      }
      const { data: row } = await supabase
        .schema("public")
        .from("users")
        .select("is_platform_admin")
        .eq("id", data.user.id)
        .maybeSingle();
      setState(row?.is_platform_admin ? "admin" : "forbidden");
    });
  }, [supabase]);

  const tabs = [
    { href: "/rexio-admin", label: a.merchants },
    { href: "/rexio-admin/devices", label: a.devicesTitle },
    { href: "/rexio-admin/sms-stats", label: a.smsStats },
    { href: "/rexio-admin/audit-logs", label: a.auditLogs },
    { href: "/rexio-admin/announcements", label: a.announcements },
  ];

  if (state === "loading") {
    return (
      <main className="flex min-h-dvh items-center justify-center">
        <p className="text-body-md text-ink-mute">{dict.common.loading}</p>
      </main>
    );
  }
  if (state === "forbidden") {
    return (
      <main className="flex min-h-dvh items-center justify-center">
        <div className="text-center">
          <p className="text-heading-md text-ink">{a.notAdmin}</p>
          <Link href="/app">
            <Button variant="secondary" className="mt-4">
              {dict.dash.title}
            </Button>
          </Link>
        </div>
      </main>
    );
  }

  return (
    <div className="min-h-dvh bg-canvas-soft">
      <header className="border-b border-hairline bg-canvas">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
          <p className="text-heading-sm text-ink">{a.title}</p>
          <Link href="/app" className="text-caption text-primary hover:underline">
            {dict.dash.title}
          </Link>
        </div>
        <nav className="mx-auto flex max-w-6xl gap-1 px-6 pb-2">
          {tabs.map((t) => (
            <Link
              key={t.href}
              href={t.href}
              className={`rounded-pill px-3 py-1.5 text-button-sm ${
                pathname === t.href
                  ? "bg-primary text-on-primary"
                  : "text-ink-mute hover:text-ink"
              }`}
            >
              {t.label}
            </Link>
          ))}
        </nav>
      </header>
      <main className="mx-auto max-w-6xl px-6 py-8">{children}</main>
    </div>
  );
}
