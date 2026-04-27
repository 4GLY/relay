import { expect, test } from "@playwright/test";
import { mkdirSync } from "node:fs";

const projectID = process.env.RELAY_QA_PROJECT_ID ?? "proj_28cc65685c63";
const publicSnapshotToken = process.env.RELAY_QA_PUBLIC_SNAPSHOT_TOKEN;
const screenshotDir = "../.gstack/qa-reports/screenshots";

test.beforeAll(() => {
  mkdirSync(screenshotDir, { recursive: true });
});

test.describe("Relay V2 live smoke", () => {
  test("onboarding is reachable and keyless", async ({ page }) => {
    await page.goto("/onboarding");
    await expect(page.getByRole("heading", { name: /First run, no keys/i })).toBeVisible();
    await page.screenshot({
      path: `${screenshotDir}/s10-onboarding.png`,
      fullPage: true,
    });
  });

  test("style memory renders an authenticated or redirectable state", async ({ page }) => {
    await page.goto(`/style-memory?project=${projectID}`);
    await expect(page.getByText(/Style Memory|Relay/i).first()).toBeVisible();
    await page.screenshot({
      path: `${screenshotDir}/s10-style-memory.png`,
      fullPage: true,
    });
  });

  test("provider settings renders an authenticated or redirectable state", async ({ page }) => {
    await page.goto("/settings/providers");
    await expect(page.getByText(/Provider settings|Sign in/i).first()).toBeVisible();
    await page.screenshot({
      path: `${screenshotDir}/s10-settings-providers.png`,
      fullPage: true,
    });
  });

  test("unknown public snapshot returns the revoked-state page", async ({ page }) => {
    const response = await page.goto("/p/unknown_s10_snapshot_token");
    expect(response?.status()).toBe(410);
    await expect(page.getByText(/no longer available/i)).toBeVisible();
    await page.screenshot({
      path: `${screenshotDir}/s10-public-snapshot-410.png`,
      fullPage: true,
    });
  });

  test("public snapshot renders canonical HTML when a token is supplied", async ({ page }) => {
    test.skip(!publicSnapshotToken, "Set RELAY_QA_PUBLIC_SNAPSHOT_TOKEN to verify a live public snapshot");

    const response = await page.goto(`/p/${publicSnapshotToken}`);
    expect(response?.status()).toBe(200);
    await expect(page.locator('meta[property="og:image"]')).toHaveCount(1);
    await page.screenshot({
      path: `${screenshotDir}/s10-public-snapshot.png`,
      fullPage: true,
    });
  });
});
