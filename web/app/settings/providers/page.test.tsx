import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import ProviderSettingsPage from "./page";

const mocks = vi.hoisted(() => ({
  cookies: vi.fn(),
  headers: vi.fn(),
  listProviderCredentials: vi.fn(),
  locale: "en",
}));

vi.mock("next/headers", () => ({
  cookies: mocks.cookies,
  headers: mocks.headers,
}));

vi.mock("@/lib/api", () => ({
  RELAY_API_URL: "https://relay.4gly.dev",
}));

vi.mock("@/lib/provider-credentials", () => ({
  listProviderCredentials: mocks.listProviderCredentials,
}));
vi.mock("next-intl/server", () => ({
  getTranslations: async (namespace: string) => (key: string) => {
    const messages: Record<string, string> = {
      "Settings.ProviderCredentials.page.eyebrow":
        mocks.locale === "ko" ? "설정 · provider 인증 정보" : "Settings · provider credentials",
      "Settings.ProviderCredentials.page.signInTitle":
        mocks.locale === "ko" ? "먼저 로그인하세요" : "Sign in first",
      "Settings.ProviderCredentials.page.signInCopy":
        mocks.locale === "ko"
          ? "Provider 인증 정보는 사용자별 설정입니다."
          : "Provider credentials are user-owned settings.",
      "Common.continueWithGitHub": mocks.locale === "ko" ? "GitHub로 계속하기" : "Continue with GitHub",
      "Common.links.backToOnboarding": mocks.locale === "ko" ? "온보딩으로 돌아가기" : "Back to onboarding",
      "Common.links.apiKeys": mocks.locale === "ko" ? "API 키 설정" : "API key settings",
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

describe("<ProviderSettingsPage>", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.locale = "en";
    mocks.cookies.mockResolvedValue(cookieStore());
    mocks.headers.mockResolvedValue(headerStore());
  });

  it("renders korean provider sign-in copy when locale resolves to ko", async () => {
    mocks.headers.mockResolvedValueOnce(headerStore("ko-KR,ko;q=0.9,en;q=0.8"));
    mocks.locale = "ko";
    mocks.listProviderCredentials.mockRejectedValueOnce(new Error("missing session"));

    render(await ProviderSettingsPage());

    expect(screen.getByRole("heading", { name: "먼저 로그인하세요" })).toBeVisible();
    expect(screen.getByText("Provider 인증 정보는 사용자별 설정입니다.")).toBeVisible();
    expect(screen.getByRole("link", { name: "GitHub로 계속하기" })).toHaveAttribute(
      "href",
      "https://relay.4gly.dev/v1/auth/github/start?redirect_to=%2Fsettings%2Fproviders",
    );
  });
});
