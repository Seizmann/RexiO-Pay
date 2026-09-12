import type { InputHTMLAttributes, ReactNode } from "react";

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  hint?: string;
  children?: ReactNode;
}

/** Form field per DESIGN.md text-input (6px radius, hairline-input border). */
export function Input({
  label,
  error,
  hint,
  id,
  className = "",
  children,
  ...rest
}: InputProps) {
  return (
    <div className="flex flex-col gap-1.5">
      {label ? (
        <label htmlFor={id} className="text-caption text-ink-secondary">
          {label}
        </label>
      ) : null}
      <input
        id={id}
        className={`h-10 w-full rounded-sm border bg-canvas px-3 py-2 text-body-md text-ink outline-none transition-colors focus:border-primary ${
          error ? "border-ruby" : "border-hairline-input"
        } ${className}`}
        {...rest}
      />
      {error ? (
        <p className="text-caption text-ruby">{error}</p>
      ) : hint ? (
        <p className="text-caption text-ink-mute">{hint}</p>
      ) : null}
      {children}
    </div>
  );
}
