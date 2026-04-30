"use client";

import { useState } from "react";

import type { Dictionary, Locale } from "@/lib/i18n";
import { translateErrorMessage } from "@/lib/i18n";
import {
  issueUserAPIKey,
  revokeUserAPIKey,
  type UserAPIKeySummary,
} from "@/lib/user-api-keys";

type Props = {
  copy: Dictionary["apiKeys"]["client"];
  errorMap: Record<string, string>;
  initialKeys: UserAPIKeySummary[];
  locale: Locale;
};

type FeedbackState =
  | { kind: "idle"; message: string }
  | { kind: "success"; message: string }
  | { kind: "error"; message: string };

export function APIKeySettingsClient({ copy, errorMap, initialKeys, locale }: Props) {
  const [keys, setKeys] = useState(initialKeys);
  const [name, setName] = useState("");
  const [confirmingKeyID, setConfirmingKeyID] = useState<string | null>(null);
  const [issuedToken, setIssuedToken] = useState<{
    name: string;
    token: string;
    tokenPrefix: string;
  } | null>(null);
  const [isIssuing, setIsIssuing] = useState(false);
  const [revokingKeyID, setRevokingKeyID] = useState<string | null>(null);
  const [copyStatus, setCopyStatus] = useState<"idle" | "copied" | "error">("idle");
  const [feedback, setFeedback] = useState<FeedbackState>({ kind: "idle", message: "" });

  async function issueKey() {
    setIsIssuing(true);
    setFeedback({ kind: "idle", message: "" });
    setCopyStatus("idle");

    try {
      const issued = await issueUserAPIKey(name.trim());
      setIssuedToken({
        name: issued.name,
        token: issued.token,
        tokenPrefix: issued.token_prefix,
      });
      setKeys((current) => [
        {
          key_id: issued.key_id,
          name: issued.name,
          token_prefix: issued.token_prefix,
          scope: issued.scope,
          project_id: issued.project_id,
          revoked: false,
        },
        ...current,
      ]);
      setName("");
      setFeedback({ kind: "success", message: copy.issuedSuccess });
    } catch (error) {
      setFeedback({
        kind: "error",
        message: translateErrorMessage({
          error,
          fallback: copy.fallbackIssueError,
          knownErrors: errorMap,
          locale,
        }),
      });
    } finally {
      setIsIssuing(false);
    }
  }

  async function revokeKey(keyID: string) {
    setRevokingKeyID(keyID);
    setFeedback({ kind: "idle", message: "" });

    try {
      const revoked = await revokeUserAPIKey(keyID);
      setKeys((current) =>
        current.map((item) =>
          item.key_id === keyID
            ? {
                ...item,
                revoked: revoked.revoked,
              }
            : item,
        ),
      );
      setConfirmingKeyID(null);
      setFeedback({ kind: "success", message: copy.revokedSuccess });
    } catch (error) {
      setFeedback({
        kind: "error",
        message: translateErrorMessage({
          error,
          fallback: copy.fallbackRevokeError,
          knownErrors: errorMap,
          locale,
        }),
      });
    } finally {
      setRevokingKeyID(null);
    }
  }

  async function copyIssuedToken() {
    if (!issuedToken) return;
    try {
      await navigator.clipboard.writeText(issuedToken.token);
      setCopyStatus("copied");
    } catch {
      setCopyStatus("error");
    }
  }

  const canIssue = name.trim().length > 0 && !isIssuing && !revokingKeyID;

  return (
    <section style={panelStyle} aria-labelledby="api-key-settings-title">
      <div>
        <p style={eyebrowStyle}>{copy.eyebrow}</p>
        <h1 id="api-key-settings-title" style={titleStyle}>
          {copy.title}
        </h1>
        <p style={copyStyle}>{copy.copy}</p>
        <p style={settingsOnlyStyle}>{copy.settingsOnlyPill}</p>
      </div>

      <label style={labelStyle} htmlFor="api-key-name">
        {copy.nameLabel}
      </label>
      <input
        id="api-key-name"
        type="text"
        value={name}
        onChange={(event) => setName(event.target.value)}
        placeholder={copy.namePlaceholder}
        autoComplete="off"
        style={inputStyle}
      />
      <p style={fieldHelpStyle}>{copy.fieldHelp}</p>

      <div style={actionsStyle}>
        <button
          type="button"
          disabled={!canIssue}
          onClick={issueKey}
          style={{
            ...primaryButtonStyle,
            opacity: canIssue ? 1 : 0.55,
            cursor: canIssue ? "pointer" : "not-allowed",
          }}
          data-testid="issue-api-key"
        >
          {isIssuing ? copy.issuingButton : copy.issueButton}
        </button>
      </div>

      {issuedToken && (
        <section style={tokenPanelStyle} aria-live="polite">
          <div>
            <strong style={tokenTitleStyle}>{copy.tokenPanelTitle}</strong>
            <p style={tokenCopyStyle}>
              {copy.tokenPanelCopy} {issuedToken.name} · {issuedToken.tokenPrefix}
            </p>
          </div>
          <label style={labelStyle} htmlFor="issued-api-key-token">
            {copy.tokenLabel}
          </label>
          <div style={tokenRowStyle}>
            <input
              id="issued-api-key-token"
              type="text"
              readOnly
              value={issuedToken.token}
              autoComplete="off"
              spellCheck={false}
              style={tokenInputStyle}
            />
            <button
              type="button"
              onClick={copyIssuedToken}
              style={secondaryButtonStyle}
              data-testid="copy-issued-api-key"
            >
              {copyStatus === "copied" ? copy.copiedButton : copy.copyButton}
            </button>
          </div>
          {copyStatus === "error" && (
            <p role="alert" style={errorStyle}>
              {copy.copyError}
            </p>
          )}
        </section>
      )}

      {feedback.message && (
        <p
          aria-live="polite"
          role={feedback.kind === "error" ? "alert" : "status"}
          style={feedback.kind === "error" ? errorStyle : successStyle}
        >
          {feedback.message}
        </p>
      )}

      <section style={listSectionStyle} aria-labelledby="issued-keys-title">
        <div style={listHeaderStyle}>
          <h2 id="issued-keys-title" style={listTitleStyle}>
            {copy.listTitle}
          </h2>
        </div>

        {keys.length === 0 ? (
          <p style={emptyStateStyle}>{copy.emptyState}</p>
        ) : (
          <ul style={listStyle}>
            {keys.map((item) => {
              const isRevoking = revokingKeyID === item.key_id;
              const isConfirming = confirmingKeyID === item.key_id;
              const scopeLabel =
                item.scope === "project" ? copy.scopeProject : copy.scopeGlobal;

              return (
                <li key={item.key_id} style={rowStyle}>
                  <div style={rowTopStyle}>
                    <div>
                      <strong style={rowTitleStyle}>{item.name}</strong>
                      <p style={rowMetaStyle}>{item.token_prefix}</p>
                    </div>
                    <span style={item.revoked ? revokedBadgeStyle : activeBadgeStyle}>
                      {item.revoked ? copy.revokedStatus : copy.activeStatus}
                    </span>
                  </div>

                  <dl style={metaGridStyle}>
                    <div>
                      <dt style={metaLabelStyle}>{copy.scopeLabel}</dt>
                      <dd style={metaValueStyle}>{scopeLabel}</dd>
                    </div>
                    {item.project_id ? (
                      <div>
                        <dt style={metaLabelStyle}>{copy.projectLabel}</dt>
                        <dd style={metaValueStyle}>{item.project_id}</dd>
                      </div>
                    ) : null}
                  </dl>

                  {!item.revoked && (
                    <div style={rowActionsStyle}>
                      {isConfirming ? (
                        <>
                          <p style={confirmCopyStyle}>{copy.revokeConfirmCopy}</p>
                          <div style={confirmActionsStyle}>
                            <button
                              type="button"
                              onClick={() => revokeKey(item.key_id)}
                              disabled={Boolean(revokingKeyID)}
                              style={dangerButtonStyle}
                              data-testid={`confirm-revoke-${item.key_id}`}
                            >
                              {isRevoking ? copy.revokingButton : copy.confirmRevokeButton}
                            </button>
                            <button
                              type="button"
                              onClick={() => setConfirmingKeyID(null)}
                              disabled={Boolean(revokingKeyID)}
                              style={secondaryButtonStyle}
                            >
                              {copy.cancelRevokeButton}
                            </button>
                          </div>
                        </>
                      ) : (
                        <button
                          type="button"
                          onClick={() => setConfirmingKeyID(item.key_id)}
                          style={secondaryButtonStyle}
                          data-testid={`revoke-api-key-${item.key_id}`}
                        >
                          {copy.revokeButton}
                        </button>
                      )}
                    </div>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </section>
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

const settingsOnlyStyle: React.CSSProperties = {
  display: "inline-flex",
  minHeight: "28px",
  alignItems: "center",
  border: "1px solid var(--border)",
  borderRadius: "999px",
  padding: "0 10px",
  margin: "0 0 24px",
  color: "var(--ink-muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  letterSpacing: "0.08em",
  textTransform: "uppercase",
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
  fontFamily: "var(--font-sans)",
  fontSize: "14px",
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

const tokenPanelStyle: React.CSSProperties = {
  marginTop: "20px",
  padding: "18px",
  border: "1px solid var(--border-strong)",
  borderRadius: "8px",
  background: "var(--canvas)",
};

const tokenTitleStyle: React.CSSProperties = {
  display: "block",
  marginBottom: "6px",
  fontSize: "14px",
};

const tokenCopyStyle: React.CSSProperties = {
  margin: "0 0 14px",
  color: "var(--ink-muted)",
  fontSize: "13px",
  lineHeight: 1.5,
};

const tokenRowStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateColumns: "minmax(0, 1fr) auto",
  gap: "12px",
  alignItems: "center",
};

const tokenInputStyle: React.CSSProperties = {
  width: "100%",
  minHeight: "44px",
  border: "1px solid var(--border-strong)",
  borderRadius: "8px",
  padding: "0 12px",
  background: "var(--canvas-raised)",
  color: "var(--ink)",
  fontFamily: "var(--font-mono)",
  fontSize: "12px",
};

const successStyle: React.CSSProperties = {
  margin: "18px 0 0",
  color: "var(--success)",
  fontSize: "13px",
};

const errorStyle: React.CSSProperties = {
  margin: "18px 0 0",
  color: "var(--danger)",
  fontSize: "13px",
};

const listSectionStyle: React.CSSProperties = {
  marginTop: "28px",
  paddingTop: "24px",
  borderTop: "1px solid var(--border)",
};

const listHeaderStyle: React.CSSProperties = {
  marginBottom: "14px",
};

const listTitleStyle: React.CSSProperties = {
  margin: 0,
  fontFamily: "var(--font-display)",
  fontWeight: 500,
  fontSize: "28px",
};

const emptyStateStyle: React.CSSProperties = {
  margin: 0,
  color: "var(--ink-muted)",
  lineHeight: 1.6,
};

const listStyle: React.CSSProperties = {
  listStyle: "none",
  padding: 0,
  margin: 0,
  display: "grid",
  gap: "14px",
};

const rowStyle: React.CSSProperties = {
  padding: "18px",
  border: "1px solid var(--border)",
  borderRadius: "8px",
  background: "var(--canvas)",
};

const rowTopStyle: React.CSSProperties = {
  display: "flex",
  flexWrap: "wrap",
  justifyContent: "space-between",
  gap: "12px",
  alignItems: "start",
};

const rowTitleStyle: React.CSSProperties = {
  display: "block",
  fontSize: "15px",
};

const rowMetaStyle: React.CSSProperties = {
  margin: "4px 0 0",
  color: "var(--ink-muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "12px",
};

const badgeBaseStyle: React.CSSProperties = {
  display: "inline-flex",
  alignItems: "center",
  minHeight: "26px",
  padding: "0 10px",
  borderRadius: "999px",
  fontSize: "11px",
  fontWeight: 800,
  letterSpacing: "0.08em",
  textTransform: "uppercase",
};

const activeBadgeStyle: React.CSSProperties = {
  ...badgeBaseStyle,
  background: "color-mix(in oklab, var(--success) 14%, var(--canvas-raised))",
  color: "var(--success)",
};

const revokedBadgeStyle: React.CSSProperties = {
  ...badgeBaseStyle,
  background: "color-mix(in oklab, var(--muted) 16%, var(--canvas-raised))",
  color: "var(--ink-muted)",
};

const metaGridStyle: React.CSSProperties = {
  display: "flex",
  flexWrap: "wrap",
  gap: "18px",
  margin: "16px 0 0",
};

const metaLabelStyle: React.CSSProperties = {
  color: "var(--muted)",
  fontSize: "11px",
  letterSpacing: "0.08em",
  textTransform: "uppercase",
  marginBottom: "4px",
};

const metaValueStyle: React.CSSProperties = {
  margin: 0,
  fontSize: "13px",
};

const rowActionsStyle: React.CSSProperties = {
  marginTop: "16px",
};

const confirmCopyStyle: React.CSSProperties = {
  margin: "0 0 10px",
  color: "var(--ink-muted)",
  fontSize: "12px",
};

const confirmActionsStyle: React.CSSProperties = {
  display: "flex",
  flexWrap: "wrap",
  gap: "10px",
};
