import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import APIKeySettingsPage from "./page";

const mocks = vi.hoisted(() => ({
  cookies: vi.fn(),
  headers: vi.fn(),
  listUserAPIKeys: vi.fn(),
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
