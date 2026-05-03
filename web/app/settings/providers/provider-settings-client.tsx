"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";

import { translateKnownError } from "@/lib/i18n";
import {
  RelayButton,
  RelayCard,
  RelayCardHeader,
  RelayCardKicker,
  RelayCardTitle,
  RelayFeedback,
  RelayField,
  RelayPageHead,
  RelayStatusBadge,
  RelayTextInput,
} from "@/components/relay";
import {
  connectProviderCredential,
  disconnectProviderCredential,
  type ProviderCredentialStatus,
} from "@/lib/provider-credentials";

type Props = {
  initialCredential?: ProviderCredentialStatus;
};

export function ProviderSettingsClient({ initialCredential }: Props) {
  const router = useRouter();
  const t = useTranslations("Settings.ProviderCredentials.client");
  const errors = useTranslations("Settings.ProviderCredentials.errorMap");
  const [credential, setCredential] = useState(initialCredential);
  const [apiKey, setAPIKey] = useState("");
  const [status, setStatus] = useState<"idle" | "saving" | "disconnecting" | "error">("idle");
  const [error, setError] = useState("");

  async function connect() {
    setStatus("saving");
    setError("");
    try {
      const saved = await connectProviderCredential(apiKey);
      setCredential(saved);
      setAPIKey("");
      setStatus("idle");
      router.refresh();
    } catch (err) {
      setStatus("error");
      setError(
        translateKnownError({
          error: err,
          fallback: t("fallbackConnectError"),
          knownErrors: providerErrorMap(errors),
        }),
      );
    }
  }

  async function disconnect() {
    setStatus("disconnecting");
    setError("");
    try {
      await disconnectProviderCredential();
      setCredential(undefined);
      setStatus("idle");
      router.refresh();
    } catch (err) {
      setStatus("error");
      setError(
        translateKnownError({
          error: err,
          fallback: t("fallbackDisconnectError"),
          knownErrors: providerErrorMap(errors),
        }),
      );
    }
  }

  const busy = status === "saving" || status === "disconnecting";
  const canSave = apiKey.trim().length > 0 && !busy;
  const statusMessage =
    status === "saving"
      ? t("savingStatus")
      : status === "disconnecting"
        ? t("disconnectingStatus")
        : credential?.connected
          ? t("connectedHelp")
          : t("disconnectedHelp");

  return (
    <section className="relay-settings-surface" aria-labelledby="provider-title">
      <RelayPageHead
        eyebrow={t("eyebrow")}
        title={t("title")}
        titleId="provider-title"
        copy={t("copy")}
        actions={<RelayStatusBadge>{t("settingsOnlyPill")}</RelayStatusBadge>}
      />

      <div className="relay-settings-grid">
        <RelayCard variant="elevated" aria-labelledby="provider-status-title">
          <RelayCardHeader>
            <RelayCardKicker>{t("eyebrow")}</RelayCardKicker>
            <RelayCardTitle id="provider-status-title">{t("title")}</RelayCardTitle>
          </RelayCardHeader>

          <div className="relay-credential-status" aria-live="polite">
            <span
              className="relay-status-dot"
              data-variant={credential?.connected ? "success" : "neutral"}
            />
            <div>
              <strong className="relay-status-title">
                {credential?.connected ? t("connected") : t("disconnected")}
              </strong>
              <p className="relay-status-copy">
                {credential?.connected
                  ? `${credential.key_prefix ?? "sk-ant"} ${t("maskedKeySeparator")} ${credential.key_last4 ?? "••••"}`
                  : t("noStoredKey")}
              </p>
              <p className="relay-status-help">{statusMessage}</p>
            </div>
          </div>

          {credential?.connected ? (
            <RelayButton
              disabled={busy}
              onClick={disconnect}
              variant="danger"
              data-testid="disconnect-provider"
            >
              {status === "disconnecting" ? t("disconnectingButton") : t("disconnectButton")}
            </RelayButton>
          ) : null}
        </RelayCard>

        <RelayCard variant="elevated" aria-labelledby="provider-connect-title">
          <RelayCardHeader>
            <RelayCardKicker>{t("settingsOnlyPill")}</RelayCardKicker>
            <RelayCardTitle id="provider-connect-title">
              {credential?.connected ? t("replaceButton") : t("connectButton")}
            </RelayCardTitle>
          </RelayCardHeader>

          <RelayField label={t("apiKeyLabel")} htmlFor="anthropic-key" help={t("fieldHelp")}>
            <RelayTextInput
              id="anthropic-key"
              type="password"
              value={apiKey}
              onChange={(event) => setAPIKey(event.target.value)}
              placeholder={t("apiKeyPlaceholder")}
              autoComplete="off"
            />
          </RelayField>

          <div className="relay-form-actions">
            <RelayButton
              disabled={!canSave}
              onClick={connect}
              data-testid="connect-provider"
            >
              {status === "saving"
                ? t("savingButton")
                : credential?.connected
                  ? t("replaceButton")
                  : t("connectButton")}
            </RelayButton>
          </div>

          {status === "error" && (
            <RelayFeedback role="alert" variant="error">
              {error}
            </RelayFeedback>
          )}
        </RelayCard>
      </div>
    </section>
  );
}

function providerErrorMap(t: ReturnType<typeof useTranslations>): Record<string, string> {
  return {
    UNAUTHENTICATED: t("UNAUTHENTICATED"),
    INVALID_INPUT: t("INVALID_INPUT"),
  };
}
