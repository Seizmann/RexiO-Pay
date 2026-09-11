import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: {
    default: "RexiO Pay",
    template: "%s · RexiO Pay",
  },
  description:
    "Accept bKash and Nagad payments automatically. RexiO Pay verifies each payment from the SMS and notifies your site instantly.",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
