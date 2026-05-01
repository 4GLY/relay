import type { HTMLAttributes, ReactNode } from "react";

import { cn } from "@/lib/utils";

type RelayTabItem = {
  label: ReactNode;
  count?: ReactNode;
  active?: boolean;
  href?: string;
};

type RelayTabsProps = HTMLAttributes<HTMLElement> & {
  items: RelayTabItem[];
};

export function RelayTabs({ items, className, ...props }: RelayTabsProps) {
  return (
    <nav className={cn("relay-tabs", className)} {...props}>
      {items.map((item, index) => {
        const content = (
          <>
            <span>{item.label}</span>
            {item.count !== undefined ? <span className="relay-tab-count">{item.count}</span> : null}
          </>
        );

        return item.href ? (
          <a
            key={`${String(item.label)}:${index}`}
            className="relay-tab"
            data-active={item.active || undefined}
            aria-current={item.active ? "page" : undefined}
            href={item.href}
          >
            {content}
          </a>
        ) : (
          <span
            key={`${String(item.label)}:${index}`}
            className="relay-tab"
            data-active={item.active || undefined}
            aria-current={item.active ? "page" : undefined}
          >
            {content}
          </span>
        );
      })}
    </nav>
  );
}

type RelayMetricTileProps = HTMLAttributes<HTMLDivElement> & {
  label: ReactNode;
  value: ReactNode;
};

export function RelayMetricTile({ label, value, className, ...props }: RelayMetricTileProps) {
  return (
    <div className={cn("relay-metric", className)} {...props}>
      <span className="relay-metric-value">{value}</span>
      <span className="relay-metric-label">{label}</span>
    </div>
  );
}

type RelayStatusVariant = "neutral" | "success" | "danger" | "pending" | "magic";

type RelayStatusBadgeProps = HTMLAttributes<HTMLSpanElement> & {
  variant?: RelayStatusVariant;
};

export function RelayStatusBadge({
  variant = "neutral",
  className,
  ...props
}: RelayStatusBadgeProps) {
  return <span className={cn("relay-status-badge", className)} data-variant={variant} {...props} />;
}

export function RelaySourceChip({ className, ...props }: HTMLAttributes<HTMLSpanElement>) {
  return <span className={cn("relay-source-chip", className)} {...props} />;
}

export function RelayListRow({ className, ...props }: HTMLAttributes<HTMLElement>) {
  return <article className={cn("relay-list-row", className)} {...props} />;
}

export function RelayMetaGrid({ className, ...props }: HTMLAttributes<HTMLDListElement>) {
  return <dl className={cn("relay-meta-grid", className)} {...props} />;
}
