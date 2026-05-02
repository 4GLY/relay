import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import OnboardingPage from "./page";

const mocks = vi.hoisted(() => ({
  redirect: vi.fn((path: string) => {
    throw new Error(`NEXT_REDIRECT:${path}`);
  }),
  cookies: vi.fn(),
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
vi.mock("next-intl/server", () => ({
  getTranslations: async (namespace: string) => (key: string) => {
    const messages: Record<string, string> = {
      "Onboarding.page.eyebrow": mocks.locale === "ko" ? "Slice 8 · 60초" : "Slice 8 · 60 seconds",
      "Onboarding.page.title": mocks.locale === "ko" ? "키 없이 시작하는 첫 실행" : "First run, no keys",
      "Onboarding.page.subtitle":
        mocks.locale === "ko"
          ? "Relay는 먼저 개인 작업공간을 만듭니다. provider 키는 첫 1분의 진입 조건이 아니라 Settings 항목입니다."
          : "Relay starts by creating a private workspace. Provider keys are a Settings concern, not a gate on the first minute.",
      "Onboarding.page.signInTitle":
        mocks.locale === "ko" ? "작업공간을 만들려면 먼저 로그인하세요" : "Sign in to create your workspace",
      "Onboarding.page.signInCopy":
        mocks.locale === "ko"
          ? "먼저 GitHub로 로그인하세요. 여기로 돌아오면 Relay가 Personal 프로젝트를 생성합니다."
          : "Use an identity provider first. Relay will create your Personal project after you return here.",
      "Common.continueWithGitHub": mocks.locale === "ko" ? "GitHub로 계속하기" : "Continue with GitHub",
    };
    return messages[`${namespace}.${key}`] ?? key;
  },
}));

function cookieStore(value = "relay_session=test") {
  return {
    toString: () => value,
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
  });

  it("renders localized sign-in copy for korean browsers", async () => {
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
    expect(screen.getAllByRole("button", { name: "Apply" })).toHaveLength(2);
  });
});
