"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import type { Dictionary, Locale } from "@/lib/i18n";
import { translateErrorMessage } from "@/lib/i18n";
import { completeOnboarding } from "@/lib/onboarding";

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
    <section style={panelStyle} aria-labelledby="onboarding-title">
      <div style={workspaceGridStyle}>
        <div style={workspaceCardStyle}>
          <span style={workspaceGlyphStyle}>●</span>
          <div>
            <h2 style={workspaceTitleStyle}>Personal</h2>
            <p style={workspaceCopyStyle}>{localCopy.personal}</p>
          </div>
        </div>
        <div style={workspaceCardStyle}>
          <span style={{ ...workspaceGlyphStyle, color: "var(--magic-primary-strong)" }}>◈</span>
          <div>
            <h2 style={workspaceTitleStyle}>Project Explorer</h2>
            <p style={workspaceCopyStyle}>{localCopy.projectExplorer}</p>
          </div>
        </div>
      </div>
      <div style={emptyCalloutStyle}>
        {localCopy.providerCallout}
      </div>
      <div style={contentStyle}>
        <p style={eyebrowStyle}>
          {copy.signedInEyebrowPrefix} · {userDisplayName ?? copy.fallbackUser}
        </p>
        <h1 id="onboarding-title" style={titleStyle}>
          {copy.title}
        </h1>
        <p style={copyStyle}>{copy.copy}</p>
        <div style={actionsStyle}>
          <button
            type="button"
            onClick={startOnboarding}
            disabled={status === "submitting"}
            style={{
              ...primaryButtonStyle,
              opacity: status === "submitting" ? 0.68 : 1,
              cursor: status === "submitting" ? "wait" : "pointer",
            }}
            data-testid="complete-onboarding"
          >
            {status === "submitting" ? copy.startingButton : copy.startButton}
          </button>
          <span style={secondaryNoteStyle}>
            {copy.providerSettingsLink} · {localCopy.afterWorkspace}
          </span>
        </div>
        {status === "error" && (
          <p role="alert" style={errorStyle}>
            {error}
          </p>
        )}
      </div>
    </section>
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

const panelStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: "24px",
  padding: "28px",
  border: "1px solid var(--border)",
  borderRadius: "12px",
  background: "var(--canvas-raised)",
};

const workspaceGridStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))",
  gap: "14px",
};

const workspaceCardStyle: React.CSSProperties = {
  display: "flex",
  gap: "14px",
  alignItems: "flex-start",
  padding: "18px",
  border: "1px solid var(--border)",
  borderRadius: "12px",
  background: "var(--canvas)",
};

const workspaceGlyphStyle: React.CSSProperties = {
  color: "var(--success)",
  fontFamily: "var(--font-mono)",
  fontSize: "22px",
  lineHeight: 1,
};

const workspaceTitleStyle: React.CSSProperties = {
  margin: 0,
  color: "var(--ink)",
  fontFamily: "var(--font-sans)",
  fontSize: "16px",
  fontWeight: 800,
};

const workspaceCopyStyle: React.CSSProperties = {
  margin: "4px 0 0",
  color: "var(--ink-muted)",
  fontSize: "13px",
  lineHeight: 1.5,
};

const emptyCalloutStyle: React.CSSProperties = {
  padding: "18px",
  border: "1px dashed var(--border-strong)",
  borderRadius: "12px",
  background: "color-mix(in srgb, var(--magic-primary) 8%, var(--canvas))",
  color: "var(--ink-muted)",
  fontFamily: "var(--font-display)",
  fontSize: "17px",
  fontStyle: "italic",
  lineHeight: 1.45,
  textAlign: "center",
};

const contentStyle: React.CSSProperties = {
  paddingTop: "2px",
};

const eyebrowStyle: React.CSSProperties = {
  margin: "0 0 14px",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
  color: "var(--muted)",
};

const titleStyle: React.CSSProperties = {
  margin: "0 0 18px",
  fontFamily: "var(--font-display)",
  fontWeight: 500,
  fontSize: "clamp(36px, 5vw, 58px)",
  lineHeight: 1.02,
  letterSpacing: "-0.02em",
  fontVariationSettings: '"opsz" 144, "SOFT" 50',
};

const copyStyle: React.CSSProperties = {
  maxWidth: "560px",
  margin: "0 0 28px",
  color: "var(--ink-muted)",
  fontSize: "16px",
  lineHeight: 1.65,
};

const actionsStyle: React.CSSProperties = {
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: "14px",
};

const primaryButtonStyle: React.CSSProperties = {
  minHeight: "44px",
  border: 0,
  borderRadius: "8px",
  padding: "0 20px",
  background: "var(--ink)",
  color: "var(--canvas)",
  fontFamily: "var(--font-sans)",
  fontSize: "14px",
  fontWeight: 800,
};

const secondaryNoteStyle: React.CSSProperties = {
  color: "var(--ink-muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  letterSpacing: "0.08em",
  textTransform: "uppercase",
};

const errorStyle: React.CSSProperties = {
  margin: "18px 0 0",
  padding: "12px 14px",
  border: "1px solid color-mix(in oklab, var(--danger) 35%, var(--border))",
  borderRadius: "8px",
  color: "var(--danger)",
  background: "color-mix(in oklab, var(--danger) 8%, var(--canvas-raised))",
};
