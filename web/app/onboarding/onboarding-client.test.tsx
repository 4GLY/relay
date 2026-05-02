import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { completeOnboarding } from "@/lib/onboarding";

import { OnboardingClient } from "./onboarding-client";

const push = vi.fn();
const refresh = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push, refresh }),
}));

vi.mock("@/lib/onboarding", () => ({
  completeOnboarding: vi.fn(),
}));

describe("<OnboardingClient>", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("completes keyless onboarding and routes to Project Explorer with the default project", async () => {
    vi.mocked(completeOnboarding).mockResolvedValueOnce({
      onboarding_complete: true,
      default_project_id: "proj_personal",
    });

    const user = userEvent.setup();
    render(<OnboardingClient locale="en" userDisplayName="Hoon" />);

    await user.click(screen.getByTestId("complete-onboarding"));

    await waitFor(() => {
      expect(completeOnboarding).toHaveBeenCalledTimes(1);
      expect(push).toHaveBeenCalledWith("/projects/proj_personal");
      expect(refresh).toHaveBeenCalledTimes(1);
    });
  });

  it("shows an error when onboarding fails", async () => {
    vi.mocked(completeOnboarding).mockRejectedValueOnce(new Error("missing session cookie"));

    const user = userEvent.setup();
    render(<OnboardingClient locale="en" />);

    await user.click(screen.getByTestId("complete-onboarding"));

    expect(await screen.findByRole("alert")).toHaveTextContent("Could not finish onboarding.");
  });
});
