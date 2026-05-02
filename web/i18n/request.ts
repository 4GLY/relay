import { getRequestConfig } from "next-intl/server";
import { cookies, headers } from "next/headers";

import { DEFAULT_LOCALE, RELAY_LOCALE_COOKIE, type Locale } from "./routing";

function normalizeLocale(value: string | undefined | null): Locale | null {
  if (!value) return null;
  const candidate = value.trim().toLowerCase();
  if (candidate === "ko" || candidate.startsWith("ko-")) return "ko";
  if (candidate === "en" || candidate.startsWith("en-")) return "en";
  return null;
}

function parseAcceptLanguage(header: string | undefined | null): string[] {
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

export async function resolveRequestLocale(): Promise<Locale> {
  const cookieStore = await cookies();
  const cookieLocale = normalizeLocale(cookieStore.get(RELAY_LOCALE_COOKIE)?.value);
  if (cookieLocale) return cookieLocale;

  const headerStore = await headers();
  for (const candidate of parseAcceptLanguage(headerStore.get("accept-language"))) {
    const locale = normalizeLocale(candidate);
    if (locale) return locale;
  }

  return DEFAULT_LOCALE;
}

export default getRequestConfig(async () => {
  const locale = await resolveRequestLocale();
  return {
    locale,
    messages: (await import(`../messages/${locale}.json`)).default,
  };
});
