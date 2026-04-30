"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import type { Dictionary, Locale } from "@/lib/i18n";
import { translateErrorMessage } from "@/lib/i18n";
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
    <section style={surfaceStyle} aria-labelledby="provider-title">
      <header style={pageHeadStyle}>
        <p style={eyebrowStyle}>{copy.eyebrow}</p>
        <div style={pageHeadRowStyle}>
          <div>
            <h1 id="provider-title" style={titleStyle}>
              {copy.title}
            </h1>
            <p style={copyStyle}>{copy.copy}</p>
          </div>
          <p style={settingsOnlyStyle}>{copy.settingsOnlyPill}</p>
        </div>
      </header>

      <div style={settingsGridStyle}>
        <section style={cardStyle} aria-labelledby="provider-status-title">
          <div style={cardHeadStyle}>
            <p style={cardKickerStyle}>{copy.eyebrow}</p>
            <h2 id="provider-status-title" style={cardTitleStyle}>
              {copy.title}
            </h2>
          </div>

          <div style={statusRowStyle} aria-live="polite">
            <span style={credential?.connected ? connectedDotStyle : disconnectedDotStyle} />
            <div>
              <strong style={statusTitleStyle}>
                {credential?.connected ? copy.connected : copy.disconnected}
              </strong>
              <p style={statusCopyStyle}>
                {credential?.connected
                  ? `${credential.key_prefix ?? "sk-ant"} ${copy.maskedKeySeparator} ${credential.key_last4 ?? "••••"}`
                  : copy.noStoredKey}
              </p>
              <p style={statusHelpStyle}>{statusMessage}</p>
            </div>
          </div>

          {credential?.connected ? (
            <button
              type="button"
              disabled={busy}
              onClick={disconnect}
              style={dangerButtonStyle}
              data-testid="disconnect-provider"
            >
              {status === "disconnecting" ? copy.disconnectingButton : copy.disconnectButton}
            </button>
          ) : null}
        </section>

        <section style={cardStyle} aria-labelledby="provider-connect-title">
          <div style={cardHeadStyle}>
            <p style={cardKickerStyle}>{copy.settingsOnlyPill}</p>
            <h2 id="provider-connect-title" style={cardTitleStyle}>
              {credential?.connected ? copy.replaceButton : copy.connectButton}
            </h2>
          </div>

          <label style={labelStyle} htmlFor="anthropic-key">
            {copy.apiKeyLabel}
          </label>
          <input
            id="anthropic-key"
            type="password"
            value={apiKey}
            onChange={(event) => setAPIKey(event.target.value)}
            placeholder={copy.apiKeyPlaceholder}
            autoComplete="off"
            style={inputStyle}
          />
          <p style={fieldHelpStyle}>{copy.fieldHelp}</p>

          <div style={actionsStyle}>
            <button
              type="button"
              disabled={!canSave}
              onClick={connect}
              style={{
                ...primaryButtonStyle,
                opacity: canSave ? 1 : 0.55,
                cursor: canSave ? "pointer" : "not-allowed",
              }}
              data-testid="connect-provider"
            >
              {status === "saving"
                ? copy.savingButton
                : credential?.connected
                  ? copy.replaceButton
                  : copy.connectButton}
            </button>
          </div>

          {status === "error" && (
            <p role="alert" style={errorStyle}>
              {error}
            </p>
          )}
        </section>
      </div>
    </section>
  );
}

const surfaceStyle: React.CSSProperties = {
  width: "100%",
  maxWidth: "1120px",
};

const pageHeadStyle: React.CSSProperties = {
  marginBottom: "24px",
  paddingBottom: "22px",
  borderBottom: "1px solid var(--border)",
};

const pageHeadRowStyle: React.CSSProperties = {
  display: "flex",
  flexWrap: "wrap",
  justifyContent: "space-between",
  gap: "18px",
  alignItems: "end",
};

const settingsGridStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateColumns: "minmax(300px, 0.9fr) minmax(340px, 1.1fr)",
  gap: "18px",
  alignItems: "start",
};

const cardStyle: React.CSSProperties = {
  minWidth: 0,
  padding: "22px",
  border: "1px solid var(--border)",
  borderRadius: "12px",
  background: "var(--canvas-raised)",
  boxShadow: "0 18px 48px var(--halo)",
};

