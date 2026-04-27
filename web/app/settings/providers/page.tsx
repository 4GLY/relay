import { cookies } from "next/headers";
import Link from "next/link";

import { RELAY_API_URL } from "@/lib/api";
import { listProviderCredentials } from "@/lib/provider-credentials";

import { ProviderSettingsClient } from "./provider-settings-client";

export const dynamic = "force-dynamic";

function signInURL() {
  const url = new URL("/v1/auth/github/start", RELAY_API_URL);
  url.searchParams.set("redirect_to", "/settings/providers");
  return url.toString();
}

export default async function ProviderSettingsPage() {
  const cookieStore = await cookies();
  const cookieHeader = cookieStore.toString();
  let initialCredential;
  let authenticated = true;

  try {
    const result = await listProviderCredentials({ cookie: cookieHeader });
    initialCredential = result.credentials.find((item) => item.provider === "anthropic");
  } catch {
    authenticated = false;
  }

  return (
    <main
      style={{
        maxWidth: "960px",
        margin: "0 auto",
        padding: "72px 32px 96px",
      }}
    >
      <Link href="/onboarding" style={backLinkStyle}>
        Back to onboarding
      </Link>
      {authenticated ? (
        <ProviderSettingsClient initialCredential={initialCredential} />
      ) : (
        <section style={panelStyle}>
          <p style={eyebrowStyle}>Settings · provider credentials</p>
          <h1 style={titleStyle}>Sign in first</h1>
          <p style={copyStyle}>Provider credentials are user-owned settings.</p>
          <a href={signInURL()} style={buttonStyle}>
            Continue with GitHub
          </a>
        </section>
      )}
    </main>
  );
}

const backLinkStyle: React.CSSProperties = {
  display: "inline-block",
  marginBottom: "22px",
  color: "var(--ink-muted)",
  fontSize: "13px",
  fontWeight: 800,
  textDecoration: "none",
};

const panelStyle: React.CSSProperties = {
  maxWidth: "620px",
  padding: "30px",
  border: "1px solid var(--border)",
  borderRadius: "8px",
  background: "var(--canvas-raised)",
};

const eyebrowStyle: React.CSSProperties = {
  margin: "0 0 12px",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
  color: "var(--muted)",
};

const titleStyle: React.CSSProperties = {
  margin: "0 0 14px",
  fontFamily: "var(--font-display)",
  fontSize: "40px",
  fontWeight: 500,
};

const copyStyle: React.CSSProperties = {
  margin: "0 0 22px",
  color: "var(--ink-muted)",
};

const buttonStyle: React.CSSProperties = {
  display: "inline-flex",
  alignItems: "center",
  minHeight: "42px",
  borderRadius: "8px",
  padding: "0 18px",
  background: "var(--ink)",
  color: "var(--canvas)",
  fontWeight: 800,
  textDecoration: "none",
};
