"use client";

import Image from "next/image";
import Link from "next/link";
import { useI18n } from "@/lib/i18n";
import { formatTaka } from "@/lib/money";
import { MarketingNav } from "@/components/marketing/nav";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

/**
 * Landing page (REQUIREMENT §14 Marketing, M9).
 * Gradient-mesh hero with the committed mockup, how-it-works, pricing
 * (3 tiers per §18, Pro featured on brand-dark-900), FAQ, compliance
 * disclaimer matching §1 in substance.
 */

const PLANS = [
  {
    code: "starter" as const,
    price: 0,
    noteKey: "planStarterNote" as const,
    limits: { profiles: "2", devices: "1", sessions: "100", webhooks: "1", team: "1" },
  },
  {
    code: "pro" as const,
    price: 1000,
    noteKey: "planProNote" as const,
    featured: true,
    limits: { profiles: "10", devices: "3", sessions: "5,000", webhooks: "5", team: "5" },
  },
  {
    code: "business" as const,
    price: 3000,
    noteKey: "planBusinessNote" as const,
    limits: {
      profiles: "unlimited",
      devices: "unlimited",
      sessions: "unlimited",
      webhooks: "unlimited",
      team: "unlimited",
    },
  },
];

export default function LandingPage() {
  const { dict, lang } = useI18n();
  const m = dict.marketing;

  const faqs = [
    { q: m.faq1Q, a: m.faq1A },
    { q: m.faq2Q, a: m.faq2A },
    { q: m.faq3Q, a: m.faq3A },
    { q: m.faq4Q, a: m.faq4A },
    { q: m.faq5Q, a: m.faq5A },
  ];

  const steps = [
    { title: m.howStep1Title, body: m.howStep1Body },
    { title: m.howStep2Title, body: m.howStep2Body },
    { title: m.howStep3Title, body: m.howStep3Body },
    { title: m.howStep4Title, body: m.howStep4Body },
  ];

  return (
    <>
      <MarketingNav />

      {/* Hero — gradient mesh occupies the upper third (DESIGN.md) */}
      <section className="relative">
        <picture className="absolute inset-x-0 top-0 -z-10 h-full w-full">
          <source media="(max-width: 767px)" srcSet="/media/hero-mesh-mobile.webp" />
          <source media="(max-width: 1023px)" srcSet="/media/hero-mesh-tablet.webp" />
          <img
            src="/media/hero-mesh-desktop.webp"
            alt=""
            className="h-full w-full object-cover"
          />
        </picture>
        <div className="mx-auto flex w-full max-w-6xl flex-col items-start px-6 pb-24 pt-20 md:pt-28">
          <span className="rounded-pill bg-primary-bg-subdued-hover px-3 py-1 text-micro-cap uppercase text-primary-deep">
            {m.heroEyebrow}
          </span>
          <h1 className="mt-6 max-w-3xl text-display-xl text-ink md:text-display-xxl">
            {m.heroTitle}
          </h1>
          <p className="mt-5 max-w-xl text-body-lg text-ink-secondary">
            {m.heroBody}
          </p>
          <div className="mt-8 flex flex-wrap items-center gap-4">
            <Link href="/signup">
              <Button>{m.heroCta}</Button>
            </Link>
            <Link href="/docs">
              <Button variant="secondary">{m.heroSecondary}</Button>
            </Link>
          </div>
          <div className="mt-14 w-full max-w-2xl overflow-hidden rounded-lg shadow-level-2">
            <Image
              src="/media/phone-mockup.webp"
              alt=""
              width={1200}
              height={800}
              className="h-auto w-full"
              priority
            />
          </div>
        </div>
      </section>

      {/* How it works */}
      <section id="how" className="bg-canvas-soft px-6 py-24">
        <div className="mx-auto w-full max-w-6xl">
          <h2 className="text-display-lg text-ink">{m.howTitle}</h2>
          <div className="mt-10 grid gap-6 md:grid-cols-2">
            {steps.map((step, i) => (
              <Card key={i}>
                <p className="tnum text-micro-cap uppercase text-ink-mute">
                  {String(i + 1).padStart(2, "0")}
                </p>
                <h3 className="mt-2 text-heading-lg text-ink">{step.title}</h3>
                <p className="mt-2 text-body-md text-ink-secondary">{step.body}</p>
              </Card>
            ))}
          </div>
        </div>
      </section>

      {/* Pricing */}
      <section id="pricing" className="px-6 py-24">
        <div className="mx-auto w-full max-w-6xl">
          <h2 className="text-display-lg text-ink">{m.pricingTitle}</h2>
          <p className="mt-3 max-w-xl text-body-md text-ink-secondary">
            {m.pricingBody}
          </p>
          <div className="mt-10 grid gap-6 lg:grid-cols-3">
            {PLANS.map((plan) => (
              <Card
                key={plan.code}
                variant={plan.featured ? "featured" : "light"}
                className="flex flex-col"
              >
                <h3 className="text-heading-lg capitalize">{plan.code}</h3>
                <p className={`mt-1 text-caption ${plan.featured ? "text-on-primary/70" : "text-ink-mute"}`}>
                  {m[plan.noteKey]}
                </p>
                <p className="tnum mt-6 text-display-md">
                  {plan.price === 0 ? formatTaka(0, lang) : formatTaka(plan.price, lang)}
                  <span className={`text-caption ${plan.featured ? "text-on-primary/70" : "text-ink-mute"}`}>
                    {m.perMonth}
                  </span>
                </p>
                <ul className="mt-6 flex flex-1 flex-col gap-2 text-body-md">
                  <LimitRow label={m.profilesLimit} value={plan.limits.profiles} featured={plan.featured} />
                  <LimitRow label={m.devicesLimit} value={plan.limits.devices} featured={plan.featured} />
                  <LimitRow label={m.sessionsLimit} value={plan.limits.sessions} featured={plan.featured} />
                  <LimitRow label={m.webhooksLimit} value={plan.limits.webhooks} featured={plan.featured} />
                  <LimitRow label={m.teamLimit} value={plan.limits.team} featured={plan.featured} />
                </ul>
                <Link href="/signup" className="mt-8">
                  <Button variant={plan.featured ? "secondary" : "primary"} className="w-full">
                    {m.choosePlan}
                  </Button>
                </Link>
              </Card>
            ))}
          </div>
        </div>
      </section>

      {/* FAQ */}
      <section id="faq" className="bg-canvas-soft px-6 py-24">
        <div className="mx-auto w-full max-w-3xl">
          <h2 className="text-display-lg text-ink">{m.faqTitle}</h2>
          <div className="mt-10 flex flex-col divide-y divide-hairline">
            {faqs.map((faq, i) => (
              <details key={i} className="group py-4">
                <summary className="cursor-pointer list-none text-heading-sm text-ink marker:hidden">
                  {faq.q}
                </summary>
                <p className="mt-3 text-body-md text-ink-secondary">{faq.a}</p>
              </details>
            ))}
          </div>
        </div>
      </section>

      {/* Compliance disclaimer — §1 positioning, substance preserved */}
      <section className="bg-canvas-cream px-6 py-16">
        <div className="mx-auto w-full max-w-3xl">
          <h2 className="text-heading-lg text-ink">{m.disclaimerTitle}</h2>
          <p className="mt-3 text-body-md text-ink-secondary">{m.disclaimerBody}</p>
        </div>
      </section>
    </>
  );
}

function LimitRow({
  label,
  value,
  featured,
}: {
  label: string;
  value: string;
  featured?: boolean;
}) {
  const { dict } = useI18n();
  const display =
    value === "unlimited"
      ? dict.marketing.unlimited
      : value;
  return (
    <li className="flex items-center justify-between gap-4">
      <span className={featured ? "text-on-primary/80" : "text-ink-mute"}>{label}</span>
      <span className={`tnum ${featured ? "text-on-primary" : "text-ink"}`}>{display}</span>
    </li>
  );
}
