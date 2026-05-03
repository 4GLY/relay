import { getRequestConfig } from "next-intl/server";
import { cookies, headers } from "next/headers";

import { normalizeLocale, parseAcceptLanguage } from "./locale";
import { DEFAULT_LOCALE, RELAY_LOCALE_COOKIE, type Locale } from "./routing";

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
