import { describe, expect, it } from "vitest";

import {
  getDictionary,
  getDictionaryKeys,
  RELAY_LOCALE_COOKIE,
  resolveLocale,
  translateErrorMessage,
} from "./i18n";

describe("resolveLocale", () => {
  it("prefers the relay_locale cookie", () => {
    expect(
      resolveLocale({
        cookie: `${RELAY_LOCALE_COOKIE}=ko`,
        acceptLanguage: "en-US,en;q=0.8",
      }),
    ).toBe("ko");
    expect(
      resolveLocale({
        cookie: `${RELAY_LOCALE_COOKIE}=en`,
        acceptLanguage: "ko-KR,ko;q=0.9",
      }),
    ).toBe("en");
  });

  it("falls back to Accept-Language when the cookie is missing", () => {
    expect(resolveLocale({ acceptLanguage: "ko-KR,ko;q=0.9,en;q=0.8" })).toBe("ko");
    expect(resolveLocale({ acceptLanguage: "en-US,en;q=0.9,ko;q=0.8" })).toBe("en");
  });

  it("ignores unsupported values and falls back to english", () => {
    expect(
      resolveLocale({
        cookie: `${RELAY_LOCALE_COOKIE}=fr`,
        acceptLanguage: "fr-FR,fr;q=0.9",
      }),
    ).toBe("en");
  });

  it("ignores zero-quality Accept-Language values", () => {
    expect(resolveLocale({ acceptLanguage: "fr, en;q=0, ko;q=0" })).toBe("en");
    expect(resolveLocale({ acceptLanguage: "fr, en;q=0, ko;q=0.5" })).toBe("ko");
  });
});

describe("messages", () => {
  it("exposes the initial next-intl message namespaces", () => {
    expect(getDictionary("en").Common.unknownError).toBe("Something went wrong.");
    expect(getDictionary("en").Root.title).toBe("Relay");
    expect(getDictionary("en").Errors.UNAUTHENTICATED).toBe("Sign in again to continue.");
    expect(getDictionary("ko").Common.language.current).toBe("현재 언어: {locale}");
    expect(getDictionary("ko").Root.signInButton).toBe("GitHub로 계속하기");
    expect(getDictionary("ko").Errors.API_KEY_NOT_FOUND_BY_ID).toBe(
      "해당 API 키를 더 이상 찾을 수 없습니다.",
    );
  });

  it("keeps english and korean message shapes aligned", () => {
    const enKeys = getDictionaryKeys(getDictionary("en")).filter((key) =>
      /^(Common|Root|Errors)\./.test(key),
    );
    const koKeys = getDictionaryKeys(getDictionary("ko")).filter((key) =>
      /^(Common|Root|Errors)\./.test(key),
    );

    expect(koKeys).toEqual(enKeys);
  });

  it("keeps ICU-style placeholders aligned across locales", () => {
    const enPlaceholders = getPlaceholdersByKey(getDictionary("en"));
    const koPlaceholders = getPlaceholdersByKey(getDictionary("ko"));

    expect(koPlaceholders).toEqual(enPlaceholders);
    expect(enPlaceholders["Common.language.current"]).toEqual(["locale"]);
  });

  it("keeps lowercase aliases for legacy callers", () => {
    expect(getDictionary("en").common.unknownError).toBe(getDictionary("en").Common.unknownError);
    expect(getDictionary("ko").root.subtitle).toBe(getDictionary("ko").Root.subtitle);
  });

  it("translates known errors from the core message map by default", () => {
    expect(
      translateErrorMessage({
        error: { code: "INVALID_INPUT" },
        fallback: "",
        locale: "ko",
      }),
    ).toBe("입력값을 확인한 뒤 다시 시도하세요.");
  });
});

function getPlaceholdersByKey(dictionary: ReturnType<typeof getDictionary>) {
  const placeholdersByKey: Record<string, string[]> = {};

  function walk(value: unknown, prefix: string) {
    if (typeof value === "string") {
      const placeholders = [...value.matchAll(/\{([A-Za-z_][A-Za-z0-9_]*)\}/g)].map(
        (match) => match[1],
      );
      if (placeholders.length > 0) {
        placeholdersByKey[prefix] = [...new Set(placeholders)].sort();
      }
      return;
    }

    if (!value || typeof value !== "object" || Array.isArray(value)) return;

    for (const [key, child] of Object.entries(value)) {
      walk(child, prefix ? `${prefix}.${key}` : key);
    }
  }

  walk(dictionary, "");
  return placeholdersByKey;
}
