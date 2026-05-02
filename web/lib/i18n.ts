import enMessages from "../messages/en.json";
import koMessages from "../messages/ko.json";
import {
  RELAY_LOCALE_COOKIE,
  SUPPORTED_LOCALES,
  type Locale,
} from "../i18n/routing";
import { resolveLocaleFromHeaders } from "../i18n/locale";

export { RELAY_LOCALE_COOKIE, SUPPORTED_LOCALES };
export type { Locale };

type Messages = typeof enMessages;

export type Dictionary = Messages & {
  common: Messages["Common"];
  root: Messages["Root"];
  errors: Record<string, string>;
  onboarding: Messages["Onboarding"];
  settings: Messages["Settings"];
};

function createDictionary(messages: Messages): Dictionary {
  return {
    ...messages,
    common: messages.Common,
    root: messages.Root,
    errors: messages.Errors,
    onboarding: messages.Onboarding,
    settings: messages.Settings,
  };
}

const dictionaries: Record<Locale, Dictionary> = {
  en: createDictionary(enMessages),
  ko: createDictionary(koMessages),
};

export function resolveLocale(input: {
  cookie?: string;
  acceptLanguage?: string;
}): Locale {
  return resolveLocaleFromHeaders(input);
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

export function translateKnownError(options: {
  error: unknown;
  fallback: string;
  knownErrors: Record<string, string>;
}): string {
  const code =
    options.error &&
    typeof options.error === "object" &&
    "code" in options.error &&
    typeof options.error.code === "string"
      ? options.error.code
      : undefined;
  const message =
    options.error instanceof Error
      ? options.error.message
      : typeof options.error === "string"
        ? options.error
        : "";

  if (code && options.knownErrors[code]) return options.knownErrors[code];
  if (message && options.knownErrors[message]) return options.knownErrors[message];
  return options.fallback;
}
