import { cookies, headers } from "next/headers";
import { redirect } from "next/navigation";

import { RELAY_API_URL, relayFetch, type RelayEnvelope } from "@/lib/api";
import { getDictionary, resolveLocale } from "@/lib/i18n";
import type { AuthMe } from "@/lib/onboarding";
import { RelayTopRail } from "@/components/relay-app-shell";

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
  const headerStore = await headers();
  const me = await resolveSession(cookieStore.toString());
  const locale = resolveLocale({
    cookie: cookieStore.toString(),
    acceptLanguage: headerStore.get("accept-language") ?? undefined,
  });
  const dictionary = getDictionary(locale);

  if (me?.onboarding_complete && me.default_project_id) {
    redirect(`/projects/${encodeURIComponent(me.default_project_id)}`);
  }

  return (
    <>
      <RelayTopRail activeStep="Face" userLabel={me?.display_name ?? me?.email ?? "signed out"} />
      <main style={pageStyle}>
        <p style={eyebrowStyle}>{dictionary.onboarding.page.eyebrow}</p>
        <h1 style={pageTitleStyle}>{dictionary.onboarding.page.title}</h1>
        <p style={pageCopyStyle}>{dictionary.onboarding.page.subtitle}</p>
        {me ? (
          <OnboardingClient
            copy={dictionary.onboarding.client}
            locale={locale}
            userDisplayName={me.display_name ?? me.email}
          />
        ) : (
          <section style={signInPanelStyle} aria-labelledby="signin-title">
            <span style={glyphStyle} aria-hidden="true">
              ●
            </span>
            <div>
              <h2 id="signin-title" style={signInTitleStyle}>
                {dictionary.onboarding.page.signInTitle}
              </h2>
              <p style={signInCopyStyle}>{dictionary.onboarding.page.signInCopy}</p>
              <div style={signInActionsStyle}>
                <a href={authStartURL("github")} style={authButtonStyle}>
                  {dictionary.common.continueWithGitHub}
                </a>
              </div>
            </div>
          </section>
        )}
      </main>
    </>
  );
}

const pageStyle: React.CSSProperties = {
  maxWidth: "720px",
  margin: "40px auto",
  padding: "0 28px 96px",
};

const eyebrowStyle: React.CSSProperties = {
  margin: "0 0 8px",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  letterSpacing: "0.16em",
  textTransform: "uppercase",
  color: "var(--muted)",
};

const pageTitleStyle: React.CSSProperties = {
  margin: "0 0 14px",
  color: "var(--ink)",
  fontFamily: "var(--font-display)",
  fontWeight: 500,
  fontSize: "clamp(44px, 7vw, 64px)",
  lineHeight: 1,
  fontVariationSettings: '"opsz" 144, "SOFT" 50',
};

const pageCopyStyle: React.CSSProperties = {
  margin: "0 0 28px",
  color: "var(--ink-muted)",
  fontFamily: "var(--font-display)",
  fontSize: "19px",
  fontStyle: "italic",
  lineHeight: 1.5,
  fontVariationSettings: '"opsz" 48',
};

const signInPanelStyle: React.CSSProperties = {
  display: "flex",
  gap: "14px",
  alignItems: "flex-start",
  padding: "28px",
  border: "1px solid var(--border)",
  borderRadius: "12px",
  background: "var(--canvas-raised)",
};

const glyphStyle: React.CSSProperties = {
  color: "var(--success)",
  fontFamily: "var(--font-mono)",
  fontSize: "22px",
  lineHeight: 1,
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
