import { describe, expect, it } from "vitest";

import {
  getDictionary,
  getDictionaryKeys,
  RELAY_LOCALE_COOKIE,
  resolveLocale,
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
});

describe("dictionaries", () => {
  it("exposes both english and korean dictionaries", () => {
    expect(getDictionary("en").root.title).toBe("Relay");
    expect(getDictionary("ko").apiKeys.client.title).toBe("Relay API 키");
  });

  it("keeps english and korean key shapes aligned", () => {
    expect(getDictionaryKeys(getDictionary("ko"))).toEqual(getDictionaryKeys(getDictionary("en")));
  });
});
