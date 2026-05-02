"use client";

import type { ReactNode } from "react";
import { useTranslations } from "next-intl";

import { RelayLanguageSwitch } from "@/components/relay/language-switch";

export type RelayStep = "Face" | "Dissect" | "Refine" | "Transform";

type RailItem = {
  href: string;
  label: string;
  glyph?: "active" | "snapshot" | "pending" | "empty";
  active?: boolean;
  ducklings?: number;
  swans?: number;
};

const steps: RelayStep[] = ["Face", "Dissect", "Refine", "Transform"];

const glyphs: Record<NonNullable<RailItem["glyph"]>, string> = {
  active: "●",
  snapshot: "◈",
  pending: "△",
  empty: "○",
};

export function RelayTopRail({
  activeStep = "Refine",
  userLabel,
  projectHref = "/",
}: {
  activeStep?: RelayStep;
  userLabel?: string;
  projectHref?: string;
}) {
  const t = useTranslations("Shell");

  return (
    <header className="relay-top-rail">
      <a className="relay-wordmark" href={projectHref}>
        Relay<span className="relay-wordmark-dot">.</span>
      </a>
      <TransformRibbon activeStep={activeStep} />
      <nav className="relay-top-actions" aria-label={t("globalNavigation")}>
        <RelayLanguageSwitch />
        <a className="relay-top-link" href="/settings/providers">
          {t("settings")}
        </a>
        <span className="relay-user-chip">{userLabel ?? t("signedInFallback")}</span>
      </nav>
    </header>
  );
}

export function RelayAppShell({
  activeStep = "Refine",
  userLabel,
  projectHref = "/",
  railItems,
  inspector,
  children,
}: {
  activeStep?: RelayStep;
  userLabel?: string;
  projectHref?: string;
  railItems: RailItem[];
  inspector?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="relay-app-shell">
      <RelayTopRail activeStep={activeStep} userLabel={userLabel} projectHref={projectHref} />
      <ProjectRail items={railItems} />
      <main className="relay-shell-main">{children}</main>
      {inspector ? <aside className="relay-inspector">{inspector}</aside> : null}
    </div>
  );
}

function TransformRibbon({ activeStep }: { activeStep: RelayStep }) {
  return (
    <div className="relay-transform-ribbon" aria-label="Relay transformation steps">
      {steps.map((step, index) => (
        <span key={step} className="relay-contents">
          <span className="relay-transform-step" data-active={step === activeStep}>
            {step}
          </span>
          {index < steps.length - 1 ? <span className="relay-transform-arrow">→</span> : null}
        </span>
      ))}
    </div>
  );
}

function ProjectRail({ items }: { items: RailItem[] }) {
  return (
    <aside className="relay-project-rail" aria-label="Project Explorer">
      <div className="relay-rail-section">Project Explorer</div>
      {items.map((item) => {
        const kind = item.glyph ?? "empty";
        return (
          <a
            key={`${item.href}:${item.label}`}
            className="relay-rail-item"
            data-active={item.active || undefined}
            href={item.href}
          >
            <span className="relay-rail-glyph" data-kind={kind}>
              {glyphs[kind]}
            </span>
            <span className="relay-rail-name">{item.label}</span>
            <span className="relay-rail-badges">
              {item.ducklings ? (
                <span className="relay-badge-duckling" title="Ducklings">
                  {item.ducklings}
                </span>
              ) : null}
              {item.swans ? (
                <span className="relay-badge-swan" title="Swans">
                  {item.swans}
                </span>
              ) : null}
            </span>
          </a>
        );
      })}
    </aside>
  );
}
