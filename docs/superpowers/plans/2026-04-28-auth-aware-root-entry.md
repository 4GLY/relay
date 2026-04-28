# Auth-Aware Root Entry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the root scaffold page with an authentication-aware Relay entry screen that sends unauthenticated users to GitHub sign-in and routes authenticated users into onboarding or Style Memory.

**Architecture:** The root `web/app/page.tsx` becomes a dynamic server component, mirroring the existing `/onboarding` session resolution pattern with `cookies()`, `relayFetch("/v1/auth/me")`, and `redirect()`. No backend changes are needed because `/v1/auth/me` already returns `onboarding_complete` and `default_project_id`. A small local helper owns the GitHub OAuth start URL so root and onboarding can stay consistent without introducing a broader auth abstraction yet.

**Tech Stack:** Next.js 16 App Router server component, React 19, Vitest, Testing Library, Relay `relayFetch`, GitHub OAuth start endpoint.

---

## File Structure

- Modify `web/app/page.tsx`: remove scaffold cards and implement the auth-aware root behavior.
- Modify `web/app/page.test.tsx`: replace the old revoked placeholder snapshot regression with root auth routing tests.
- Optional later refactor, not part of this plan: extract duplicated `authStartURL()` / session resolution from root and onboarding into `web/lib/auth.ts` if a third page needs it.

The plan deliberately keeps the first implementation small: one page and one test file. `/onboarding` remains the first-run workspace creation screen; `/` becomes the entry gate.

---

### Task 1: Root Auth Routing Tests

**Files:**
- Modify: `web/app/page.test.tsx`

- [ ] **Step 1: Replace the old scaffold test with routing tests**

Replace `web/app/page.test.tsx` with:

```tsx
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import HomePage from "./page";

const redirect = vi.fn((path: string) => {
  throw new Error(`NEXT_REDIRECT:${path}`);
});
const cookies = vi.fn();
const relayFetch = vi.fn();

vi.mock("next/navigation", () => ({ redirect }));
vi.mock("next/headers", () => ({ cookies }));
vi.mock("@/lib/api", () => ({
  RELAY_API_URL: "https://relay.4gly.dev",
  relayFetch,
}));

function cookieStore(value = "relay_session=test") {
  return {
    toString: () => value,
  };
}

function authResponse(status: number, body: unknown, ok = status >= 200 && status < 300) {
  return {
    status,
    ok,
    json: async () => body,
  } as Response;
}

describe("<HomePage>", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    cookies.mockResolvedValue(cookieStore());
  });

  it("shows the GitHub sign-in entry when there is no session", async () => {
    relayFetch.mockResolvedValueOnce(
      authResponse(401, {
        ok: false,
        command: "relay auth me",
        error: { code: "UNAUTHENTICATED", message: "missing session", retryable: false },
      }),
    );

    render(await HomePage());

    expect(screen.getByRole("heading", { name: "Relay" })).toBeVisible();
    expect(screen.getByText("A quiet engine that turns chaos into swans.")).toBeVisible();
    expect(screen.getByRole("link", { name: "Continue with GitHub" })).toHaveAttribute(
      "href",
      "https://relay.4gly.dev/v1/auth/github/start?redirect_to=%2Fonboarding",
    );
    expect(screen.queryByText("Sharable Snapshot")).not.toBeInTheDocument();
  });

  it("redirects authenticated users who still need onboarding to /onboarding", async () => {
    relayFetch.mockResolvedValueOnce(
      authResponse(200, {
        ok: true,
        command: "relay auth me",
        data: {
          user_id: "user_1",
          display_name: "Hoon",
          onboarding_complete: false,
        },
        warnings: [],
      }),
    );

    await expect(HomePage()).rejects.toThrow("NEXT_REDIRECT:/onboarding");
    expect(redirect).toHaveBeenCalledWith("/onboarding");
  });

  it("redirects fully onboarded users to their Style Memory project", async () => {
    relayFetch.mockResolvedValueOnce(
      authResponse(200, {
        ok: true,
        command: "relay auth me",
        data: {
          user_id: "user_1",
          display_name: "Hoon",
          onboarding_complete: true,
          default_project_id: "proj_personal",
        },
        warnings: [],
      }),
    );

    await expect(HomePage()).rejects.toThrow(
      "NEXT_REDIRECT:/style-memory?project=proj_personal",
    );
    expect(redirect).toHaveBeenCalledWith("/style-memory?project=proj_personal");
  });
});
```

