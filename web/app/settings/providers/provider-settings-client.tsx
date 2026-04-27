"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

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
      setError(err instanceof Error ? err.message : "Could not connect provider.");
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
      setError(err instanceof Error ? err.message : "Could not disconnect provider.");
    }
  }

  const busy = status === "saving" || status === "disconnecting";
  const canSave = apiKey.trim().length > 0 && !busy;

  return (
    <section style={panelStyle} aria-labelledby="provider-title">
      <div>
        <p style={eyebrowStyle}>Settings · provider credentials</p>
        <h1 id="provider-title" style={titleStyle}>
          Claude provider
        </h1>
        <p style={copyStyle}>
          Connect Anthropic only when Claude-backed features need it. This key is not
          part of first-run onboarding.
        </p>
      </div>

      <div style={statusRowStyle}>
        <span style={credential?.connected ? connectedDotStyle : disconnectedDotStyle} />
        <div>
          <strong style={statusTitleStyle}>
            {credential?.connected ? "Connected" : "Not connected"}
          </strong>
          <p style={statusCopyStyle}>
            {credential?.connected
              ? `${credential.key_prefix ?? "sk-ant"} ending ${credential.key_last4 ?? "••••"}`
              : "No provider key is stored for this user."}
          </p>
        </div>
      </div>

      <label style={labelStyle} htmlFor="anthropic-key">
        Anthropic API key
      </label>
      <input
        id="anthropic-key"
        type="password"
        value={apiKey}
        onChange={(event) => setAPIKey(event.target.value)}
        placeholder="sk-ant-..."
        autoComplete="off"
        style={inputStyle}
      />

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
          {status === "saving" ? "Saving..." : credential?.connected ? "Replace key" : "Connect key"}
        </button>
        {credential?.connected && (
          <button
            type="button"
            disabled={busy}
            onClick={disconnect}
            style={secondaryButtonStyle}
            data-testid="disconnect-provider"
          >
            {status === "disconnecting" ? "Disconnecting..." : "Disconnect"}
          </button>
        )}
      </div>

      {status === "error" && (
        <p role="alert" style={errorStyle}>
          {error}
        </p>
      )}
    </section>
  );
}

const panelStyle: React.CSSProperties = {
  maxWidth: "720px",
  padding: "30px",
  border: "1px solid var(--border)",
  borderRadius: "8px",
  background: "var(--canvas-raised)",
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
  margin: "0 0 14px",
  fontFamily: "var(--font-display)",
  fontSize: "42px",
  fontWeight: 500,
  letterSpacing: "-0.02em",
};

const copyStyle: React.CSSProperties = {
  maxWidth: "560px",
  margin: "0 0 24px",
  color: "var(--ink-muted)",
  lineHeight: 1.6,
};

const statusRowStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateColumns: "16px minmax(0, 1fr)",
  gap: "12px",
  alignItems: "start",
  padding: "16px",
  border: "1px solid var(--border)",
  borderRadius: "8px",
  marginBottom: "22px",
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

const labelStyle: React.CSSProperties = {
  display: "block",
  marginBottom: "8px",
  fontSize: "13px",
  fontWeight: 800,
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
  color: "var(--danger)",
  fontWeight: 800,
  cursor: "pointer",
};

const errorStyle: React.CSSProperties = {
  margin: "16px 0 0",
  color: "var(--danger)",
  fontSize: "13px",
};
