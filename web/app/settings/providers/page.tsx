import { cookies, headers } from "next/headers";

import { RELAY_API_URL } from "@/lib/api";
import { getDictionary, resolveLocale } from "@/lib/i18n";
import { listProviderCredentials } from "@/lib/provider-credentials";
import { RelayCard, RelayLinkButton, RelayPageHead, RelayTopRail } from "@/components/relay";

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
      <main className="relay-settings-page">
        <nav className="relay-settings-nav" aria-label="Settings navigation">
          <a href="/onboarding" className="relay-settings-nav-link">
            {dictionary.common.links.backToOnboarding}
          </a>
          <a href="/settings/api-keys" className="relay-settings-nav-link">
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
          <RelayCard className="relay-settings-fallback" variant="elevated">
            <RelayPageHead
              eyebrow={dictionary.providers.page.eyebrow}
              title={dictionary.providers.page.signInTitle}
              copy={dictionary.providers.page.signInCopy}
              actions={
                <RelayLinkButton href={signInURL()} variant="primary">
                  {dictionary.common.continueWithGitHub}
                </RelayLinkButton>
              }
            />
          </RelayCard>
        )}
      </main>
    </>
  );
}