const cardHeadStyle: React.CSSProperties = {
  marginBottom: "20px",
};

const cardKickerStyle: React.CSSProperties = {
  margin: "0 0 8px",
  color: "var(--magic-primary-strong)",
  fontFamily: "var(--font-mono)",
  fontSize: "10px",
  letterSpacing: "0.16em",
  textTransform: "uppercase",
};

const cardTitleStyle: React.CSSProperties = {
  margin: 0,
  fontFamily: "var(--font-display)",
  fontSize: "28px",
  fontWeight: 600,
};

const eyebrowStyle: React.CSSProperties = {
  margin: "0 0 12px",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
  color: "var(--muted)",
};

const titleStyle: React.CSSProperties = {
  margin: "0 0 10px",
  fontFamily: "var(--font-display)",
  fontSize: "clamp(38px, 5vw, 64px)",
  fontWeight: 600,
  letterSpacing: "0",
  lineHeight: 0.95,
};

const copyStyle: React.CSSProperties = {
  maxWidth: "620px",
  margin: 0,
  color: "var(--ink-muted)",
  lineHeight: 1.6,
};

const settingsOnlyStyle: React.CSSProperties = {
  display: "inline-flex",
  minHeight: "32px",
  alignItems: "center",
  border: "1px solid var(--border-strong)",
  borderRadius: "6px",
  padding: "0 11px",
  margin: 0,
  color: "var(--ink-muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  letterSpacing: "0.08em",
  textTransform: "uppercase",
  background: "var(--canvas)",
};

const statusRowStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateColumns: "16px minmax(0, 1fr)",
  gap: "12px",
  alignItems: "start",
  padding: "16px",
  border: "1px solid var(--border)",
  borderRadius: "12px",
  marginBottom: "16px",
  background: "var(--canvas)",
};

const connectedDotStyle: React.CSSProperties = {
  width: "11px",
  height: "11px",
  marginTop: "6px",
  borderRadius: "50%",
  background: "var(--success)",
};

const disconnectedDotStyle: React.CSSProperties = {
  ...connectedDotStyle,
  background: "var(--muted)",
};

const statusTitleStyle: React.CSSProperties = {
  display: "block",
  fontSize: "14px",
};

const statusCopyStyle: React.CSSProperties = {
  margin: "3px 0 0",
  color: "var(--ink-muted)",
  fontSize: "13px",
};

const statusHelpStyle: React.CSSProperties = {
  margin: "8px 0 0",
  color: "var(--muted)",
  fontSize: "12px",
  lineHeight: 1.5,
};

const labelStyle: React.CSSProperties = {
  display: "block",
  marginBottom: "8px",
  color: "var(--ink)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  fontWeight: 700,
  letterSpacing: "0.12em",
  textTransform: "uppercase",
};

const inputStyle: React.CSSProperties = {
  width: "100%",
  minHeight: "44px",
  border: "1px solid var(--border-strong)",
  borderRadius: "8px",
  padding: "0 12px",
  background: "var(--canvas)",
  color: "var(--ink)",
  fontFamily: "var(--font-mono)",
  fontSize: "13px",
};

const fieldHelpStyle: React.CSSProperties = {
  margin: "8px 0 0",
  color: "var(--muted)",
  fontSize: "12px",
  lineHeight: 1.5,
};

const actionsStyle: React.CSSProperties = {
  display: "flex",
  flexWrap: "wrap",
  gap: "12px",
  marginTop: "16px",
};

const primaryButtonStyle: React.CSSProperties = {
  minHeight: "42px",
  border: 0,
  borderRadius: "8px",
  padding: "0 18px",
  background: "var(--ink)",
  color: "var(--canvas)",
  fontWeight: 800,
};

const secondaryButtonStyle: React.CSSProperties = {
  minHeight: "42px",
  border: "1px solid var(--border-strong)",
  borderRadius: "8px",
  padding: "0 18px",
  background: "var(--canvas)",
  color: "var(--ink)",
  fontWeight: 800,
  cursor: "pointer",
};

const dangerButtonStyle: React.CSSProperties = {
  ...secondaryButtonStyle,
  color: "var(--danger)",
};

const errorStyle: React.CSSProperties = {
  margin: "16px 0 0",
  color: "var(--danger)",
  fontSize: "13px",
};
