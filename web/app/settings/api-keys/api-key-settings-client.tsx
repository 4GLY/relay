"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";

import { translateKnownError } from "@/lib/i18n";
import {
  RelayButton,
  RelayCard,
  RelayCardHeader,
  RelayCardKicker,
  RelayCardTitle,
  RelayEmptyState,
  RelayFeedback,
  RelayField,
  RelayMetaGrid,
  RelayPageHead,
  RelayStatusBadge,
  RelayTextInput,
} from "@/components/relay";
import {
  issueUserAPIKey,
  revokeUserAPIKey,
  type UserAPIKeySummary,
} from "@/lib/user-api-keys";

type Props = {
  initialKeys: UserAPIKeySummary[];
};

type FeedbackState =
  | { kind: "idle"; message: string }
  | { kind: "success"; message: string }
  | { kind: "error"; message: string };

export function APIKeySettingsClient({ initialKeys }: Props) {
  const t = useTranslations("Settings.ApiKeys.client");
  const errors = useTranslations("Settings.ApiKeys.errorMap");
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
      setFeedback({ kind: "success", message: t("issuedSuccess") });
    } catch (error) {
      setFeedback({
        kind: "error",
        message: translateKnownError({
          error,
          fallback: t("fallbackIssueError"),
          knownErrors: apiKeyErrorMap(errors),
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
      setFeedback({ kind: "success", message: t("revokedSuccess") });
    } catch (error) {
      setFeedback({
        kind: "error",
        message: translateKnownError({
          error,
          fallback: t("fallbackRevokeError"),
          knownErrors: apiKeyErrorMap(errors),
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
    <section className="relay-settings-surface" aria-labelledby="api-key-settings-title">
      <RelayPageHead
        eyebrow={t("eyebrow")}
        title={t("title")}
        titleId="api-key-settings-title"
        copy={t("copy")}
        actions={<RelayStatusBadge>{t("settingsOnlyPill")}</RelayStatusBadge>}
      />

      <div className="relay-settings-grid">
        <RelayCard variant="elevated" aria-labelledby="issue-api-key-title">
          <RelayCardHeader>
            <RelayCardKicker>{t("settingsOnlyPill")}</RelayCardKicker>
            <RelayCardTitle id="issue-api-key-title">{t("issueButton")}</RelayCardTitle>
          </RelayCardHeader>

          <RelayField label={t("nameLabel")} htmlFor="api-key-name" help={t("fieldHelp")}>
            <RelayTextInput
              id="api-key-name"
              type="text"
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder={t("namePlaceholder")}
              autoComplete="off"
            />
          </RelayField>

          <div className="relay-form-actions">
            <RelayButton
              disabled={!canIssue}
              onClick={issueKey}
              data-testid="issue-api-key"
            >
              {isIssuing ? t("issuingButton") : t("issueButton")}
            </RelayButton>
          </div>

          {issuedToken && (
            <section className="relay-token-panel" aria-live="polite">
              <div>
                <strong className="relay-token-title">{t("tokenPanelTitle")}</strong>
                <p className="relay-token-copy">
                  {t("tokenPanelCopy")} {issuedToken.name} · {issuedToken.tokenPrefix}
                </p>
              </div>
              <RelayField label={t("tokenLabel")} htmlFor="issued-api-key-token">
                <div className="relay-token-row">
                  <RelayTextInput
                    id="issued-api-key-token"
                    type="text"
                    readOnly
                    value={issuedToken.token}
                    autoComplete="off"
                    spellCheck={false}
                  />
                  <RelayButton
                    onClick={copyIssuedToken}
                    variant="secondary"
                    data-testid="copy-issued-api-key"
                  >
                    {copyStatus === "copied" ? t("copiedButton") : t("copyButton")}
                  </RelayButton>
                </div>
              </RelayField>
              {copyStatus === "error" && (
                <RelayFeedback role="alert" variant="error">
                  {t("copyError")}
                </RelayFeedback>
              )}
            </section>
          )}

          {feedback.message && (
            <RelayFeedback
              aria-live="polite"
              role={feedback.kind === "error" ? "alert" : "status"}
              variant={feedback.kind === "error" ? "error" : "success"}
            >
              {feedback.message}
            </RelayFeedback>
          )}
        </RelayCard>

        <RelayCard variant="elevated" aria-labelledby="issued-keys-title">
          <div className="relay-list-header">
            <div>
              <RelayCardKicker>{t("scopeLabel")}</RelayCardKicker>
              <RelayCardTitle id="issued-keys-title">{t("listTitle")}</RelayCardTitle>
            </div>
            <RelayStatusBadge>{keys.length}</RelayStatusBadge>
          </div>

          {keys.length === 0 ? (
            <RelayEmptyState copy={t("emptyState")} />
          ) : (
            <ul className="relay-key-list">
              {keys.map((item) => {
                const isRevoking = revokingKeyID === item.key_id;
                const isConfirming = confirmingKeyID === item.key_id;
                const scopeLabel =
                  item.scope === "project" ? t("scopeProject") : t("scopeGlobal");

                return (
                  <li key={item.key_id} className="relay-key-row">
                    <div className="relay-key-row-top">
                      <div>
                        <strong className="relay-key-name">{item.name}</strong>
                        <p className="relay-key-meta">{item.token_prefix}</p>
                      </div>
                      <RelayStatusBadge variant={item.revoked ? "neutral" : "success"}>
                        {item.revoked ? t("revokedStatus") : t("activeStatus")}
                      </RelayStatusBadge>
                    </div>

                    <RelayMetaGrid>
                      <div>
                        <dt className="relay-meta-label">{t("scopeLabel")}</dt>
                        <dd className="relay-meta-value">{scopeLabel}</dd>
                      </div>
                      {item.project_id ? (
                        <div>
                          <dt className="relay-meta-label">{t("projectLabel")}</dt>
                          <dd className="relay-meta-value">{item.project_id}</dd>
                        </div>
                      ) : null}
                    </RelayMetaGrid>

                    {!item.revoked && (
                      <div className="relay-row-actions">
                        {isConfirming ? (
                          <>
                            <p className="relay-confirm-copy">{t("revokeConfirmCopy")}</p>
                            <div className="relay-form-actions">
                              <RelayButton
                                onClick={() => revokeKey(item.key_id)}
                                disabled={Boolean(revokingKeyID)}
                                variant="danger"
                                data-testid={`confirm-revoke-${item.key_id}`}
                              >
                                {isRevoking ? t("revokingButton") : t("confirmRevokeButton")}
                              </RelayButton>
                              <RelayButton
                                onClick={() => setConfirmingKeyID(null)}
                                disabled={Boolean(revokingKeyID)}
                                variant="secondary"
                              >
                                {t("cancelRevokeButton")}
                              </RelayButton>
                            </div>
                          </>
                        ) : (
                          <RelayButton
                            onClick={() => setConfirmingKeyID(item.key_id)}
                            variant="secondary"
                            data-testid={`revoke-api-key-${item.key_id}`}
                          >
                            {t("revokeButton")}
                          </RelayButton>
                        )}
                      </div>
                    )}
                  </li>
                );
              })}
            </ul>
          )}
        </RelayCard>
      </div>
    </section>
  );
}

function apiKeyErrorMap(t: ReturnType<typeof useTranslations>): Record<string, string> {
  return {
    UNAUTHENTICATED: t("UNAUTHENTICATED"),
    INVALID_INPUT: t("INVALID_INPUT"),
    API_KEY_NOT_FOUND_BY_ID: t("API_KEY_NOT_FOUND_BY_ID"),
  };
}
