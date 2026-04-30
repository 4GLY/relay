import { expect, test } from "@playwright/test";
import { mkdirSync } from "node:fs";

const projectID = process.env.RELAY_QA_PROJECT_ID ?? "proj_28cc65685c63";
const publicSnapshotToken = process.env.RELAY_QA_PUBLIC_SNAPSHOT_TOKEN;
const sessionCookie = process.env.RELAY_QA_SESSION_COOKIE;
const authenticatedProjectID = process.env.RELAY_QA_AUTH_PROJECT_ID ?? projectID;
const screenshotDir = "../.gstack/qa-reports/screenshots";
const baseURL = process.env.RELAY_WEB_BASE_URL ?? "https://relay.4gly.dev";
const isLocalWebBase = /^https?:\/\/(127\.0\.0\.1|localhost)(:|\/|$)/.test(baseURL);

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
    test.skip(isLocalWebBase, "Next /p route is a placeholder redirect; verify canonical public snapshots on live Go routing");

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
    test.skip(isLocalWebBase, "Next /p route is a placeholder redirect; verify canonical public snapshots on live Go routing");

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

  test("onboarding redirects completed users into Project Explorer", async ({ page }) => {
    await page.goto("/onboarding");
    await expect(page).toHaveURL(new RegExp(`/projects/${authenticatedProjectID}$`));
    await expect(page.getByText(/Project Explorer/i).first()).toBeVisible();
    await expect(page.getByRole("link", { name: "Style Memory" })).toBeVisible();
    await page.screenshot({
      path: `${screenshotDir}/s10-auth-onboarding-redirect.png`,
      fullPage: true,
    });
  });

  test("project explorer links core project surfaces", async ({ page }) => {
    await page.goto("/");
    await expect(page).toHaveURL(new RegExp(`/projects/${authenticatedProjectID}$`));
    await expect(page.getByText(/Project Explorer/i).first()).toBeVisible();
    await expect(page.getByText(/Snapshots/i).first()).toBeVisible();
    await expect(page.getByRole("link", { name: "Style Memory" })).toHaveAttribute(
      "href",
      `/style-memory?project=${authenticatedProjectID}`,
    );
    await expect(page.getByRole("link", { name: "Trace Browser" })).toHaveAttribute(
      "href",
      `/projects/${authenticatedProjectID}/traces`,
    );
    await expect(page.getByRole("link", { name: "Decision Graph" })).toHaveAttribute(
      "href",
      `/projects/${authenticatedProjectID}/graph`,
    );
    await expect(page.getByRole("link", { name: "Packet Builder" })).toHaveAttribute(
      "href",
      `/projects/${authenticatedProjectID}/packet-builder`,
    );
    if (publicSnapshotToken && !isLocalWebBase) {
      await expect(page.getByRole("link", { name: "Open public snapshot" })).toHaveAttribute(
        "href",
        `/p/${publicSnapshotToken}`,
      );
    }
    await page.screenshot({
      path: `${screenshotDir}/s10-auth-project-explorer.png`,
      fullPage: true,
    });
  });

  test("trace browser renders authenticated judgment traces", async ({ page }) => {
    await page.goto(`/projects/${authenticatedProjectID}/traces`);
    await expect(page.getByRole("heading", { name: "Trace Browser" })).toBeVisible();
    await expect(page.getByText(/Prefer explicit recovery actions over generic error states/i)).toBeVisible();
    await expect(page.getByText(/qa:live:e2e/i)).toBeVisible();
    await expect(page.getByRole("link", { name: "Project Explorer" })).toHaveAttribute(
      "href",
      `/projects/${authenticatedProjectID}`,
    );
    await page.screenshot({
      path: `${screenshotDir}/s10-auth-trace-browser.png`,
      fullPage: true,
    });
  });

  test("decision graph renders authenticated evidence map", async ({ page }) => {
    await page.goto(`/projects/${authenticatedProjectID}/graph`);
    await expect(page.getByRole("heading", { name: "Decision Graph" })).toBeVisible();
    await expect(page.getByText(/Prefer explicit recovery actions over generic error states/i)).toBeVisible();
    await expect(page.getByText(/packet snapshot|judgment trace/i).first()).toBeVisible();
    await expect(page.getByRole("link", { name: "Project Explorer" })).toHaveAttribute(
      "href",
      `/projects/${authenticatedProjectID}`,
    );
    await page.screenshot({
      path: `${screenshotDir}/s10-auth-decision-graph.png`,
      fullPage: true,
    });
  });

  test("packet builder renders the latest snapshot document", async ({ page }) => {
    await page.goto(`/projects/${authenticatedProjectID}/packet-builder`);
    await expect(page.getByRole("heading", { name: "Packet Builder" })).toBeVisible();
    await expect(page.getByText(/Prefer explicit recovery actions over generic error states/i)).toBeVisible();
    await expect(page.getByText("Source evidence")).toBeVisible();
    await expect(page.getByRole("link", { name: "Project Explorer" })).toHaveAttribute(
      "href",
      `/projects/${authenticatedProjectID}`,
    );
    await expect(page.getByRole("link", { name: "Decision Graph" })).toHaveAttribute(
      "href",
      `/projects/${authenticatedProjectID}/graph`,
    );
    await page.screenshot({
      path: `${screenshotDir}/s10-auth-packet-builder.png`,
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
    test.skip(isLocalWebBase, "Provider credential mutation requires same-origin live routing");

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
