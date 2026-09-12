"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { bn } from "./bn";
import { readLangCookie, writeLangCookie } from "./cookie";
import { en, type Dict } from "./en";
import type { Lang } from "./types";

interface I18nContextValue {
  lang: Lang;
  dict: Dict;
  setLang: (lang: Lang) => void;
}

const I18nContext = createContext<I18nContextValue | null>(null);

export function I18nProvider({
  initialLang,
  children,
}: {
  initialLang: Lang;
  children: ReactNode;
}) {
  const [lang, setLangState] = useState<Lang>(initialLang);

  const dict = useMemo(() => (lang === "bn" ? bn : en), [lang]);

  const setLang = useCallback((next: Lang) => {
    setLangState(next);
    writeLangCookie(next);
    document.documentElement.lang = next;
  }, []);

  // The cookie may have changed between SSR and hydration; reconcile once.
  useEffect(() => {
    const cookieLang = readLangCookie();
    if (cookieLang !== initialLang) {
      setLangState(cookieLang);
      document.documentElement.lang = cookieLang;
    }
  }, [initialLang]);

  const value = useMemo(() => ({ lang, dict, setLang }), [lang, dict, setLang]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nContextValue {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used inside I18nProvider");
  return ctx;
}
