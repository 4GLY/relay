import { en } from "./i18n/dictionaries/en";
import { ko } from "./i18n/dictionaries/ko";
import type { Dictionary, Locale } from "./i18n/types";

export type { Dictionary, Locale } from "./i18n/types";

export const RELAY_LOCALE_COOKIE = "relay_locale";
export const SUPPORTED_LOCALES: Locale[] = ["en", "ko"];

const dictionaries: Record<Locale, Dictionary> = {
  en,
  ko,
};

function parseCookieValue(cookieHeader: string | undefined, name: string): string | undefined {
  if (!cookieHeader) return undefined;
  for (const segment of cookieHeader.split(";")) {
    const [rawName, ...rawValue] = segment.trim().split("=");
    if (rawName !== name) continue;
    return decodeURIComponent(rawValue.join("="));
  }
  return undefined;
}

function normalizeLocale(value: string | undefined | null): Locale | null {
  if (!value) return null;
  const candidate = value.trim().toLowerCase();
  if (candidate === "ko" || candidate.startsWith("ko-")) return "ko";
  if (candidate === "en" || candidate.startsWith("en-")) return "en";
  return null;
}

function parseAcceptLanguage(header: string | undefined): string[] {
  if (!header) return [];
  return header
    .split(",")
    .map((part) => {
      const [tag, ...params] = part.trim().split(";");
      const qValue = params.find((param) => param.trim().startsWith("q="));
      const q = qValue ? Number.parseFloat(qValue.trim().slice(2)) : 1;
      return { tag, q: Number.isFinite(q) ? q : 0 };
    })
    .filter((item) => item.tag.length > 0)
    .sort((left, right) => right.q - left.q)
    .map((item) => item.tag);
}

export function resolveLocale(input: {
  cookie?: string;
  acceptLanguage?: string;
}): Locale {
  const cookieLocale = normalizeLocale(parseCookieValue(input.cookie, RELAY_LOCALE_COOKIE));
  if (cookieLocale) return cookieLocale;

  for (const candidate of parseAcceptLanguage(input.acceptLanguage)) {
    const locale = normalizeLocale(candidate);
    if (locale) return locale;
  }

  return "en";
}

export function getDictionary(locale: Locale): Dictionary {
  return dictionaries[locale];
}

export function getDictionaryKeys(dictionary: Dictionary): string[] {
  const keys: string[] = [];

  function walk(value: unknown, prefix: string) {
    if (!value || typeof value !== "object" || Array.isArray(value)) {
      keys.push(prefix);
      return;
    }

    for (const [key, child] of Object.entries(value)) {
      walk(child, prefix ? `${prefix}.${key}` : key);
    }
  }

  walk(dictionary, "");
  return keys.sort();
}

export function translateErrorMessage(options: {
  error: unknown;
  fallback: string;
  locale: Locale;
  knownErrors?: Record<string, string>;
}): string {
  const knownErrors = options.knownErrors ?? {};
  const message =
    options.error instanceof Error
      ? options.error.message
      : typeof options.error === "string"
        ? options.error
        : "";
  const code =
    options.error &&
    typeof options.error === "object" &&
    "code" in options.error &&
    typeof options.error.code === "string"
      ? options.error.code
      : undefined;

  if (code && knownErrors[code]) return knownErrors[code];
  if (message && knownErrors[message]) return knownErrors[message];
  if (message) return message;
  return options.fallback || getDictionary(options.locale).common.unknownError;
}
