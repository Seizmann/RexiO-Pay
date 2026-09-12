"use client";

import Link from "next/link";
import { useI18n } from "@/lib/i18n";
import { LangToggle } from "@/components/lang-toggle";
import { Button } from "@/components/ui/button";

/** Floating nav over the gradient-mesh hero (DESIGN.md nav-bar-on-mesh). */
export function MarketingNav() {
  const { dict } = useI18n();
  return (
    <nav className="mx-auto flex w-full max-w-6xl items-center justify-between rounded-xs bg-canvas px-6 py-4 shadow-level-1">
      <Link href="/" className="text-heading-sm font-normal text-ink">
        RexiO&nbsp;Pay
      </Link>
      <div className="hidden items-center gap-6 text-body-md text-ink-mute-2 md:flex">
        <a href="#how" className="hover:text-ink">
          {dict.marketing.navProduct}
        </a>
        <a href="#pricing" className="hover:text-ink">
          {dict.marketing.navPricing}
        </a>
        <a href="#faq" className="hover:text-ink">
          {dict.marketing.navFaq}
        </a>
        <Link href="/docs" className="hover:text-ink">
          {dict.marketing.navDocs}
        </Link>
      </div>
      <div className="flex items-center gap-3">
        <LangToggle />
        <Link href="/login" className="hidden text-body-md text-ink hover:text-primary sm:block">
          {dict.marketing.navSignIn}
        </Link>
        <Link href="/signup">
          <Button size="sm">{dict.marketing.navGetStarted}</Button>
        </Link>
      </div>
    </nav>
  );
}
