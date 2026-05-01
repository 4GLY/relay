import type { HTMLAttributes, ReactNode } from "react";

import { cn } from "@/lib/utils";

type RelayPageHeadProps = HTMLAttributes<HTMLElement> & {
  eyebrow?: ReactNode;
  title: ReactNode;
  titleId?: string;
  copy?: ReactNode;
  actions?: ReactNode;
};

export function RelayPageHead({
  eyebrow,
  title,
  titleId,
  copy,
  actions,
  className,
  ...props
}: RelayPageHeadProps) {
  return (
    <header className={cn("relay-page-head", className)} {...props}>
      <div className="relay-page-head-copy">
        {eyebrow ? <RelayPageKicker>{eyebrow}</RelayPageKicker> : null}
        <h1 id={titleId} className="relay-page-title">
          {title}
        </h1>
        {copy ? <p className="relay-page-copy">{copy}</p> : null}
      </div>
      {actions ? <RelayPageActions>{actions}</RelayPageActions> : null}
    </header>
  );
}

export function RelayPageKicker({ className, ...props }: HTMLAttributes<HTMLParagraphElement>) {
  return <p className={cn("relay-page-kicker", className)} {...props} />;
}

export function RelayPageActions({ className, ...props }: HTMLAttributes<HTMLElement>) {
  return <nav className={cn("relay-page-actions", className)} {...props} />;
}
