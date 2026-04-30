import { cookies, headers } from "next/headers";
import { redirect } from "next/navigation";

import { RELAY_API_URL, relayFetch, type RelayEnvelope } from "@/lib/api";
import { getDictionary, resolveLocale } from "@/lib/i18n";
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

  if (me) {
    redirect("/onboarding");
  }

  return (
    <main style={pageStyle}>
      <p style={eyebrowStyle}>{dictionary.root.eyebrow}</p>
      <h1 style={titleStyle}>{dictionary.root.title}</h1>
      <p style={subtitleStyle}>{dictionary.root.subtitle}</p>
      <section style={panelStyle} aria-labelledby="entry-title">
        <h2 id="entry-title" style={panelTitleStyle}>
          {dictionary.root.panelTitle}
        </h2>
        <p style={panelCopyStyle}>{dictionary.root.panelCopy}</p>
        <a href={authStartURL()} style={buttonStyle}>
          {dictionary.root.signInButton}
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
