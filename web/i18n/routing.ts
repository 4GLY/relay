export const RELAY_LOCALE_COOKIE = "relay_locale";
export const SUPPORTED_LOCALES = ["en", "ko"] as const;
export type Locale = (typeof SUPPORTED_LOCALES)[number];
export const DEFAULT_LOCALE: Locale = "en";

export function isLocale(value: string): value is Locale {
  return SUPPORTED_LOCALES.includes(value as Locale);
}
