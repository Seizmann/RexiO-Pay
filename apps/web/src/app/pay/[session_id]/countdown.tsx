"use client";

import { useEffect, useState } from "react";
import { useI18n } from "@/lib/i18n";

function secondsLeft(iso: string): number {
  return Math.max(0, Math.floor((new Date(iso).getTime() - Date.now()) / 1000));
}

function formatClock(total: number, locale: "en" | "bn"): string {
  const m = Math.floor(total / 60);
  const s = total % 60;
  const pad = String(s).padStart(2, "0");
  return locale === "bn" ? `${m}:${pad}` : `${m}:${pad}`;
}

/** Countdown derived from expires_at; ticks every second. */
export function Countdown({
  expiresAt,
  onExpired,
}: {
  expiresAt: string;
  onExpired?: () => void;
}) {
  const { dict, lang } = useI18n();
  const [left, setLeft] = useState(() => secondsLeft(expiresAt));

  useEffect(() => {
    setLeft(secondsLeft(expiresAt));
    const id = setInterval(() => {
      const next = secondsLeft(expiresAt);
      setLeft(next);
      if (next <= 0) {
        clearInterval(id);
        onExpired?.();
      }
    }, 1000);
    return () => clearInterval(id);
  }, [expiresAt, onExpired]);

  return (
    <div className="flex items-baseline gap-2">
      <span className="text-caption text-ink-mute">{dict.checkout.timeLeft}</span>
      <span className="tnum text-body-lg text-ink-secondary">
        {formatClock(left, lang)}
      </span>
    </div>
  );
}
