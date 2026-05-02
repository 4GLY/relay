import { cookies } from "next/headers";
import Link from "next/link";
import { getTranslations } from "next-intl/server";

import { RELAY_API_URL } from "@/lib/api";
import { translateKnownError } from "@/lib/i18n";
import {
  listUserAPIKeys,
  RelayAPIError,
  type UserAPIKeySummary,
} from "@/lib/user-api-keys";
import { RelayCard, RelayLinkButton, RelayPageHead, RelayTopRail } from "@/components/relay";

import { APIKeySettingsClient } from "./api-key-settings-client";

export const dynamic = "force-dynamic";

function signInURL() {
  const url = new URL("/v1/auth/github/start", RELAY_API_URL);
  url.searchParams.set("redirect_to", "/settings/api-keys");
  return url.toString();
}

export default async function APIKeySettingsPage() {
  const cookieStore = await cookies();
  const cookieHeader = cookieStore.toString();
  const t = await getTranslations("Settings.ApiKeys.page");
  const common = await getTranslations("Common");
  const errors = await getTranslations("Settings.ApiKeys.errorMap");

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
      loadError = translateKnownError({
        error,
        fallback: t("loadErrorCopy"),
        knownErrors: apiKeyErrorMap(errors),
      });
    }
  }

  return (
    <>
      <RelayTopRail activeStep="Transform" />
      <main className="relay-settings-page">
        <nav className="relay-settings-nav" aria-label={common("settingsNavigation")}>
          <Link href="/" className="relay-settings-nav-link">
            {common("links.projectExplorer")}
          </Link>
          <a href="/settings/providers" className="relay-settings-nav-link">
            {common("links.providerSettings")}
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
            <APIKeySettingsClient initialKeys={initialKeys} />
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

function apiKeyErrorMap(t: Awaited<ReturnType<typeof getTranslations>>): Record<string, string> {
  return {
    UNAUTHENTICATED: t("UNAUTHENTICATED"),
    INVALID_INPUT: t("INVALID_INPUT"),
    API_KEY_NOT_FOUND_BY_ID: t("API_KEY_NOT_FOUND_BY_ID"),
  };
}
