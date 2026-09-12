"use client";

import { useI18n } from "@/lib/i18n";

export type StatusTone =
  | "succeeded"
  | "pending"
  | "muted"
  | "danger"
  | "info";

const toneClasses: Record<StatusTone, string> = {
  succeeded: "bg-primary-bg-subdued-hover text-primary-deep",
  pending: "bg-canvas-cream text-lemon",
  muted: "bg-canvas-soft text-ink-mute",
  danger: "bg-canvas-soft text-ruby",
  info: "bg-canvas-soft text-ink-secondary",
};

export interface StatusBadgeProps {
  tone: StatusTone;
  label: string;
}

/** Soft pill tag per DESIGN.md pill-tag-soft. Colors stay inside the documented palette. */
export function StatusBadge({ tone, label }: StatusBadgeProps) {
  return (
    <span
      className={`inline-flex items-center whitespace-nowrap rounded-pill px-2 py-1 text-micro-cap uppercase ${toneClasses[tone]}`}
    >
      {label}
    </span>
  );
}

export interface StatusBadgeFromCodeProps {
  code: string;
  /** Translates a backend status into a dictionary label. */
  labelFor: (code: string) => string;
}

export function StatusBadgeFromCode({
  code,
  labelFor,
}: StatusBadgeFromCodeProps) {
  const { dict } = useI18n();
  const tone: StatusTone =
    code === "succeeded" || code === "active" || code === "delivered" || code === "verified"
      ? "succeeded"
      : code === "pending" || code === "held" || code === "review"
        ? "pending"
        : code === "expired" || code === "canceled" || code === "disabled" || code === "offline"
          ? "muted"
          : code === "failed" || code === "suspect"
            ? "danger"
            : "info";
  return <StatusBadge tone={tone} label={labelFor(code) ?? dict.status.pending} />;
}
