import { isLang, LANG_COOKIE, type Lang } from "./types";

/** Read the language cookie client-side. */
export function readLangCookie(): Lang {
  if (typeof document === "undefined") return "en";
  const match = document.cookie
    .split("; ")
    .find((row) => row.startsWith(`${LANG_COOKIE}=`));
  if (!match) return "en";
  const value = match.slice(LANG_COOKIE.length + 1);
  return isLang(value) ? value : "en";
}

/** Persist the language cookie for one year, site-wide. */
export function writeLangCookie(lang: Lang): void {
  document.cookie = `${LANG_COOKIE}=${lang}; path=/; max-age=31536000; samesite=lax`;
}