- [ ] **Step 2: Run the failing tests**

Run:

```bash
cd web
npm run test -- app/page.test.tsx
```

Expected: FAIL because the current root page is a static scaffold, does not call `relayFetch`, does not redirect, and still renders scaffold cards.

---

### Task 2: Auth-Aware Root Page

**Files:**
- Modify: `web/app/page.tsx`

- [ ] **Step 1: Replace scaffold cards with session resolution**

Replace `web/app/page.tsx` with:

```tsx
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { RELAY_API_URL, relayFetch, type RelayEnvelope } from "@/lib/api";
import type { AuthMe } from "@/lib/onboarding";

export const dynamic = "force-dynamic";

async function resolveSession(cookieHeader: string): Promise<AuthMe | null> {
  const res = await relayFetch("/v1/auth/me", {
    method: "GET",
    headers: { cookie: cookieHeader },
    cache: "no-store",
  });
  if (res.status === 401) return null;
  if (!res.ok) return null;
  const body = (await res.json()) as RelayEnvelope<AuthMe>;
  if (!body.ok) return null;
  return body.data;
}

function authStartURL() {
  const url = new URL("/v1/auth/github/start", RELAY_API_URL);
  url.searchParams.set("redirect_to", "/onboarding");
  return url.toString();
}

export default async function HomePage() {
  const cookieStore = await cookies();
  const me = await resolveSession(cookieStore.toString());

  if (me?.onboarding_complete && me.default_project_id) {
    redirect(`/style-memory?project=${encodeURIComponent(me.default_project_id)}`);
  }

  if (me) {
    redirect("/onboarding");
  }

  return (
    <main style={pageStyle}>
      <p style={eyebrowStyle}>4gly Labs · Relay</p>
      <h1 style={titleStyle}>Relay</h1>
      <p style={subtitleStyle}>A quiet engine that turns chaos into swans.</p>
      <section style={panelStyle} aria-labelledby="entry-title">
        <h2 id="entry-title" style={panelTitleStyle}>
          Sign in to start
        </h2>
        <p style={panelCopyStyle}>
          Relay creates a private workspace first. Provider keys stay in Settings,
          not in first-run setup.
        </p>
        <a href={authStartURL()} style={buttonStyle}>
          Continue with GitHub
        </a>
      </section>
    </main>
  );
}

const pageStyle: React.CSSProperties = {
  maxWidth: "760px",
  margin: "0 auto",
  padding: "96px 32px 120px",
};

const eyebrowStyle: React.CSSProperties = {
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  letterSpacing: "0.18em",
  textTransform: "uppercase",
  color: "var(--muted)",
  marginBottom: "32px",
};

const titleStyle: React.CSSProperties = {
  fontFamily: "var(--font-display)",
  fontWeight: 500,
  fontSize: "clamp(56px, 9vw, 112px)",
  lineHeight: 0.95,
  letterSpacing: "-0.03em",
  color: "var(--ink)",
  marginBottom: "24px",
  fontVariationSettings: '"opsz" 144, "SOFT" 50',
};

const subtitleStyle: React.CSSProperties = {
  fontFamily: "var(--font-display)",
  fontStyle: "italic",
  fontWeight: 400,
  fontSize: "clamp(20px, 2.8vw, 28px)",
  lineHeight: 1.35,
  color: "var(--ink-muted)",
  maxWidth: "640px",
  marginBottom: "42px",
  fontVariationSettings: '"opsz" 48',
};

const panelStyle: React.CSSProperties = {
  maxWidth: "560px",
  padding: "28px",
  border: "1px solid var(--border)",
  borderRadius: "8px",
  background: "var(--canvas-raised)",
};

const panelTitleStyle: React.CSSProperties = {
  margin: "0 0 12px",
  fontFamily: "var(--font-display)",
  fontWeight: 500,
  fontSize: "30px",
  letterSpacing: "-0.015em",
  fontVariationSettings: '"opsz" 96',
};

const panelCopyStyle: React.CSSProperties = {
  margin: "0 0 22px",
  color: "var(--ink-muted)",
  fontSize: "15px",
  lineHeight: 1.6,
};

const buttonStyle: React.CSSProperties = {
  display: "inline-flex",
  alignItems: "center",
  justifyContent: "center",
  minHeight: "42px",
  padding: "0 16px",
  borderRadius: "8px",
  border: "1px solid var(--border-strong)",
  color: "var(--canvas)",
  background: "var(--ink)",
  textDecoration: "none",
  fontFamily: "var(--font-sans)",
  fontSize: "14px",
  fontWeight: 800,
};
```

