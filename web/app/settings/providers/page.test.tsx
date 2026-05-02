import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import ProviderSettingsPage from "./page";

const mocks = vi.hoisted(() => ({
  cookies: vi.fn(),
  listProviderCredentials: vi.fn(),
  locale: "en",
  ProviderCredentialAPIError: class ProviderCredentialAPIError extends Error {
    code: string;
    retryable: boolean;

    constructor(payload: { code: string; message: string; retryable: boolean }) {
      super(payload.message || payload.code);
      this.name = "ProviderCredentialAPIError";
      this.code = payload.code;
      this.retryable = payload.retryable;
    }
  },
}));

vi.mock("next/headers", () => ({
  cookies: mocks.cookies,
}));

vi.mock("@/lib/api", () => ({
  RELAY_API_URL: "https://relay.4gly.dev",
}));

vi.mock("@/lib/provider-credentials", () => ({
  ProviderCredentialAPIError: mocks.ProviderCredentialAPIError,
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
      "Settings.ProviderCredentials.page.loadErrorTitle":
        mocks.locale === "ko"
          ? "provider 설정을 불러오지 못했습니다"
          : "Could not load provider settings",
      "Settings.ProviderCredentials.page.loadErrorCopy":
        mocks.locale === "ko"
          ? "Relay API가 다시 응답하면 이 페이지를 새로고침해 보세요."
          : "Try refreshing this page after the Relay API is reachable again.",
      "Settings.ProviderCredentials.errorMap.UNAUTHENTICATED":
        mocks.locale === "ko"
          ? "provider 인증 정보를 관리하려면 다시 로그인하세요."
          : "Sign in again to manage provider credentials.",
      "Settings.ProviderCredentials.errorMap.INVALID_INPUT":
        mocks.locale === "ko"
          ? "Anthropic 키는 sk-ant- 로 시작해야 합니다."
          : "Anthropic keys must start with sk-ant-",
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

describe("<ProviderSettingsPage>", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.locale = "en";
    mocks.cookies.mockResolvedValue(cookieStore());
  });

  it("renders korean provider sign-in copy when locale resolves to ko", async () => {
    mocks.locale = "ko";
    mocks.listProviderCredentials.mockRejectedValueOnce(
      new mocks.ProviderCredentialAPIError({
        code: "UNAUTHENTICATED",
        message: "missing session",
        retryable: false,
      }),
    );

    render(await ProviderSettingsPage());

    expect(screen.getByRole("heading", { name: "먼저 로그인하세요" })).toBeVisible();
    expect(screen.getByText("Provider 인증 정보는 사용자별 설정입니다.")).toBeVisible();
    expect(screen.getByRole("link", { name: "GitHub로 계속하기" })).toHaveAttribute(
      "href",
      "https://relay.4gly.dev/v1/auth/github/start?redirect_to=%2Fsettings%2Fproviders",
    );
  });

  it("renders the localized load-error state for non-auth failures", async () => {
    mocks.listProviderCredentials.mockRejectedValueOnce(
      new mocks.ProviderCredentialAPIError({
        code: "UPSTREAM_UNAVAILABLE",
        message: "raw upstream timeout",
        retryable: true,
      }),
    );

    render(await ProviderSettingsPage());

    expect(screen.getByRole("heading", { name: "Could not load provider settings" })).toBeVisible();
    expect(
      screen.getByText("Try refreshing this page after the Relay API is reachable again."),
    ).toBeVisible();
    expect(screen.queryByText("raw upstream timeout")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Continue with GitHub" })).not.toBeInTheDocument();
  });
});
