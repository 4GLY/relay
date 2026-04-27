import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import HomePage from "./page";

describe("<HomePage>", () => {
  it("does not link the snapshot card to a revoked placeholder snapshot", () => {
    render(<HomePage />);

    // Regression: ISSUE-002 - the home page linked users to /p/example, a 410 page.
    // Found by /qa on 2026-04-27.
    // Report: .gstack/qa-reports/qa-report-relay-4gly-dev-2026-04-27.md
    expect(screen.getByText("Sharable Snapshot").closest("a")).not.toHaveAttribute(
      "href",
      "/p/example",
    );
  });
});
