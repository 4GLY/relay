import { afterEach, describe, expect, it, vi } from "vitest";

import {
  connectProviderCredential,
  disconnectProviderCredential,
  listProviderCredentials,
} from "./provider-credentials";

afterEach(() => {
  vi.restoreAllMocks();
});

describe("provider credential API client", () => {
  it("lists provider credentials", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          ok: true,
          command: "relay provider credentials list",
          data: { credentials: [{ provider: "anthropic", connected: true }] },
          warnings: [],
        }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
    );

    const result = await listProviderCredentials({ cookie: "relay_session=x" });

    expect(result.credentials).toHaveLength(1);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/v1/settings/provider-credentials",
      expect.objectContaining({
        method: "GET",
        credentials: "include",
        headers: { cookie: "relay_session=x" },
      }),
    );
  });

  it("connects and disconnects the Anthropic credential", async () => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            ok: true,
            command: "relay provider credential upsert",
            data: { provider: "anthropic", connected: true, key_last4: "1234" },
            warnings: [],
          }),
          { status: 200, headers: { "content-type": "application/json" } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            ok: true,
            command: "relay provider credential delete",
            data: { status: "ok" },
            warnings: [],
          }),
          { status: 200, headers: { "content-type": "application/json" } },
        ),
      );

    await connectProviderCredential("sk-ant-settings-1234");
    await disconnectProviderCredential();

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "http://localhost:8080/v1/settings/provider-credentials",
      expect.objectContaining({
        method: "POST",
        credentials: "include",
        body: JSON.stringify({ provider: "anthropic", api_key: "sk-ant-settings-1234" }),
      }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "http://localhost:8080/v1/settings/provider-credentials/anthropic",
      expect.objectContaining({ method: "DELETE", credentials: "include" }),
    );
  });

  it("uses the public Relay API URL for browser-side provider settings calls", async () => {
    vi.stubEnv("NEXT_PUBLIC_RELAY_API_URL", "https://relay.4gly.dev");
    vi.resetModules();
    const { connectProviderCredential: connectWithPublicURL } = await import(
      "./provider-credentials"
    );
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          ok: true,
          command: "relay provider credential upsert",
          data: { provider: "anthropic", connected: true, key_last4: "1234" },
          warnings: [],
        }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
    );

    await connectWithPublicURL("sk-ant-settings-1234");

    expect(fetchMock).toHaveBeenCalledWith(
      "https://relay.4gly.dev/v1/settings/provider-credentials",
      expect.objectContaining({
        method: "POST",
        credentials: "include",
      }),
    );
  });
});
