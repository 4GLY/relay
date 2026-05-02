import "@testing-library/jest-dom/vitest";
import { afterEach, vi } from "vitest";
import { cleanup } from "@testing-library/react";

import enMessages from "./messages/en.json";
import koMessages from "./messages/ko.json";

declare global {
  // Provided for tests that need to exercise localized rendering.
  // eslint-disable-next-line no-var
  var __setNextIntlLocale: (locale: "en" | "ko") => void;
}

let testLocale: "en" | "ko" = "en";

function readMessage(namespace: string, key: string, values?: Record<string, unknown>): string {
  const path = `${namespace}.${key}`.split(".");
  let cursor: unknown = testLocale === "ko" ? koMessages : enMessages;

  for (const segment of path) {
    if (!cursor || typeof cursor !== "object" || !(segment in cursor)) return key;
    cursor = (cursor as Record<string, unknown>)[segment];
  }

  if (typeof cursor !== "string") return key;
  return cursor.replace(/\{([A-Za-z_][A-Za-z0-9_]*)\}/g, (_match, name: string) =>
    values && name in values ? String(values[name]) : `{${name}}`,
  );
}

vi.mock("next-intl", () => ({
  useLocale: () => testLocale,
  useTranslations: (namespace: string) => (key: string, values?: Record<string, unknown>) =>
    readMessage(namespace, key, values),
}));

vi.mock("next-intl/server", () => ({
  getLocale: async () => testLocale,
  getTranslations: async (namespace: string) => (key: string, values?: Record<string, unknown>) =>
    readMessage(namespace, key, values),
}));

vi.mock("next/navigation", () => ({
  redirect: vi.fn((path: string) => {
    throw new Error(`NEXT_REDIRECT:${path}`);
  }),
  usePathname: () => "/",
  useSearchParams: () => new URLSearchParams(),
}));

afterEach(() => {
  testLocale = "en";
  cleanup();
});

Object.assign(globalThis, {
  __setNextIntlLocale: (locale: "en" | "ko") => {
    testLocale = locale;
  },
});

Object.defineProperty(window, "scrollTo", {
  value: vi.fn(),
  writable: true,
});
