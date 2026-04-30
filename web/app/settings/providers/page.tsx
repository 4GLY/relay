import { cookies, headers } from "next/headers";

import { RELAY_API_URL } from "@/lib/api";
import { getDictionary, resolveLocale } from "@/lib/i18n";
import { listProviderCredentials } from "@/lib/provider-credentials";
import { RelayTopRail } from "@/components/relay-app-shell";

import { ProviderSettingsClient } from "./provider-settings-client";

export const dynamic = "force-dynamic";

function signInURL() {
  const url = new URL("/v1/auth/github/start", RELAY_API_URL);
  url.searchParams.set("redirect_to", "/settings/providers");
  return url.toString();
}

export default async function ProviderSettingsPage() {
  const cookieStore = await cookies();
  const headerStore = await headers();
  const cookieHeader = cookieStore.toString();
  const locale = resolveLocale({
    cookie: cookieHeader,
    acceptLanguage: headerStore.get("accept-language") ?? undefined,
  });
  const dictionary = getDictionary(locale);
  let initialCredential;
  let authenticated = true;

  try {
    const result = await listProviderCredentials({ cookie: cookieHeader });
    initialCredential = result.credentials.find((item) => item.provider === "anthropic");
  } catch {
    authenticated = false;
  }

  return (
    <>
      <RelayTopRail activeStep="Transform" />
      <main
        style={{
          maxWidth: "960px",
          margin: "0 auto",
          padding: "48px 32px 96px",
        }}
      >
        <nav style={navStyle} aria-label="Settings navigation">
          <a href="/onboarding" style={backLinkStyle}>
            {dictionary.common.links.backToOnboarding}
          </a>
          <a href="/settings/api-keys" style={backLinkStyle}>
            {dictionary.common.links.apiKeys}
          </a>
        </nav>
        {authenticated ? (
          <ProviderSettingsClient
            copy={dictionary.providers.client}
            errorMap={dictionary.providers.errorMap}
            initialCredential={initialCredential}
            locale={locale}
          />
        ) : (
          <section style={panelStyle}>
            <p style={eyebrowStyle}>{dictionary.providers.page.eyebrow}</p>
            <h1 style={titleStyle}>{dictionary.providers.page.signInTitle}</h1>
            <p style={copyStyle}>{dictionary.providers.page.signInCopy}</p>
            <a href={signInURL()} style={buttonStyle}>
              {dictionary.common.continueWithGitHub}
            </a>
          </section>
        )}
      </main>
    </>
  );
}

const navStyle: React.CSSProperties = {
  display: "flex",
  flexWrap: "wrap",
  gap: "16px",
  marginBottom: "22px",
};

const backLinkStyle: React.CSSProperties = {
  display: "inline-block",
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
