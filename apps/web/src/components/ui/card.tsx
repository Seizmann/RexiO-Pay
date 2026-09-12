import type { HTMLAttributes, ReactNode } from "react";

type Variant = "light" | "featured" | "cream";

const variantClasses: Record<Variant, string> = {
  light: "bg-canvas border border-hairline",
  featured: "bg-brand-dark-900 text-on-primary",
  cream: "bg-canvas-cream",
};

export interface CardProps extends HTMLAttributes<HTMLDivElement> {
  variant?: Variant;
  children: ReactNode;
}

/** Card per DESIGN.md card-feature-light / card-pricing(-featured) / card-cream-band. */
export function Card({
  variant = "light",
  className = "",
  children,
  ...rest
}: CardProps) {
  return (
    <div
      className={`rounded-lg p-8 ${variantClasses[variant]} ${className}`}
      {...rest}
    >
      {children}
    </div>
  );
}
