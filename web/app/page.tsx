import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { getTranslations } from "next-intl/server";

import { RELAY_API_URL, relayFetch, type RelayEnvelope } from "@/lib/api";
import type { AuthMe } from "@/lib/onboarding";
import { RelayCard, RelayLanguageSwitch, RelayLinkButton } from "@/components/relay";

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
  const t = await getTranslations("Root");

  if (me?.onboarding_complete && me.default_project_id) {
    redirect(`/projects/${encodeURIComponent(me.default_project_id)}`);
  }

  if (me) {
    redirect("/onboarding");
  }

  return (
    <main className="relay-auth-entry">
      <p className="relay-auth-eyebrow">{t("eyebrow")}</p>
      <h1 className="relay-auth-title">{t("title")}</h1>
      <p className="relay-auth-subtitle">{t("subtitle")}</p>
      <RelayCard className="relay-auth-panel" aria-labelledby="entry-title">
        <h2 id="entry-title" className="relay-auth-panel-title">
          {t("panelTitle")}
        </h2>
        <p className="relay-auth-panel-copy">{t("panelCopy")}</p>
        <RelayLinkButton href={authStartURL()} variant="primary">
          {t("signInButton")}
        </RelayLinkButton>
      </RelayCard>
      <div className="relay-auth-language">
        <RelayLanguageSwitch />
      </div>
    </main>
  );
}
