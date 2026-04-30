import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import ProviderSettingsPage from "./page";

const mocks = vi.hoisted(() => ({
  cookies: vi.fn(),
  headers: vi.fn(),
  listProviderCredentials: vi.fn(),
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
    mocks.cookies.mockResolvedValue(cookieStore());
    mocks.headers.mockResolvedValue(headerStore());
  });

  it("renders korean provider sign-in copy when locale resolves to ko", async () => {
    mocks.headers.mockResolvedValueOnce(headerStore("ko-KR,ko;q=0.9,en;q=0.8"));
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
