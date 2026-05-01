"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import type { Dictionary, Locale } from "@/lib/i18n";
import { translateErrorMessage } from "@/lib/i18n";
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
  copy: Dictionary["providers"]["client"];
  errorMap: Record<string, string>;
  initialCredential?: ProviderCredentialStatus;
  locale: Locale;
};

export function ProviderSettingsClient({ copy, errorMap, initialCredential, locale }: Props) {
  const router = useRouter();
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
        translateErrorMessage({
          error: err,
          fallback: copy.fallbackConnectError,
          knownErrors: errorMap,
          locale,
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
        translateErrorMessage({
          error: err,
          fallback: copy.fallbackDisconnectError,
          knownErrors: errorMap,
          locale,
        }),
      );
    }
  }

  const busy = status === "saving" || status === "disconnecting";
  const canSave = apiKey.trim().length > 0 && !busy;
  const statusMessage =
    status === "saving"
      ? copy.savingStatus
      : status === "disconnecting"
        ? copy.disconnectingStatus
        : credential?.connected
          ? copy.connectedHelp
          : copy.disconnectedHelp;

  return (
    <section className="relay-settings-surface" aria-labelledby="provider-title">
      <RelayPageHead
        eyebrow={copy.eyebrow}
        title={copy.title}
        titleId="provider-title"
        copy={copy.copy}
        actions={<RelayStatusBadge>{copy.settingsOnlyPill}</RelayStatusBadge>}
      />

      <div className="relay-settings-grid">
        <RelayCard variant="elevated" aria-labelledby="provider-status-title">
          <RelayCardHeader>
            <RelayCardKicker>{copy.eyebrow}</RelayCardKicker>
            <RelayCardTitle id="provider-status-title">{copy.title}</RelayCardTitle>
          </RelayCardHeader>

          <div className="relay-credential-status" aria-live="polite">
            <span
              className="relay-status-dot"
              data-variant={credential?.connected ? "success" : "neutral"}
            />
            <div>
              <strong className="relay-status-title">
                {credential?.connected ? copy.connected : copy.disconnected}
              </strong>
              <p className="relay-status-copy">
                {credential?.connected
                  ? `${credential.key_prefix ?? "sk-ant"} ${copy.maskedKeySeparator} ${credential.key_last4 ?? "••••"}`
                  : copy.noStoredKey}
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
              {status === "disconnecting" ? copy.disconnectingButton : copy.disconnectButton}
            </RelayButton>
          ) : null}
        </RelayCard>

        <RelayCard variant="elevated" aria-labelledby="provider-connect-title">
          <RelayCardHeader>
            <RelayCardKicker>{copy.settingsOnlyPill}</RelayCardKicker>
            <RelayCardTitle id="provider-connect-title">
              {credential?.connected ? copy.replaceButton : copy.connectButton}
            </RelayCardTitle>
          </RelayCardHeader>

          <RelayField label={copy.apiKeyLabel} htmlFor="anthropic-key" help={copy.fieldHelp}>
            <RelayTextInput
              id="anthropic-key"
              type="password"
              value={apiKey}
              onChange={(event) => setAPIKey(event.target.value)}
              placeholder={copy.apiKeyPlaceholder}
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
                ? copy.savingButton
                : credential?.connected
                  ? copy.replaceButton
                  : copy.connectButton}
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
