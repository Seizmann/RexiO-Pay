import type { ButtonHTMLAttributes, ReactNode } from "react";

type Variant = "primary" | "secondary" | "on-dark" | "danger" | "ghost";
type Size = "md" | "sm";

const variantClasses: Record<Variant, string> = {
  primary:
    "bg-primary text-on-primary active:bg-primary-press hover:bg-primary-deep",
  secondary: "bg-canvas text-primary border border-primary hover:bg-canvas-soft",
  "on-dark": "bg-brand-dark-900 text-on-primary",
  danger: "bg-ruby text-on-primary",
  ghost: "text-primary hover:bg-canvas-soft",
};

const sizeClasses: Record<Size, string> = {
  md: "text-button-md px-4 py-2",
  sm: "text-button-sm px-3 py-1.5",
};

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: Size;
  children: ReactNode;
}

/** Pill button per DESIGN.md button-primary-pill (8px 16px, radius pill). */
export function Button({
  variant = "primary",
  size = "md",
  className = "",
  disabled,
  type = "button",
  children,
  ...rest
}: ButtonProps) {
  return (
    <button
      type={type}
      disabled={disabled}
      className={`inline-flex items-center justify-center gap-2 rounded-pill font-normal transition-colors ${variantClasses[variant]} ${sizeClasses[size]} ${
        disabled ? "cursor-not-allowed opacity-50" : ""
      } ${className}`}
      {...rest}
    >
      {children}
    </button>
  );
}
