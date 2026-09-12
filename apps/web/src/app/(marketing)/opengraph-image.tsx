import { ImageResponse } from "next/og";

/**
 * Static fallback OG image built from DESIGN.md tokens: brand color wash,
 * indigo accent, RexiO Pay wordmark. (The /docs folder also ships
 * og-image.webp; ImageResponse keeps the metadata route self-contained.)
 */
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";
export const alt = "RexiO Pay: bKash and Nagad payments, verified automatically";

export default function OpengraphImage() {
  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          alignItems: "flex-start",
          justifyContent: "flex-end",
          padding: 72,
          background: "linear-gradient(120deg, #f5e9d4 0%, #f96bee55 30%, #533afd 100%)",
          color: "#0d253d",
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 16,
            fontSize: 28,
            color: "#4434d4",
            marginBottom: 12,
          }}
        >
          RexiO Pay
        </div>
        <div
          style={{
            display: "flex",
            fontSize: 64,
            fontWeight: 300,
            letterSpacing: -2,
            lineHeight: 1.05,
            maxWidth: 900,
          }}
        >
          bKash and Nagad payments, verified automatically
        </div>
        <div style={{ display: "flex", fontSize: 26, marginTop: 24, color: "#273951" }}>
          A SpritexAI product
        </div>
      </div>
    ),
    size,
  );
}
