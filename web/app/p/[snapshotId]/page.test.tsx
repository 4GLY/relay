import { describe, expect, it, vi } from "vitest";

import SnapshotPage from "./page";

const mocks = vi.hoisted(() => {
  return {
    redirect: vi.fn((path: string) => {
      throw new Error(`NEXT_REDIRECT:${path}`);
    }),
  };
});

vi.mock("next/navigation", () => ({ redirect: mocks.redirect }));

describe("<SnapshotPage>", () => {
  it("redirects to the Go-backed canonical public snapshot route", async () => {
    await expect(
      SnapshotPage({ params: Promise.resolve({ snapshotId: "psnap_token with space" }) }),
    ).rejects.toThrow("NEXT_REDIRECT:/p/psnap_token%20with%20space");

    expect(mocks.redirect).toHaveBeenCalledWith("/p/psnap_token%20with%20space");
  });
});
