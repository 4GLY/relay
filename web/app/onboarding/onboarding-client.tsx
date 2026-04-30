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
      <div style={stepRailStyle} aria-hidden="true">
        <span style={activeDotStyle} />
        <span style={lineStyle} />
        <span style={dotStyle} />
        <span style={lineStyle} />
        <span style={dotStyle} />
      </div>
      <div>
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
          <a href="/style-memory" style={secondaryLinkStyle}>
            {copy.styleMemoryLink}
          </a>
          <a href="/settings/providers" style={secondaryLinkStyle}>
            {copy.providerSettingsLink}
          </a>
          <a href="/settings/api-keys" style={secondaryLinkStyle}>
            {copy.apiKeysLink}
          </a>
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

const panelStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateColumns: "32px minmax(0, 1fr)",
  gap: "24px",
  maxWidth: "760px",
  padding: "32px",
  border: "1px solid var(--border)",
  borderRadius: "8px",
  background: "var(--canvas-raised)",
};

const stepRailStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateRows: "18px 1fr 18px 1fr 18px",
  justifyItems: "center",
  minHeight: "168px",
  paddingTop: "6px",
};

const dotBaseStyle: React.CSSProperties = {
  width: "14px",
  height: "14px",
  borderRadius: "50%",
  border: "1px solid var(--border-strong)",
};

const activeDotStyle: React.CSSProperties = {
  ...dotBaseStyle,
  background: "var(--success)",
  boxShadow: "0 0 0 5px color-mix(in oklab, var(--success) 18%, transparent)",
};

const dotStyle: React.CSSProperties = {
  ...dotBaseStyle,
  background: "var(--canvas)",
};

const lineStyle: React.CSSProperties = {
  width: "1px",
  height: "100%",
  background: "var(--border)",
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

const secondaryLinkStyle: React.CSSProperties = {
  color: "var(--ink-muted)",
  fontFamily: "var(--font-sans)",
  fontSize: "14px",
  fontWeight: 700,
  textDecoration: "none",
};

const errorStyle: React.CSSProperties = {
  margin: "18px 0 0",
  padding: "12px 14px",
  border: "1px solid color-mix(in oklab, var(--danger) 35%, var(--border))",
  borderRadius: "8px",
  color: "var(--danger)",
  background: "color-mix(in oklab, var(--danger) 8%, var(--canvas-raised))",
};
