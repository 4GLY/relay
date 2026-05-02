import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import APIKeySettingsPage from "./page";

const mocks = vi.hoisted(() => ({
  cookies: vi.fn(),
  headers: vi.fn(),
  listUserAPIKeys: vi.fn(),
  locale: "en",
  RelayAPIError: class RelayAPIError extends Error {
    code: string;

    constructor(code: string, message: string) {
      super(message);
      this.code = code;
    }
  },
}));

vi.mock("next/headers", () => ({
  cookies: mocks.cookies,
  headers: mocks.headers,
}));

vi.mock("@/lib/api", () => ({
  RELAY_API_URL: "https://relay.4gly.dev",
}));

vi.mock("@/lib/user-api-keys", () => ({
  RelayAPIError: mocks.RelayAPIError,
  listUserAPIKeys: mocks.listUserAPIKeys,
}));
vi.mock("next-intl/server", () => ({
  getTranslations: async (namespace: string) => (key: string) => {
    const messages: Record<string, string> = {
      "Settings.ApiKeys.page.eyebrow":
        mocks.locale === "ko" ? "설정 · Relay API 키" : "Settings · Relay API keys",
      "Settings.ApiKeys.page.signInTitle":
        mocks.locale === "ko" ? "먼저 로그인하세요" : "Sign in first",
      "Settings.ApiKeys.page.signInCopy":
        mocks.locale === "ko"
          ? "Relay API 키는 API와 MCP 접근을 위한 사용자별 설정입니다."
          : "Relay API keys are user-owned settings for API and MCP access.",
      "Settings.ApiKeys.page.loadErrorTitle":
        mocks.locale === "ko" ? "API 키를 불러오지 못했습니다" : "Could not load API keys",
      "Settings.ApiKeys.page.loadErrorCopy":
        mocks.locale === "ko"
          ? "Relay API가 다시 응답하면 이 페이지를 새로고침해 보세요."
          : "Try refreshing this page after the Relay API is reachable again.",
      "Settings.ApiKeys.errorMap.UNAUTHENTICATED":
        mocks.locale === "ko"
          ? "Relay API 키를 관리하려면 다시 로그인하세요."
          : "Sign in again to manage Relay API keys.",
      "Settings.ApiKeys.errorMap.INVALID_INPUT":
        mocks.locale === "ko"
          ? "키를 발급하기 전에 이름을 입력하세요."
          : "Enter a name before issuing a key.",
      "Settings.ApiKeys.errorMap.API_KEY_NOT_FOUND_BY_ID":
        mocks.locale === "ko"
          ? "해당 API 키를 더 이상 찾을 수 없습니다."
          : "That API key no longer exists.",
      "Common.continueWithGitHub": mocks.locale === "ko" ? "GitHub로 계속하기" : "Continue with GitHub",
      "Common.links.projectExplorer": "Project Explorer",
      "Common.links.providerSettings": mocks.locale === "ko" ? "Provider 설정" : "Provider settings",
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

describe("<APIKeySettingsPage>", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.locale = "en";
    mocks.cookies.mockResolvedValue(cookieStore());
    mocks.headers.mockResolvedValue(headerStore());
  });

  it("renders the authenticated API key settings client", async () => {
    mocks.listUserAPIKeys.mockResolvedValueOnce({
      items: [
        {
          key_id: "key_1",
          name: "CLI",
          token_prefix: "relay_live_abc",
          scope: "global",
          revoked: false,
        },
      ],
    });

    render(await APIKeySettingsPage());

    expect(screen.getByRole("heading", { name: "Relay API keys" })).toBeVisible();
    expect(screen.getByText("Issued keys")).toBeVisible();
    expect(screen.getByText("CLI")).toBeVisible();
  });

  it("renders a korean sign-in panel when locale resolves to ko", async () => {
    mocks.headers.mockResolvedValue(headerStore("ko-KR,ko;q=0.9,en;q=0.8"));
    mocks.locale = "ko";
    mocks.listUserAPIKeys.mockRejectedValueOnce(
      new mocks.RelayAPIError("UNAUTHENTICATED", "missing session"),
    );

    render(await APIKeySettingsPage());

    expect(screen.getByRole("heading", { name: "먼저 로그인하세요" })).toBeVisible();
    expect(screen.getByText("Relay API 키는 API와 MCP 접근을 위한 사용자별 설정입니다.")).toBeVisible();
    expect(screen.getByRole("link", { name: "GitHub로 계속하기" })).toHaveAttribute(
      "href",
      "https://relay.4gly.dev/v1/auth/github/start?redirect_to=%2Fsettings%2Fapi-keys",
    );
  });
});
