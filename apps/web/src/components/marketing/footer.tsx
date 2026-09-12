"use client";

import Link from "next/link";
import { useI18n } from "@/lib/i18n";

const ORG_LINKS = [
  { href: "https://spritexai.pro.bd", label: "SpritexAI" },
  { href: "https://sijan.pro.bd", label: "Mohammad Sijan" },
  { href: "https://rexio.pro", label: "RexiO" },
];

/** Site footer per DESIGN.md footer-light. */
export function MarketingFooter() {
  const { dict } = useI18n();
  return (
    <footer className="border-t border-hairline bg-canvas px-6 py-16 text-caption text-ink-mute">
      <div className="mx-auto grid w-full max-w-6xl grid-cols-2 gap-8 md:grid-cols-4">
        <div>
          <p className="text-heading-sm text-ink">RexiO Pay</p>
          <p className="mt-2">
            SMS-verified bKash and Nagad payments for Bangladesh.
          </p>
        </div>
        <div>
          <p className="text-ink-secondary">{dict.marketing.footerProduct}</p>
          <ul className="mt-2 flex flex-col gap-1.5">
            <li>
              <Link href="/docs" className="hover:text-ink">
                {dict.marketing.footerDocs}
              </Link>
            </li>
            <li>
              <a href="#pricing" className="hover:text-ink">
                {dict.marketing.navPricing}
              </a>
            </li>
            <li>
              <a href="#faq" className="hover:text-ink">
                {dict.marketing.navFaq}
              </a>
            </li>
          </ul>
        </div>
        <div>
          <p className="text-ink-secondary">{dict.marketing.footerOrg}</p>
          <ul className="mt-2 flex flex-col gap-1.5">
            {ORG_LINKS.map((link) => (
              <li key={link.href}>
                <a
                  href={link.href}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="hover:text-ink"
                >
                  {link.label}
                </a>
              </li>
            ))}
          </ul>
        </div>
        <div>
          <p className="text-ink-secondary">&nbsp;</p>
          <ul className="mt-2 flex flex-col gap-1.5">
            <li>
              <a href="/llms.txt" className="hover:text-ink">
                llms.txt
              </a>
            </li>
          </ul>
        </div>
      </div>
      <div className="mx-auto mt-12 w-full max-w-6xl">
        <p>
          © {new Date().getFullYear()} SpritexAI. RexiO Pay does not hold, move,
          or pool merchant funds. Merchants are responsible for their own MFS
          compliance.
        </p>
      </div>
    </footer>
  );
}