- [ ] **Step 2: Run the focused tests**

Run:

```bash
cd web
npm run test -- app/page.test.tsx
```

Expected: PASS.

---

### Task 3: Root Entry Browser QA Coverage

**Files:**
- Modify: `web/e2e/relay-live-smoke.spec.ts`

- [ ] **Step 1: Add root entry smoke coverage**

Add this test near the top of `web/e2e/relay-live-smoke.spec.ts`:

```ts
test("root entry shows sign-in or redirects to the active app", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByText(/Relay|Style Memory|First run, no keys/i).first()).toBeVisible();
  await page.screenshot({
    path: `${screenshotDir}/s10-root-entry.png`,
    fullPage: true,
  });
});
```

This smoke is intentionally permissive because it runs against either an unauthenticated browser state or an authenticated gstack state:

- unauthenticated: root shows the Relay sign-in entry
- authenticated but not onboarded: root redirects to `/onboarding`
- authenticated and onboarded: root redirects to `/style-memory?project=...`

- [ ] **Step 2: Run Playwright discovery**

Run:

```bash
cd web
npx playwright test --list
```

Expected: one additional root-entry test appears in both desktop and mobile projects.

- [ ] **Step 3: Run live smoke**

Run:

```bash
cd web
RELAY_WEB_BASE_URL=https://relay.4gly.dev npm run qa:e2e
```

Expected: root, onboarding, style-memory, settings, and unknown public snapshot checks pass on desktop and mobile. Positive public snapshot checks remain skipped unless `RELAY_QA_PUBLIC_SNAPSHOT_TOKEN` is set.

---

### Task 4: Verification And Commit

**Files:**
- Verify all changed files.

- [ ] **Step 1: Run web unit tests**

Run:

```bash
cd web
npm run test
```

Expected: PASS.

- [ ] **Step 2: Run lint and typecheck**

Run:

```bash
cd web
npm run lint
npm run typecheck
```

Expected: PASS.

- [ ] **Step 3: Run backend tests for regression confidence**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 4: Commit**

Run:

```bash
git status --short
git add web/app/page.tsx web/app/page.test.tsx web/e2e/relay-live-smoke.spec.ts docs/superpowers/plans/2026-04-28-auth-aware-root-entry.md
git commit -m "feat: make root auth aware"
```

Expected: commit succeeds.

---

## Self-Review

- Spec coverage: the plan changes `/` from a scaffold to a login/start gate, redirects already-authenticated users to `/onboarding` or `/style-memory`, keeps `/onboarding` responsible for first-run workspace creation, and does not introduce a marketing landing page.
- Placeholder scan: no task uses TBD, TODO, "implement later", or vague testing language.
- Type consistency: `AuthMe` comes from `web/lib/onboarding.ts`; `relayFetch`, `RELAY_API_URL`, and `RelayEnvelope` come from `web/lib/api.ts`; redirect paths match existing `/onboarding` and `/style-memory?project=...` routes.
