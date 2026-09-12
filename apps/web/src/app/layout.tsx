import type { Metadata } from "next";
import { Inter, Noto_Sans_Bengali } from "next/font/google";
import { I18nProvider } from "@/lib/i18n";
import "./globals.css";

const inter = Inter({
  subsets: ["latin"],
  weight: ["300", "400"],
  variable: "--font-inter",
  display: "swap",
});

const bengali = Noto_Sans_Bengali({
  subsets: ["bengali", "latin"],
  weight: ["300", "400"],
  variable: "--font-bengali",
  display: "swap",
});

export const metadata: Metadata = {
  metadataBase: new URL(
    process.env.NEXT_PUBLIC_SITE_URL ?? "https://pay.rexio.pro",
  ),
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
    <html lang="en" className={`${inter.variable} ${bengali.variable}`}>
      <body>
        <I18nProvider initialLang="en">{children}</I18nProvider>
      </body>
    </html>
  );
}
