import { cookies, headers } from "next/headers";
import Link from "next/link";

import { RELAY_API_URL } from "@/lib/api";
import { getDictionary, resolveLocale, translateErrorMessage } from "@/lib/i18n";
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
      <main className="relay-settings-page">
        <nav className="relay-settings-nav" aria-label="Settings navigation">
          <Link href="/" className="relay-settings-nav-link">
            {dictionary.common.links.projectExplorer}
          </Link>
          <a href="/settings/providers" className="relay-settings-nav-link">
            {dictionary.common.links.providerSettings}
          </a>
        </nav>
        {authenticated ? (
          loadError ? (
            <RelayCard className="relay-settings-fallback" variant="elevated">
              <RelayPageHead
                eyebrow={dictionary.apiKeys.page.eyebrow}
                title={dictionary.apiKeys.page.loadErrorTitle}
                copy={loadError}
              />
            </RelayCard>
          ) : (
            <APIKeySettingsClient
              copy={dictionary.apiKeys.client}
              errorMap={dictionary.apiKeys.errorMap}
              initialKeys={initialKeys}
              locale={locale}
            />
          )
        ) : (
          <RelayCard className="relay-settings-fallback" variant="elevated">
            <RelayPageHead
              eyebrow={dictionary.apiKeys.page.eyebrow}
              title={dictionary.apiKeys.page.signInTitle}
              copy={dictionary.apiKeys.page.signInCopy}
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
