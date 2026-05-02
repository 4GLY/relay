import { cookies, headers } from "next/headers";
import { redirect } from "next/navigation";

import { RELAY_API_URL, relayFetch, type RelayEnvelope } from "@/lib/api";
import { getDictionary, resolveLocale } from "@/lib/i18n";
import type { AuthMe } from "@/lib/onboarding";
import {
  RelayCard,
  RelayLanguageSwitch,
  RelayLinkButton,
  RelayPageHead,
  RelayTopRail,
} from "@/components/relay";

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
      <main className="relay-onboarding-page">
        <RelayPageHead
          eyebrow={dictionary.onboarding.page.eyebrow}
          title={dictionary.onboarding.page.title}
          copy={dictionary.onboarding.page.subtitle}
        />
        {me ? (
          <OnboardingClient
            copy={dictionary.onboarding.client}
            locale={locale}
            userDisplayName={me.display_name ?? me.email}
          />
        ) : (
          <RelayCard className="relay-onboarding-signin" aria-labelledby="signin-title">
            <span className="relay-onboarding-glyph" aria-hidden="true">
              ●
            </span>
            <div>
              <h2 id="signin-title" className="relay-auth-panel-title">
                {dictionary.onboarding.page.signInTitle}
              </h2>
              <p className="relay-auth-panel-copy">{dictionary.onboarding.page.signInCopy}</p>
              <div className="relay-form-actions">
                <RelayLinkButton href={authStartURL("github")} variant="secondary">
                  {dictionary.common.continueWithGitHub}
                </RelayLinkButton>
              </div>
              <div className="relay-auth-language">
                <RelayLanguageSwitch />
              </div>
            </div>
          </RelayCard>
        )}
      </main>
    </>
  );
}
