export type Lang = "en" | "bn";

export const LANG_COOKIE = "rexio_lang";

export const LANGS: Lang[] = ["en", "bn"];

export function isLang(value: unknown): value is Lang {
  return value === "en" || value === "bn";
}
