import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import HomePage from "./page";

const mocks = vi.hoisted(() => ({
  redirect: vi.fn((path: string) => {
    throw new Error(`NEXT_REDIRECT:${path}`);
  }),
  cookies: vi.fn(),
  relayFetch: vi.fn(),
}));

vi.mock("next/navigation", () => ({ redirect: mocks.redirect }));
vi.mock("next/headers", () => ({ cookies: mocks.cookies }));
vi.mock("@/lib/api", () => ({
  RELAY_API_URL: "https://relay.4gly.dev",
  relayFetch: mocks.relayFetch,
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

describe("<HomePage>", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.cookies.mockResolvedValue(cookieStore());
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
    expect(screen.queryByText("Sharable Snapshot")).not.toBeInTheDocument();
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

  it("redirects fully onboarded users to their Style Memory project", async () => {
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

    await expect(HomePage()).rejects.toThrow(
      "NEXT_REDIRECT:/style-memory?project=proj_personal",
    );
    expect(mocks.redirect).toHaveBeenCalledWith("/style-memory?project=proj_personal");
  });
});
