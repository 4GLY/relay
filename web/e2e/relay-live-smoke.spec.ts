import { expect, test } from "@playwright/test";
import { mkdirSync } from "node:fs";

const projectID = process.env.RELAY_QA_PROJECT_ID ?? "proj_28cc65685c63";
const publicSnapshotToken = process.env.RELAY_QA_PUBLIC_SNAPSHOT_TOKEN;
const sessionCookie = process.env.RELAY_QA_SESSION_COOKIE;
const authenticatedProjectID = process.env.RELAY_QA_AUTH_PROJECT_ID ?? projectID;
const screenshotDir = "../.gstack/qa-reports/screenshots";

test.beforeAll(() => {
  mkdirSync(screenshotDir, { recursive: true });
});

async function addSessionCookie(page: import("@playwright/test").Page) {
  if (!sessionCookie) return;
  const baseURL = test.info().project.use.baseURL ?? process.env.RELAY_WEB_BASE_URL ?? "https://relay.4gly.dev";
  await page.context().addCookies([
    {
      name: "relay_session",
      value: sessionCookie,
      url: baseURL,
      httpOnly: true,
      sameSite: "Lax",
      secure: baseURL.startsWith("https://"),
    },
  ]);
}

test.describe("Relay V2 live smoke", () => {
  test("root entry shows sign-in or redirects to the active app", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByText(/Relay|Style Memory|First run, no keys/i).first()).toBeVisible();
    await page.screenshot({
      path: `${screenshotDir}/s10-root-entry.png`,
      fullPage: true,
    });
  });

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
    const ogResponse = await page.request.get(`/p/${publicSnapshotToken}/og.png`);
    expect(ogResponse.status()).toBe(200);
    expect(ogResponse.headers()["content-type"]).toContain("image/png");
    await page.screenshot({
      path: `${screenshotDir}/s10-public-snapshot.png`,
      fullPage: true,
    });
  });
});

test.describe("Relay authenticated live smoke", () => {
  test.skip(!sessionCookie, "Set RELAY_QA_SESSION_COOKIE to verify authenticated user flows");

  test.beforeEach(async ({ page }) => {
    await addSessionCookie(page);
  });

  test("onboarding redirects completed users into Style Memory", async ({ page }) => {
    await page.goto("/onboarding");
    await expect(page.getByRole("heading", { name: /Style Memory/i })).toBeVisible();
    await expect(page.getByText(/Relay E2E|QA E2E|Style Memory/i).first()).toBeVisible();
    await page.screenshot({
      path: `${screenshotDir}/s10-auth-onboarding-redirect.png`,
      fullPage: true,
    });
  });

  test("style memory renders the authenticated project queue", async ({ page }) => {
    await page.goto(`/style-memory?project=${authenticatedProjectID}`);
    await expect(page.getByRole("heading", { name: /Style Memory/i })).toBeVisible();
    await expect(page.getByText(/Proposals/i).first()).toBeVisible();
    await expect(page.getByText(/qa_live\s+×\s+style_memory/i).first()).toBeVisible();
    await page.screenshot({
      path: `${screenshotDir}/s10-auth-style-memory.png`,
      fullPage: true,
    });
  });

  test("provider settings validates, stores masked metadata, and disconnects", async ({ page }) => {
    test.skip(
      test.info().project.name !== "chromium",
      "Provider credential mutation runs once to avoid cross-project state races",
    );

    await page.goto("/settings/providers");
    await expect(page.getByRole("heading", { name: /Claude provider/i })).toBeVisible();
    await expect(page.getByText(/not part of first-run onboarding/i)).toBeVisible();

    await page.getByLabel(/Anthropic API key/i).fill("not-anthropic");
    await page.getByTestId("connect-provider").click();
    await expect(page.getByText(/Anthropic keys must start with sk-ant-/i)).toBeVisible();

    await page.getByLabel(/Anthropic API key/i).fill(`sk-ant-e2e-${Date.now()}-1234`);
    await page.getByTestId("connect-provider").click();
    await expect(page.getByText("Connected")).toBeVisible();
    await expect(page.getByText(/ending 1234/i)).toBeVisible();

    await page.getByTestId("disconnect-provider").click();
    await expect(page.getByText("Not connected")).toBeVisible();
    await page.screenshot({
      path: `${screenshotDir}/s10-auth-settings-providers.png`,
      fullPage: true,
    });
  });
});
