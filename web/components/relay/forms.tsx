import type { HTMLAttributes, InputHTMLAttributes, ReactNode } from "react";

import { cn } from "@/lib/utils";

type RelayFieldProps = HTMLAttributes<HTMLDivElement> & {
  label: ReactNode;
  htmlFor: string;
  help?: ReactNode;
};

export function RelayField({
  label,
  htmlFor,
  help,
  className,
  children,
  ...props
}: RelayFieldProps) {
  return (
    <div className={cn("relay-field", className)} {...props}>
      <label className="relay-field-label" htmlFor={htmlFor}>
        {label}
      </label>
      {children}
      {help ? <p className="relay-field-help">{help}</p> : null}
    </div>
  );
}

export function RelayTextInput({ className, ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={cn("relay-text-input", className)} {...props} />;
}

type RelayFeedbackProps = HTMLAttributes<HTMLParagraphElement> & {
  variant?: "info" | "success" | "error";
};

export function RelayFeedback({
  variant = "info",
  className,
  ...props
}: RelayFeedbackProps) {
  return <p className={cn("relay-feedback", className)} data-variant={variant} {...props} />;
}

type RelayEmptyStateProps = HTMLAttributes<HTMLDivElement> & {
  glyph?: ReactNode;
  title?: ReactNode;
  copy?: ReactNode;
};

export function RelayEmptyState({
  glyph = "○",
  title,
  copy,
  className,
  children,
  ...props
}: RelayEmptyStateProps) {
  return (
    <div className={cn("relay-empty-state", className)} {...props}>
      <span className="relay-empty-glyph" aria-hidden="true">
        {glyph}
      </span>
      {title ? <h2 className="relay-empty-title">{title}</h2> : null}
      {copy ? <p className="relay-empty-copy">{copy}</p> : null}
      {children}
    </div>
  );
}
