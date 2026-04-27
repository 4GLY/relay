type PlaceholderCard = {
  href?: string;
  eyebrow: string;
  title: string;
  blurb: string;
  ships: string;
};

const cards: PlaceholderCard[] = [
  {
    href: "/style-memory",
    eyebrow: "S6 · authenticated",
    title: "Style Memory",
    blurb:
      "Approve a duckling, watch it become a swan. The 900 ms signature transform lives here.",
    ships: "Ships in S6",
  },
  {
    eyebrow: "S7 · public",
    title: "Sharable Snapshot",
    blurb:
      "A handoff packet from one AI agent to another, carrying your judgment style across the gap.",
    ships: "Snapshot link appears after publish",
  },
  {
    href: "/onboarding",
    eyebrow: "S8 · 60 seconds",
    title: "1-click Onboarding",
    blurb:
      "Create your workspace and receive your first packet inside a minute. Provider keys can wait.",
    ships: "Ships in S8",
  },
];

export default function HomePage() {
  return (
    <main
      style={{
        maxWidth: "960px",
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
          marginBottom: "32px",
        }}
      >
        4gly Labs · Relay V2 · Web scaffold
      </p>

      <h1
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 500,
          fontSize: "clamp(56px, 9vw, 112px)",
          lineHeight: 0.95,
          letterSpacing: "-0.03em",
          color: "var(--ink)",
          marginBottom: "24px",
          fontVariationSettings: '"opsz" 144, "SOFT" 50',
        }}
      >
        Relay
      </h1>

      <p
        style={{
          fontFamily: "var(--font-display)",
          fontStyle: "italic",
          fontWeight: 400,
          fontSize: "clamp(20px, 2.8vw, 28px)",
          lineHeight: 1.35,
          color: "var(--ink-muted)",
          maxWidth: "640px",
          marginBottom: "64px",
          fontVariationSettings: '"opsz" 48',
        }}
      >
        A quiet engine that turns chaos into swans.
      </p>

      <ul
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fit, minmax(240px, 1fr))",
          gap: "16px",
          listStyle: "none",
          padding: 0,
          margin: 0,
        }}
      >
        {cards.map((card) => (
          <li key={card.title} style={{ display: "flex" }}>
            <a
              href={card.href}
              aria-disabled={card.href ? undefined : true}
              style={{
                display: "flex",
                flexDirection: "column",
                gap: "12px",
                width: "100%",
                padding: "24px 22px",
                background: "var(--canvas-raised)",
                border: "1px solid var(--border)",
                borderRadius: "12px",
                color: "var(--ink)",
                textDecoration: "none",
                pointerEvents: card.href ? undefined : "none",
                transition:
                  "border-color 200ms cubic-bezier(0.2,0.8,0.2,1), box-shadow 200ms cubic-bezier(0.2,0.8,0.2,1)",
              }}
            >
              <span
                style={{
                  fontFamily: "var(--font-mono)",
                  fontSize: "10.5px",
                  letterSpacing: "0.14em",
                  textTransform: "uppercase",
                  color: "var(--muted)",
                }}
              >
                {card.eyebrow}
              </span>
              <h2
                style={{
                  fontFamily: "var(--font-display)",
                  fontWeight: 500,
                  fontSize: "24px",
                  letterSpacing: "-0.018em",
                  margin: 0,
                  fontVariationSettings: '"opsz" 96',
                }}
              >
                {card.title}
              </h2>
              <p
                style={{
                  fontFamily: "var(--font-sans)",
                  fontSize: "14px",
                  lineHeight: 1.55,
                  color: "var(--ink-muted)",
                  margin: 0,
                }}
              >
                {card.blurb}
              </p>
              <span
                style={{
                  fontFamily: "var(--font-mono)",
                  fontSize: "10.5px",
                  letterSpacing: "0.10em",
                  textTransform: "uppercase",
                  color: "var(--magic-primary-strong)",
                  marginTop: "auto",
                }}
              >
                {card.ships} →
              </span>
            </a>
          </li>
        ))}
      </ul>
    </main>
  );
}
