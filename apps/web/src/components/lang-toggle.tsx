"use client";

import { useI18n } from "@/lib/i18n";

/** bn/en toggle for nav bars. Persists via the i18n provider's cookie. */
export function LangToggle({ className = "" }: { className?: string }) {
  const { lang, setLang, dict } = useI18n();
  return (
    <div
      className={`inline-flex items-center rounded-pill border border-hairline p-0.5 ${className}`}
      role="group"
      aria-label={dict.common.language}
    >
      {(["en", "bn"] as const).map((code) => (
        <button
          key={code}
          type="button"
          onClick={() => setLang(code)}
          className={`rounded-pill px-3 py-1 text-button-sm transition-colors ${
            lang === code
              ? "bg-primary text-on-primary"
              : "text-ink-mute hover:text-ink"
          }`}
          aria-pressed={lang === code}
        >
          {code === "en" ? dict.common.english : dict.common.bangla}
        </button>
      ))}
    </div>
  );
}
