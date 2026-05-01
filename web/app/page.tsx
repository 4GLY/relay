import { cookies, headers } from "next/headers";
import { redirect } from "next/navigation";

import { RELAY_API_URL, relayFetch, type RelayEnvelope } from "@/lib/api";
import { getDictionary, resolveLocale } from "@/lib/i18n";
import type { AuthMe } from "@/lib/onboarding";
import { RelayCard, RelayLinkButton } from "@/components/relay";

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
    <main className="relay-auth-entry">
      <p className="relay-auth-eyebrow">{dictionary.root.eyebrow}</p>
      <h1 className="relay-auth-title">{dictionary.root.title}</h1>
      <p className="relay-auth-subtitle">{dictionary.root.subtitle}</p>
      <RelayCard className="relay-auth-panel" aria-labelledby="entry-title">
        <h2 id="entry-title" className="relay-auth-panel-title">
          {dictionary.root.panelTitle}
        </h2>
        <p className="relay-auth-panel-copy">{dictionary.root.panelCopy}</p>
        <RelayLinkButton href={authStartURL()} variant="primary">
          {dictionary.root.signInButton}
        </RelayLinkButton>
      </RelayCard>
    </main>
  );
}
