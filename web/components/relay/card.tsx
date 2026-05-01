import type { HTMLAttributes } from "react";

import { cn } from "@/lib/utils";

type RelayCardProps = HTMLAttributes<HTMLElement> & {
  selected?: boolean;
  variant?: "default" | "elevated" | "dark" | "soft";
};

export function RelayCard({
  selected,
  variant = "default",
  className,
  ...props
}: RelayCardProps) {
  return (
    <section
      className={cn("relay-card", className)}
      data-selected={selected || undefined}
      data-variant={variant}
      {...props}
    />
  );
}

export function RelayCardHeader({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("relay-card-header", className)} {...props} />;
}

export function RelayCardKicker({ className, ...props }: HTMLAttributes<HTMLParagraphElement>) {
  return <p className={cn("relay-card-kicker", className)} {...props} />;
}

export function RelayCardTitle({ className, ...props }: HTMLAttributes<HTMLHeadingElement>) {
  return <h2 className={cn("relay-card-title", className)} {...props} />;
}
