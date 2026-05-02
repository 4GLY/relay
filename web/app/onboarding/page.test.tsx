import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import OnboardingPage from "./page";

const mocks = vi.hoisted(() => ({
  redirect: vi.fn((path: string) => {
    throw new Error(`NEXT_REDIRECT:${path}`);
  }),
  cookies: vi.fn(),
  headers: vi.fn(),
  relayFetch: vi.fn(),
  locale: "en",
  pathname: "/onboarding",
  search: "",
}));

vi.mock("next/navigation", () => ({
  redirect: mocks.redirect,
  usePathname: () => mocks.pathname,
  useSearchParams: () => new URLSearchParams(mocks.search),
}));
vi.mock("next/headers", () => ({
  cookies: mocks.cookies,
  headers: mocks.headers,
}));
vi.mock("@/lib/api", () => ({
  RELAY_API_URL: "https://relay.4gly.dev",
  relayFetch: mocks.relayFetch,
}));
vi.mock("next-intl", () => ({
  useLocale: () => mocks.locale,
  useTranslations: (namespace: string) => (key: string) => {
    const messages: Record<string, string> = {
      "Common.language.label": "Language",
      "Common.language.english": "English",
      "Common.language.korean": "Korean",
      "Shell.globalNavigation": "Global navigation",
      "Shell.settings": "Settings",
      "Shell.signedInFallback": "signed in",
    };
    return messages[`${namespace}.${key}`] ?? key;
  },
}));

function cookieStore(value = "relay_session=test") {
  return {
    toString: () => value,
  };
}

function headerStore(acceptLanguage = "en-US,en;q=0.9") {
  return {
    get: (name: string) => (name.toLowerCase() === "accept-language" ? acceptLanguage : null),
  };
}

function authResponse(status: number, body: unknown, ok = status >= 200 && status < 300) {
  return {
    status,
    ok,
    json: async () => body,
  } as Response;
}

describe("<OnboardingPage>", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.locale = "en";
    mocks.pathname = "/onboarding";
    mocks.search = "";
    mocks.cookies.mockResolvedValue(cookieStore());
    mocks.headers.mockResolvedValue(headerStore());
  });

  it("renders localized sign-in copy for korean browsers", async () => {
    mocks.headers.mockResolvedValueOnce(headerStore("ko-KR,ko;q=0.9,en;q=0.8"));
    mocks.locale = "ko";
    mocks.relayFetch.mockResolvedValueOnce(
      authResponse(401, {
        ok: false,
        command: "relay auth me",
        error: { code: "UNAUTHENTICATED", message: "missing session", retryable: false },
      }),
    );

    render(await OnboardingPage());

    expect(screen.getByRole("heading", { name: "키 없이 시작하는 첫 실행" })).toBeVisible();
    expect(
      screen.getByText(
        "먼저 GitHub로 로그인하세요. 여기로 돌아오면 Relay가 Personal 프로젝트를 생성합니다.",
      ),
    ).toBeVisible();
    expect(screen.getByRole("link", { name: "GitHub로 계속하기" })).toBeVisible();
    expect(screen.getAllByLabelText("Language")).toHaveLength(2);
    expect(screen.getAllByDisplayValue("Korean")).toHaveLength(2);
  });
});
