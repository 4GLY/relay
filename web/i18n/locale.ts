import { DEFAULT_LOCALE, RELAY_LOCALE_COOKIE, type Locale } from "./routing";

export function parseCookieValue(
  cookieHeader: string | undefined,
  name: string,
): string | undefined {
  if (!cookieHeader) return undefined;
  for (const segment of cookieHeader.split(";")) {
    const [rawName, ...rawValue] = segment.trim().split("=");
    if (rawName !== name) continue;
    return decodeURIComponent(rawValue.join("="));
  }
  return undefined;
}

export function normalizeLocale(value: string | undefined | null): Locale | null {
  if (!value) return null;
  const candidate = value.trim().toLowerCase();
  if (candidate === "ko" || candidate.startsWith("ko-")) return "ko";
  if (candidate === "en" || candidate.startsWith("en-")) return "en";
  return null;
}

export function parseAcceptLanguage(header: string | undefined | null): string[] {
  if (!header) return [];
  return header
    .split(",")
    .map((part) => {
      const [tag, ...params] = part.trim().split(";");
      const qValue = params.find((param) => param.trim().startsWith("q="));
      const q = qValue ? Number.parseFloat(qValue.trim().slice(2)) : 1;
      return { tag, q: Number.isFinite(q) ? q : 0 };
    })
    .filter((item) => item.tag.length > 0 && item.q > 0)
    .sort((left, right) => right.q - left.q)
    .map((item) => item.tag);
}

export function resolveLocaleFromHeaders(input: {
  cookie?: string;
  acceptLanguage?: string | null;
}): Locale {
  const cookieLocale = normalizeLocale(parseCookieValue(input.cookie, RELAY_LOCALE_COOKIE));
  if (cookieLocale) return cookieLocale;

  for (const candidate of parseAcceptLanguage(input.acceptLanguage)) {
    const locale = normalizeLocale(candidate);
    if (locale) return locale;
  }

  return DEFAULT_LOCALE;
}
