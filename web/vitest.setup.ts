import "@testing-library/jest-dom/vitest";
import { afterEach, vi } from "vitest";
import { cleanup } from "@testing-library/react";

vi.mock("next-intl", () => ({
  useLocale: () => "en",
  useTranslations: (namespace: string) => (key: string) => {
    const messages: Record<string, string> = {
      "Common.language.label": "Language",
      "Common.language.apply": "Apply",
      "Common.language.english": "English",
      "Common.language.korean": "Korean",
      "Shell.globalNavigation": "Global navigation",
      "Shell.settings": "Settings",
      "Shell.signedInFallback": "signed in",
    };

    return messages[`${namespace}.${key}`] ?? key;
  },
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
