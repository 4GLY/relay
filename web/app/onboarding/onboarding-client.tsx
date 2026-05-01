"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import type { Dictionary, Locale } from "@/lib/i18n";
import { translateErrorMessage } from "@/lib/i18n";
import { completeOnboarding } from "@/lib/onboarding";
import { RelayButton, RelayCard, RelayFeedback, RelayPageHead } from "@/components/relay";

type Props = {
  copy: Dictionary["onboarding"]["client"];
  locale: Locale;
  userDisplayName?: string;
};

export function OnboardingClient({ copy, locale, userDisplayName }: Props) {
  const router = useRouter();
  const [status, setStatus] = useState<"idle" | "submitting" | "error">("idle");
  const [error, setError] = useState<string>("");
  const localCopy = onboardingLocalCopy(locale);

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
        translateErrorMessage({
          error: err,
          fallback: copy.fallbackError,
          locale,
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
            <h2 className="relay-onboarding-workspace-title">Personal</h2>
            <p className="relay-onboarding-workspace-copy">{localCopy.personal}</p>
          </div>
        </div>
        <div className="relay-onboarding-workspace-card">
          <span className="relay-onboarding-glyph" data-variant="magic">
            ◈
          </span>
          <div>
            <h2 className="relay-onboarding-workspace-title">Project Explorer</h2>
            <p className="relay-onboarding-workspace-copy">{localCopy.projectExplorer}</p>
          </div>
        </div>
      </div>
      <div className="relay-onboarding-callout">
        {localCopy.providerCallout}
      </div>
      <div>
        <RelayPageHead
          eyebrow={`${copy.signedInEyebrowPrefix} · ${userDisplayName ?? copy.fallbackUser}`}
          title={copy.title}
          titleId="onboarding-title"
          copy={copy.copy}
        />
        <div className="relay-form-actions">
          <RelayButton
            onClick={startOnboarding}
            disabled={status === "submitting"}
            data-testid="complete-onboarding"
          >
            {status === "submitting" ? copy.startingButton : copy.startButton}
          </RelayButton>
          <span className="relay-onboarding-note">
            {copy.providerSettingsLink} · {localCopy.afterWorkspace}
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

function onboardingLocalCopy(locale: Locale) {
  if (locale === "ko") {
    return {
      personal: "항상 켜져 있는 개인 작업공간입니다.",
      projectExplorer: "provider 키 없이 먼저 작업공간을 만듭니다.",
      providerCallout:
        "Provider 키는 Settings에 남겨둡니다. 먼저 작업공간을 만들고, Claude 기반 기능이 필요할 때만 연결하세요.",
      afterWorkspace: "작업공간 생성 후",
    };
  }

  return {
    personal: "Always-on. Notes that are not yet a project.",
    projectExplorer: "Relay creates this workspace before any provider keys.",
    providerCallout:
      "Provider keys stay in Settings. Create the workspace first, then connect providers only when Claude-backed features need them.",
    afterWorkspace: "after workspace creation",
  };
}
