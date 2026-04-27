import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { RELAY_API_URL, relayFetch, type RelayEnvelope } from "@/lib/api";
import type { AuthMe } from "@/lib/onboarding";

import { OnboardingClient } from "./onboarding-client";

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

function authStartURL(provider: "github") {
  const url = new URL(`/v1/auth/${provider}/start`, RELAY_API_URL);
  url.searchParams.set("redirect_to", "/onboarding");
  return url.toString();
}

export default async function OnboardingPage() {
  const cookieStore = await cookies();
  const me = await resolveSession(cookieStore.toString());

  if (me?.onboarding_complete && me.default_project_id) {
    redirect(`/style-memory?project=${encodeURIComponent(me.default_project_id)}`);
  }

  return (
    <main
      style={{
        maxWidth: "1040px",
        margin: "0 auto",
        padding: "72px 32px 96px",
      }}
    >
      <p
        style={{
          fontFamily: "var(--font-mono)",
          fontSize: "11px",
          letterSpacing: "0.18em",
          textTransform: "uppercase",
          color: "var(--muted)",
          marginBottom: "16px",
        }}
      >
        Slice 8 · 60 seconds
      </p>
      <h1
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 500,
          fontSize: "clamp(40px, 6.5vw, 64px)",
          lineHeight: 1.05,
          letterSpacing: "-0.025em",
          color: "var(--ink)",
          marginBottom: "20px",
          fontVariationSettings: '"opsz" 144, "SOFT" 50',
        }}
      >
        First run, no keys
      </h1>
      <p
        style={{
          fontFamily: "var(--font-sans)",
          fontSize: "17px",
          lineHeight: 1.6,
          color: "var(--ink-muted)",
          maxWidth: "620px",
          marginBottom: "36px",
        }}
      >
        Relay starts by creating a private workspace. Provider keys are a Settings
        concern, not a gate on the first minute.
      </p>
      {me ? (
        <OnboardingClient userDisplayName={me.display_name ?? me.email} />
      ) : (
        <section style={signInPanelStyle} aria-labelledby="signin-title">
          <h2 id="signin-title" style={signInTitleStyle}>
            Sign in to create your workspace
          </h2>
          <p style={signInCopyStyle}>
            Use an identity provider first. Relay will create your Personal project
            after you return here.
          </p>
          <div style={signInActionsStyle}>
            <a href={authStartURL("github")} style={authButtonStyle}>
              Continue with GitHub
            </a>
          </div>
        </section>
      )}
    </main>
  );
}

const signInPanelStyle: React.CSSProperties = {
  maxWidth: "620px",
  padding: "28px",
  border: "1px solid var(--border)",
  borderRadius: "8px",
  background: "var(--canvas-raised)",
};

const signInTitleStyle: React.CSSProperties = {
  margin: "0 0 12px",
  fontFamily: "var(--font-display)",
  fontWeight: 500,
  fontSize: "30px",
  letterSpacing: "-0.015em",
  fontVariationSettings: '"opsz" 96',
};

const signInCopyStyle: React.CSSProperties = {
  margin: "0 0 22px",
  color: "var(--ink-muted)",
  fontSize: "15px",
  lineHeight: 1.6,
};

const signInActionsStyle: React.CSSProperties = {
  display: "flex",
  flexWrap: "wrap",
  gap: "12px",
};

const authButtonStyle: React.CSSProperties = {
  display: "inline-flex",
  alignItems: "center",
  justifyContent: "center",
  minHeight: "42px",
  padding: "0 16px",
  borderRadius: "8px",
  border: "1px solid var(--border-strong)",
  color: "var(--ink)",
  textDecoration: "none",
  fontFamily: "var(--font-sans)",
  fontSize: "14px",
  fontWeight: 800,
  background: "var(--canvas)",
};
