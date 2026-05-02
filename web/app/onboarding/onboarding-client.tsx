"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";

import { translateKnownError } from "@/lib/i18n";
import { completeOnboarding } from "@/lib/onboarding";
import { RelayButton, RelayCard, RelayFeedback, RelayPageHead } from "@/components/relay";

type Props = {
  userDisplayName?: string;
};

export function OnboardingClient({ userDisplayName }: Props) {
  const router = useRouter();
  const t = useTranslations("Onboarding.client");
  const commonErrors = useTranslations("Errors");
  const [status, setStatus] = useState<"idle" | "submitting" | "error">("idle");
  const [error, setError] = useState<string>("");

  async function startOnboarding() {
    setStatus("submitting");
    setError("");
    try {
      const result = await completeOnboarding();
      router.push(`/projects/${encodeURIComponent(result.default_project_id)}`);
      router.refresh();
    } catch (err) {
      setStatus("error");
      setError(
        translateKnownError({
          error: err,
          fallback: t("fallbackError"),
          knownErrors: {
            UNAUTHENTICATED: commonErrors("UNAUTHENTICATED"),
            INVALID_INPUT: commonErrors("INVALID_INPUT"),
          },
        }),
      );
    }
  }

  return (
    <RelayCard className="relay-onboarding-panel" aria-labelledby="onboarding-title">
      <div className="relay-onboarding-workspace-grid">
        <div className="relay-onboarding-workspace-card">
          <span className="relay-onboarding-glyph">●</span>
          <div>
            <h2 className="relay-onboarding-workspace-title">{t("personalTitle")}</h2>
            <p className="relay-onboarding-workspace-copy">{t("personalCopy")}</p>
          </div>
        </div>
        <div className="relay-onboarding-workspace-card">
          <span className="relay-onboarding-glyph" data-variant="magic">
            ◈
          </span>
          <div>
            <h2 className="relay-onboarding-workspace-title">{t("projectExplorerTitle")}</h2>
            <p className="relay-onboarding-workspace-copy">{t("projectExplorerCopy")}</p>
          </div>
        </div>
      </div>
      <div className="relay-onboarding-callout">{t("providerCallout")}</div>
      <div>
        <RelayPageHead
          eyebrow={`${t("signedInEyebrowPrefix")} · ${userDisplayName ?? t("fallbackUser")}`}
          title={t("title")}
          titleId="onboarding-title"
          copy={t("copy")}
        />
        <div className="relay-form-actions">
          <RelayButton
            onClick={startOnboarding}
            disabled={status === "submitting"}
            data-testid="complete-onboarding"
          >
            {status === "submitting" ? t("startingButton") : t("startButton")}
          </RelayButton>
          <span className="relay-onboarding-note">
            {t("providerSettingsLink")} · {t("afterWorkspace")}
          </span>
        </div>
        {status === "error" && (
          <RelayFeedback role="alert" variant="error">
            {error}
          </RelayFeedback>
        )}
      </div>
    </RelayCard>
  );
}
