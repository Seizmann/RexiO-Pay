"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useI18n } from "@/lib/i18n";
import type { Merchant } from "@rexio-pay/shared-types";
import { LangToggle } from "@/components/lang-toggle";
import { createClient } from "@/lib/supabase/client";
import { formatTaka } from "@/lib/money";

const NAV = [
  { href: "/app", key: "overview" as const },
  { href: "/app/transactions", key: "transactions" as const },
  { href: "/app/profiles", key: "profiles" as const },
  { href: "/app/devices", key: "devices" as const },
  { href: "/app/inbox", key: "inbox" as const },
  { href: "/app/links", key: "links" as const },
  { href: "/app/keys", key: "keys" as const },
  { href: "/app/webhooks", key: "webhooks" as const },
  { href: "/app/branding", key: "branding" as const },
  { href: "/app/domains", key: "domains" as const },
  { href: "/app/settings", key: "settings" as const },
];

/** Dark-navy dashboard sidebar (DESIGN.md dark-app shell). */
export function DashboardSidebar({ merchant }: { merchant: Merchant | null }) {
  const { dict } = useI18n();
  const pathname = usePathname();
  const supabase = createClient();

  return (
    <aside className="flex w-full shrink-0 flex-col bg-brand-dark-900 px-4 py-6 text-on-primary md:h-dvh md:w-60 md:sticky md:top-0">
      <Link href="/app" className="px-2 text-heading-sm">
        RexiO&nbsp;Pay
      </Link>
      {merchant ? (
        <p className="mt-1 px-2 text-caption text-on-primary/60">
          {merchant.name} · {dict.dash.planBadge}{" "}
          <span className="capitalize">{merchant.plan_id}</span>
        </p>
      ) : null}
      <nav className="mt-6 flex flex-row flex-wrap gap-1 md:flex-1 md:flex-col md:flex-nowrap md:overflow-y-auto">
        {NAV.map((item) => {
          const active =
            item.href === "/app"
              ? pathname === "/app"
              : pathname.startsWith(item.href);
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`rounded-sm px-3 py-2 text-body-md transition-colors ${
                active
                  ? "bg-on-primary/10 text-on-primary"
                  : "text-on-primary/60 hover:bg-on-primary/5 hover:text-on-primary"
              }`}
            >
              {dict.nav[item.key]}
            </Link>
          );
        })}
      </nav>
      <div className="mt-6 flex items-center justify-between gap-2 px-2 md:flex-col md:items-stretch">
        <LangToggle className="border-on-primary/20" />
        <button
          type="button"
          onClick={() => void supabase.auth.signOut()}
          className="px-1 text-body-md text-on-primary/60 hover:text-on-primary"
        >
          {dict.common.signOut}
        </button>
      </div>
    </aside>
  );
}

export function TakaCell({ amount }: { amount: number }) {
  const { lang } = useI18n();
  return <span className="tnum">{formatTaka(amount, lang)}</span>;
}
