import enMessages from "../messages/en.json";
import koMessages from "../messages/ko.json";
import {
  DEFAULT_LOCALE,
  RELAY_LOCALE_COOKIE,
  SUPPORTED_LOCALES,
  type Locale,
} from "../i18n/routing";
import { en as legacyEn } from "./i18n/dictionaries/en";
import { ko as legacyKo } from "./i18n/dictionaries/ko";
import type { Dictionary as LegacyDictionary } from "./i18n/types";

export { RELAY_LOCALE_COOKIE, SUPPORTED_LOCALES };
export type { Locale };

type CoreMessages = typeof enMessages;

export type Dictionary = LegacyDictionary &
  CoreMessages & {
    common: LegacyDictionary["common"] & CoreMessages["Common"];
    root: LegacyDictionary["root"] & CoreMessages["Root"];
    errors: CoreMessages["Errors"];
  };

const dictionaries: Record<Locale, Dictionary> = {
  en: {
    ...legacyEn,
    Common: enMessages.Common,
    Root: enMessages.Root,
    Errors: enMessages.Errors,
    common: enMessages.Common,
    root: enMessages.Root,
    errors: enMessages.Errors,
  },
  ko: {
    ...legacyKo,
    Common: koMessages.Common,
    Root: koMessages.Root,
    Errors: koMessages.Errors,
    common: koMessages.Common,
    root: koMessages.Root,
    errors: koMessages.Errors,
  },
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

  return DEFAULT_LOCALE;
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
  const knownErrors: Record<string, string> =
    options.knownErrors ?? getDictionary(options.locale).errors;
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
  return options.fallback || getDictionary(options.locale).Common.unknownError;
}
