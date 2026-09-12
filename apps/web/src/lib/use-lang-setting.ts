"use client";

import { useI18n } from "@/lib/i18n";

/**
 * Thin wrapper exposing setLang without the full context value, for
 * settings pages that only need the setter.
 */
export function useI18nSetter() {
  const { setLang } = useI18n();
  return { setLang };
}
