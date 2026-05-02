import { cookies } from "next/headers";
import { getTranslations } from "next-intl/server";

import { RELAY_API_URL } from "@/lib/api";
import { translateKnownError } from "@/lib/i18n";
import {
  listProviderCredentials,
  ProviderCredentialAPIError,
} from "@/lib/provider-credentials";
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
  const cookieHeader = cookieStore.toString();
  const t = await getTranslations("Settings.ProviderCredentials.page");
  const common = await getTranslations("Common");
  const errors = await getTranslations("Settings.ProviderCredentials.errorMap");
  let initialCredential;
  let authenticated = true;
  let loadError = "";

  try {
    const result = await listProviderCredentials({ cookie: cookieHeader });
    initialCredential = result.credentials.find((item) => item.provider === "anthropic");
  } catch (error) {
    if (error instanceof ProviderCredentialAPIError && error.code === "UNAUTHENTICATED") {
      authenticated = false;
    } else {
      loadError = translateKnownError({
        error,
        fallback: t("loadErrorCopy"),
        knownErrors: providerErrorMap(errors),
      });
    }
  }

  return (
    <>
      <RelayTopRail activeStep="Transform" />
      <main className="relay-settings-page">
        <nav className="relay-settings-nav" aria-label={common("settingsNavigation")}>
          <a href="/onboarding" className="relay-settings-nav-link">
            {common("links.backToOnboarding")}
          </a>
          <a href="/settings/api-keys" className="relay-settings-nav-link">
            {common("links.apiKeys")}
          </a>
        </nav>
        {authenticated ? (
          loadError ? (
            <RelayCard className="relay-settings-fallback" variant="elevated">
              <RelayPageHead
                eyebrow={t("eyebrow")}
                title={t("loadErrorTitle")}
                copy={loadError}
              />
            </RelayCard>
          ) : (
            <ProviderSettingsClient initialCredential={initialCredential} />
          )
        ) : (
          <RelayCard className="relay-settings-fallback" variant="elevated">
            <RelayPageHead
              eyebrow={t("eyebrow")}
              title={t("signInTitle")}
              copy={t("signInCopy")}
              actions={
                <RelayLinkButton href={signInURL()} variant="primary">
                  {common("continueWithGitHub")}
                </RelayLinkButton>
              }
            />
          </RelayCard>
        )}
      </main>
    </>
  );
}

function providerErrorMap(t: Awaited<ReturnType<typeof getTranslations>>): Record<string, string> {
  return {
    UNAUTHENTICATED: t("UNAUTHENTICATED"),
    INVALID_INPUT: t("INVALID_INPUT"),
  };
}
