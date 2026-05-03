import { describe, expect, it } from "vitest";

import { RELAY_LOCALE_COOKIE } from "@/i18n/routing";

import { POST } from "./route";

describe("POST /settings/language", () => {
  it("stores a valid form locale and redirects to the submitted local path", async () => {
    const formData = new FormData();
    formData.set("locale", "ko");
    formData.set("redirectTo", "/projects/proj_1");

    const response = await POST(
      new Request("https://relay.test/settings/language", {
        method: "POST",
        body: formData,
      }),
    );

    expect(response.status).toBe(307);
    expect(response.headers.get("location")).toBe("https://relay.test/projects/proj_1");
    expect(response.cookies.get(RELAY_LOCALE_COOKIE)?.value).toBe("ko");
  });

  it("rejects an unsupported JSON locale", async () => {
    const response = await POST(
      new Request("https://relay.test/settings/language", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ locale: "fr" }),
      }),
    );

    await expect(response.json()).resolves.toMatchObject({
      ok: false,
      error: { code: "INVALID_LOCALE" },
    });
    expect(response.status).toBe(400);
  });

  it("stores a valid JSON locale and returns the updated locale", async () => {
    const response = await POST(
      new Request("https://relay.test/settings/language", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ locale: "ko", redirectTo: "/projects/proj_1" }),
      }),
    );

    await expect(response.json()).resolves.toEqual({ ok: true, locale: "ko" });
    expect(response.cookies.get(RELAY_LOCALE_COOKIE)?.value).toBe("ko");
  });

  it("rejects an external form redirect and redirects to root", async () => {
    const formData = new FormData();
    formData.set("locale", "ko");
    formData.set("redirectTo", "https://example.com/projects/proj_1");

    const response = await POST(
      new Request("https://relay.test/settings/language", {
        method: "POST",
        body: formData,
      }),
    );

    expect(response.status).toBe(307);
    expect(response.headers.get("location")).toBe("https://relay.test/");
    expect(response.cookies.get(RELAY_LOCALE_COOKIE)?.value).toBe("ko");
  });

  it("rejects a backslash-host redirect and redirects to root", async () => {
    const formData = new FormData();
    formData.set("locale", "ko");
    formData.set("redirectTo", "/\\evil.com");

    const response = await POST(
      new Request("http://relay.test/settings/language", {
        method: "POST",
        body: formData,
      }),
    );

    expect(response.status).toBe(307);
    expect(response.headers.get("location")).toBe("http://relay.test/");
    expect(response.cookies.get(RELAY_LOCALE_COOKIE)?.value).toBe("ko");
  });
});
