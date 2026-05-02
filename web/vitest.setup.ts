import "@testing-library/jest-dom/vitest";
import { afterEach, vi } from "vitest";
import { cleanup } from "@testing-library/react";

import enMessages from "./messages/en.json";

function readMessage(namespace: string, key: string, values?: Record<string, unknown>): string {
  const path = `${namespace}.${key}`.split(".");
  let cursor: unknown = enMessages;

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
  useLocale: () => "en",
  useTranslations: (namespace: string) => (key: string, values?: Record<string, unknown>) =>
    readMessage(namespace, key, values),
}));

vi.mock("next-intl/server", () => ({
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
  cleanup();
});
