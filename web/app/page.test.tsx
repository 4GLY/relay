import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import HomePage from "./page";

const mocks = vi.hoisted(() => ({
  redirect: vi.fn((path: string) => {
    throw new Error(`NEXT_REDIRECT:${path}`);
  }),
  cookies: vi.fn(),
  headers: vi.fn(),
  relayFetch: vi.fn(),
  locale: "en",
  pathname: "/",
  search: "",
}));

vi.mock("next/navigation", () => ({
  redirect: mocks.redirect,
  usePathname: () => mocks.pathname,
  useSearchParams: () => new URLSearchParams(mocks.search),
}));
vi.mock("next/headers", () => ({ cookies: mocks.cookies, headers: mocks.headers }));
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
      "Root.eyebrow": "4gly Labs · Relay",
      "Root.title": "Relay",
      "Root.subtitle":
        mocks.locale === "ko"
          ? "흩어진 작업을 조용히 백조처럼 정리하는 엔진."
          : "A quiet engine that turns chaos into swans.",
      "Root.panelTitle": mocks.locale === "ko" ? "시작하려면 로그인하세요" : "Sign in to start",
      "Root.panelCopy":
        mocks.locale === "ko"
          ? "Relay는 먼저 개인 작업공간을 만듭니다. provider 키는 첫 실행 설정이 아니라 Settings에서 관리합니다."
          : "Relay creates a private workspace first. Provider keys stay in Settings, not in first-run setup.",
      "Root.signInButton": mocks.locale === "ko" ? "GitHub로 계속하기" : "Continue with GitHub",
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

describe("<HomePage>", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.locale = "en";
    mocks.pathname = "/";
    mocks.search = "";
    mocks.cookies.mockResolvedValue(cookieStore());
    mocks.headers.mockResolvedValue(headerStore());
  });

  it("shows the GitHub sign-in entry when there is no session", async () => {
    mocks.relayFetch.mockResolvedValueOnce(
      authResponse(401, {
        ok: false,
        command: "relay auth me",
        error: { code: "UNAUTHENTICATED", message: "missing session", retryable: false },
      }),
    );

    render(await HomePage());

    expect(screen.getByRole("heading", { name: "Relay" })).toBeVisible();
    expect(screen.getByText("A quiet engine that turns chaos into swans.")).toBeVisible();
    expect(screen.getByRole("link", { name: "Continue with GitHub" })).toHaveAttribute(
      "href",
      "https://relay.4gly.dev/v1/auth/github/start?redirect_to=%2Fonboarding",
    );
    expect(screen.getByLabelText("Language")).toHaveValue("en");
    expect(screen.getByDisplayValue("English").closest("form")).toHaveAttribute(
      "action",
      "/settings/language",
    );
    expect(screen.getByRole("button", { name: "Apply" })).toBeVisible();
    expect(screen.queryByText("Sharable Snapshot")).not.toBeInTheDocument();
  });

  it("preserves the current path and query when switching language", async () => {
    mocks.pathname = "/current";
    mocks.search = "tab=files";
    mocks.relayFetch.mockResolvedValueOnce(
      authResponse(401, {
        ok: false,
        command: "relay auth me",
        error: { code: "UNAUTHENTICATED", message: "missing session", retryable: false },
      }),
    );

    const { container } = render(await HomePage());

    expect(container.querySelector('input[name="redirectTo"]')).toHaveAttribute(
      "value",
      "/current?tab=files",
    );
  });

  it("redirects authenticated users who still need onboarding to /onboarding", async () => {
    mocks.relayFetch.mockResolvedValueOnce(
      authResponse(200, {
        ok: true,
        command: "relay auth me",
        data: {
          user_id: "user_1",
          display_name: "Hoon",
          onboarding_complete: false,
        },
        warnings: [],
      }),
    );

    await expect(HomePage()).rejects.toThrow("NEXT_REDIRECT:/onboarding");
    expect(mocks.redirect).toHaveBeenCalledWith("/onboarding");
  });

  it("redirects fully onboarded users to their Project Explorer", async () => {
    mocks.relayFetch.mockResolvedValueOnce(
      authResponse(200, {
        ok: true,
        command: "relay auth me",
        data: {
          user_id: "user_1",
          display_name: "Hoon",
          onboarding_complete: true,
          default_project_id: "proj_personal",
        },
        warnings: [],
      }),
    );

    await expect(HomePage()).rejects.toThrow("NEXT_REDIRECT:/projects/proj_personal");
    expect(mocks.redirect).toHaveBeenCalledWith("/projects/proj_personal");
  });

  it("renders korean root copy when locale resolves to ko", async () => {
    mocks.headers.mockResolvedValueOnce(headerStore("ko-KR,ko;q=0.9,en;q=0.8"));
    mocks.locale = "ko";
    mocks.relayFetch.mockResolvedValueOnce(
      authResponse(401, {
        ok: false,
        command: "relay auth me",
        error: { code: "UNAUTHENTICATED", message: "missing session", retryable: false },
      }),
    );

    render(await HomePage());

    expect(screen.getByText("흩어진 작업을 조용히 백조처럼 정리하는 엔진.")).toBeVisible();
    expect(screen.getByRole("link", { name: "GitHub로 계속하기" })).toBeVisible();
    expect(screen.getByLabelText("Language")).toHaveValue("ko");
  });
});
