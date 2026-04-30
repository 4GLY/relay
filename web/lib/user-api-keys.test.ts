import { afterEach, describe, expect, it, vi } from "vitest";

import {
  issueUserAPIKey,
  listUserAPIKeys,
  revokeUserAPIKey,
} from "./user-api-keys";

afterEach(() => {
  vi.restoreAllMocks();
});

describe("user API key client", () => {
  it("lists user-owned API keys", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          ok: true,
          command: "relay settings api keys list",
          data: {
            items: [
              {
                key_id: "key_1",
                name: "CLI",
                token_prefix: "relay_live_abc",
                scope: "global",
                revoked: false,
              },
            ],
          },
          warnings: [],
        }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
    );

    const result = await listUserAPIKeys({ cookie: "relay_session=x" });

    expect(result.items).toHaveLength(1);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/v1/settings/api-keys",
      expect.objectContaining({
        method: "GET",
        credentials: "include",
        headers: { cookie: "relay_session=x" },
      }),
    );
  });

  it("issues a new API key and returns the raw token once", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          ok: true,
          command: "relay settings api key issue",
          data: {
            key_id: "key_1",
            name: "CLI",
            token: "relay_live_secret",
            token_prefix: "relay_live_abc",
            scope: "global",
          },
          warnings: [],
        }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
    );

    const result = await issueUserAPIKey("CLI");

    expect(result.token).toBe("relay_live_secret");
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/v1/settings/api-keys",
      expect.objectContaining({
        method: "POST",
        credentials: "include",
        body: JSON.stringify({ name: "CLI" }),
      }),
    );
  });

  it("revokes a user-owned API key", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          ok: true,
          command: "relay settings api key revoke",
          data: {
            key_id: "key_1",
            name: "CLI",
            token_prefix: "relay_live_abc",
            scope: "global",
            revoked: true,
          },
          warnings: [],
        }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
    );

    const result = await revokeUserAPIKey("key_1");

    expect(result.revoked).toBe(true);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/v1/settings/api-keys/revoke",
      expect.objectContaining({
        method: "POST",
        credentials: "include",
        body: JSON.stringify({ key_id: "key_1" }),
      }),
    );
  });

  it("maps Relay envelope errors to RelayAPIError", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          ok: false,
          command: "relay settings api key issue",
          error: {
            code: "INVALID_INPUT",
            message: "name is required",
            retryable: false,
            missing_fields: ["name"],
          },
        }),
        { status: 400, headers: { "content-type": "application/json" } },
      ),
    );

    await expect(issueUserAPIKey("")).rejects.toMatchObject({
      name: "RelayAPIError",
      code: "INVALID_INPUT",
      message: "name is required",
      missingFields: ["name"],
    });
  });
});
