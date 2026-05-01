import type { AnchorHTMLAttributes, ButtonHTMLAttributes } from "react";

import { cn } from "@/lib/utils";

type RelayActionVariant = "primary" | "secondary" | "danger" | "ghost";

type RelayButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: RelayActionVariant;
};

export function RelayButton({
  variant = "primary",
  className,
  ...props
}: RelayButtonProps) {
  return (
    <button
      className={cn("relay-action", className)}
      data-variant={variant}
      type={props.type ?? "button"}
      {...props}
    />
  );
}

type RelayLinkButtonProps = AnchorHTMLAttributes<HTMLAnchorElement> & {
  variant?: RelayActionVariant;
};

export function RelayLinkButton({
  variant = "secondary",
  className,
  ...props
}: RelayLinkButtonProps) {
  return <a className={cn("relay-action", className)} data-variant={variant} {...props} />;
}
