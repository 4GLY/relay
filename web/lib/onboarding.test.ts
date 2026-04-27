import { afterEach, describe, expect, it, vi } from "vitest";

import { completeOnboarding } from "./onboarding";

afterEach(() => {
  vi.restoreAllMocks();
});

describe("completeOnboarding", () => {
  it("posts an empty object to the keyless onboarding endpoint", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          ok: true,
          command: "relay onboarding complete",
          data: { onboarding_complete: true, default_project_id: "proj_1" },
          warnings: [],
        }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
    );

    const result = await completeOnboarding();

    expect(result.default_project_id).toBe("proj_1");
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/v1/onboarding",
      expect.objectContaining({
        method: "POST",
        credentials: "include",
        body: "{}",
      }),
    );
  });

  it("uses the public Relay API URL baked into the browser bundle", async () => {
    vi.stubEnv("NEXT_PUBLIC_RELAY_API_URL", "https://relay.4gly.dev");
    vi.resetModules();
    const { completeOnboarding: completeWithPublicURL } = await import("./onboarding");
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          ok: true,
          command: "relay onboarding complete",
          data: { onboarding_complete: true, default_project_id: "proj_1" },
          warnings: [],
        }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
    );

    await completeWithPublicURL();

    expect(fetchMock).toHaveBeenCalledWith(
      "https://relay.4gly.dev/v1/onboarding",
      expect.objectContaining({
        method: "POST",
        credentials: "include",
      }),
    );
  });
});
