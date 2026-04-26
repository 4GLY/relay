export default function OnboardingPage() {
  return (
    <main
      style={{
        maxWidth: "720px",
        margin: "0 auto",
        padding: "96px 32px 120px",
      }}
    >
      <p
        style={{
          fontFamily: "var(--font-mono)",
          fontSize: "11px",
          letterSpacing: "0.18em",
          textTransform: "uppercase",
          color: "var(--muted)",
          marginBottom: "16px",
        }}
      >
        Slice 8 · 60 seconds
      </p>
      <h1
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 500,
          fontSize: "clamp(40px, 6.5vw, 64px)",
          lineHeight: 1.05,
          letterSpacing: "-0.025em",
          color: "var(--ink)",
          marginBottom: "20px",
          fontVariationSettings: '"opsz" 144, "SOFT" 50',
        }}
      >
        Onboarding
      </h1>
      <p
        style={{
          fontFamily: "var(--font-sans)",
          fontSize: "16px",
          lineHeight: 1.6,
          color: "var(--ink-muted)",
          maxWidth: "560px",
        }}
      >
        Coming in S8 — keyless first-run onboarding. Claude provider keys move to a later settings flow.
      </p>
    </main>
  );
}
