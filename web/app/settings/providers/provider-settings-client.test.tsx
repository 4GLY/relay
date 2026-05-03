import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  connectProviderCredential,
  disconnectProviderCredential,
} from "@/lib/provider-credentials";

import { ProviderSettingsClient } from "./provider-settings-client";

const refresh = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ refresh }),
}));

vi.mock("@/lib/provider-credentials", () => ({
  connectProviderCredential: vi.fn(),
  disconnectProviderCredential: vi.fn(),
}));

describe("<ProviderSettingsClient>", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("connects an Anthropic credential from Settings", async () => {
    vi.mocked(connectProviderCredential).mockResolvedValueOnce({
      provider: "anthropic",
      connected: true,
      key_prefix: "sk-ant-settings...",
      key_last4: "1234",
    });

    const user = userEvent.setup();
    render(<ProviderSettingsClient />);

    await user.type(screen.getByLabelText(/anthropic api key/i), "sk-ant-settings-1234");
    await user.click(screen.getByTestId("connect-provider"));

    await waitFor(() => {
      expect(connectProviderCredential).toHaveBeenCalledWith("sk-ant-settings-1234");
      expect(screen.getByText("Connected")).toBeInTheDocument();
      expect(refresh).toHaveBeenCalledTimes(1);
    });
  });

  it("disconnects an existing Anthropic credential", async () => {
    vi.mocked(disconnectProviderCredential).mockResolvedValueOnce();

    const user = userEvent.setup();
    render(
      <ProviderSettingsClient
        initialCredential={{
          provider: "anthropic",
          connected: true,
          key_prefix: "sk-ant-settings...",
          key_last4: "1234",
        }}
      />,
    );

    await user.click(screen.getByTestId("disconnect-provider"));

    await waitFor(() => {
      expect(disconnectProviderCredential).toHaveBeenCalledTimes(1);
      expect(screen.getByText("Not connected")).toBeInTheDocument();
    });
  });
});
