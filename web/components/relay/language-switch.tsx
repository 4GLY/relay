"use client";

import { useId } from "react";
import { useLocale, useTranslations } from "next-intl";
import { usePathname, useSearchParams } from "next/navigation";

import { SUPPORTED_LOCALES, type Locale } from "@/i18n/routing";

export function RelayLanguageSwitch() {
  const locale = useLocale() as Locale;
  const t = useTranslations("Common.language");
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const selectId = useId();
  const search = searchParams.toString();
  const redirectTo = search ? `${pathname}?${search}` : pathname;

  return (
    <form className="relay-language-switch" action="/settings/language" method="post">
      <label className="relay-sr-only" htmlFor={selectId}>
        {t("label")}
      </label>
      <input type="hidden" name="redirectTo" value={redirectTo} />
      <select
        id={selectId}
        name="locale"
        aria-label={t("label")}
        defaultValue={locale}
        onChange={(event) => event.currentTarget.form?.requestSubmit()}
      >
        {SUPPORTED_LOCALES.map((candidate) => (
          <option key={candidate} value={candidate}>
            {candidate === "en" ? t("english") : t("korean")}
          </option>
        ))}
      </select>
      <button className="relay-sr-only" type="submit">
        {t("label")}
      </button>
    </form>
  );
}
