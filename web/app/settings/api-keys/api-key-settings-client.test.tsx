import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { issueUserAPIKey, revokeUserAPIKey } from "@/lib/user-api-keys";

import { APIKeySettingsClient } from "./api-key-settings-client";

vi.mock("@/lib/user-api-keys", () => ({
  issueUserAPIKey: vi.fn(),
  revokeUserAPIKey: vi.fn(),
}));

describe("<APIKeySettingsClient>", () => {
  let clipboardWriteText: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.clearAllMocks();
    clipboardWriteText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: {
        writeText: clipboardWriteText,
      },
    });
  });

  it("issues a key, reveals the raw token once, and copies it", async () => {
    vi.mocked(issueUserAPIKey).mockResolvedValueOnce({
      key_id: "key_1",
      name: "CLI",
      token: "relay_live_secret",
      token_prefix: "relay_live_abc",
      scope: "global",
    });

    const user = userEvent.setup();
    render(<APIKeySettingsClient initialKeys={[]} />);

    await user.type(screen.getByLabelText(/key name/i), "CLI");
    await user.click(screen.getByTestId("issue-api-key"));

    await waitFor(() => {
      expect(issueUserAPIKey).toHaveBeenCalledWith("CLI");
      expect(screen.getByDisplayValue("relay_live_secret")).toBeInTheDocument();
      expect(screen.getByText("Issued a new Relay API key.")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("copy-issued-api-key"));

    await waitFor(() => {
      expect(screen.getByText("Copied")).toBeInTheDocument();
    });
  });

  it("requires an explicit confirmation step before revoking", async () => {
    vi.mocked(revokeUserAPIKey).mockResolvedValueOnce({
      key_id: "key_1",
      name: "CLI",
      token_prefix: "relay_live_abc",
      scope: "global",
      revoked: true,
    });

    const user = userEvent.setup();
    render(
      <APIKeySettingsClient
        initialKeys={[
          {
            key_id: "key_1",
            name: "CLI",
            token_prefix: "relay_live_abc",
            scope: "global",
            revoked: false,
          },
        ]}
      />,
    );

    await user.click(screen.getByTestId("revoke-api-key-key_1"));
    expect(screen.getByText("This revokes the key immediately.")).toBeInTheDocument();

    await user.click(screen.getByTestId("confirm-revoke-key_1"));

    await waitFor(() => {
      expect(revokeUserAPIKey).toHaveBeenCalledWith("key_1");
      expect(screen.getByText("Revoked")).toBeInTheDocument();
    });
  });
});
