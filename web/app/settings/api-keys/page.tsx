import { cookies, headers } from "next/headers";
import Link from "next/link";

import { RELAY_API_URL } from "@/lib/api";
import { getDictionary, resolveLocale, translateErrorMessage } from "@/lib/i18n";
import {
  listUserAPIKeys,
  RelayAPIError,
  type UserAPIKeySummary,
} from "@/lib/user-api-keys";
import { RelayTopRail } from "@/components/relay-app-shell";

import { APIKeySettingsClient } from "./api-key-settings-client";

export const dynamic = "force-dynamic";

function signInURL() {
  const url = new URL("/v1/auth/github/start", RELAY_API_URL);
  url.searchParams.set("redirect_to", "/settings/api-keys");
  return url.toString();
}

export default async function APIKeySettingsPage() {
  const cookieStore = await cookies();
  const headerStore = await headers();
  const cookieHeader = cookieStore.toString();
  const locale = resolveLocale({
    cookie: cookieHeader,
    acceptLanguage: headerStore.get("accept-language") ?? undefined,
  });
  const dictionary = getDictionary(locale);

  let authenticated = true;
  let initialKeys: UserAPIKeySummary[] = [];
  let loadError = "";

  try {
    const result = await listUserAPIKeys({ cookie: cookieHeader });
    initialKeys = result.items;
  } catch (error) {
    if (error instanceof RelayAPIError && error.code === "UNAUTHENTICATED") {
      authenticated = false;
    } else {
      loadError = translateErrorMessage({
        error,
        fallback: dictionary.apiKeys.page.loadErrorCopy,
        knownErrors: dictionary.apiKeys.errorMap,
        locale,
      });
    }
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
          <Link href="/" style={backLinkStyle}>
            {dictionary.common.links.projectExplorer}
          </Link>
          <a href="/settings/providers" style={backLinkStyle}>
            {dictionary.common.links.providerSettings}
          </a>
        </nav>
        {authenticated ? (
          loadError ? (
            <section style={panelStyle}>
              <p style={eyebrowStyle}>{dictionary.apiKeys.page.eyebrow}</p>
              <h1 style={titleStyle}>{dictionary.apiKeys.page.loadErrorTitle}</h1>
              <p style={copyStyle}>{loadError}</p>
            </section>
          ) : (
            <APIKeySettingsClient
              copy={dictionary.apiKeys.client}
              errorMap={dictionary.apiKeys.errorMap}
              initialKeys={initialKeys}
              locale={locale}
            />
          )
        ) : (
          <section style={panelStyle}>
            <p style={eyebrowStyle}>{dictionary.apiKeys.page.eyebrow}</p>
            <h1 style={titleStyle}>{dictionary.apiKeys.page.signInTitle}</h1>
            <p style={copyStyle}>{dictionary.apiKeys.page.signInCopy}</p>
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
  maxWidth: "720px",
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
  lineHeight: 1.6,
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
