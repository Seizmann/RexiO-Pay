"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { api, ApiError } from "@/lib/api";
import type { Merchant, PresignResponse } from "@rexio-pay/shared-types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useI18n } from "@/lib/i18n";

/**
 * Branding (REQUIREMENT §14.9): logo/favicon upload via R2 presigned PUT
 * (POST /v1/assets/presign → PUT to R2 → PATCH /v1/branding with the
 * public_url), brand color, support contacts, live checkout preview.
 * The backend PATCH requires name + brand_color + valid support_phone on
 * every call, so the form always sends the full set.
 */
export default function BrandingPage() {
  const { dict } = useI18n();
  const m = dict.modules;
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [phone, setPhone] = useState("");
  const [color, setColor] = useState("#533afd");
  const [logoUrl, setLogoUrl] = useState("");
  const [faviconUrl, setFaviconUrl] = useState("");
  const [busy, setBusy] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const logoInput = useRef<HTMLInputElement>(null);
  const faviconInput = useRef<HTMLInputElement>(null);

  useEffect(() => {
    api
      .get<Merchant>("/branding")
      .then((m2) => {
        setName(m2.name);
        setEmail(m2.support_email);
        setPhone(m2.support_phone);
        setColor(m2.brand_color || "#533afd");
        setLogoUrl(m2.logo_url);
        setFaviconUrl(m2.favicon_url);
      })
      .catch(() => undefined);
  }, []);

  async function upload(file: File, kind: "logo" | "favicon") {
    setError(null);
    try {
      const presign = await api.post<PresignResponse>("/assets/presign", {
        filename: file.name,
        content_type: file.type,
        kind,
      });
      const put = await fetch(presign.upload_url, {
        method: "PUT",
        body: file,
        headers: { "Content-Type": file.type },
      });
      if (!put.ok) throw new Error("upload failed");
      if (kind === "logo") setLogoUrl(presign.public_url);
      else setFaviconUrl(presign.public_url);
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : `${m.brandingUploadFailed}`,
      );
    }
  }

  const save = useCallback(async () => {
    setBusy(true);
    setSaved(false);
    setError(null);
    try {
      await api.patch<Merchant>("/branding", {
        name,
        brand_color: color,
        support_email: email,
        support_phone: phone,
        logo_url: logoUrl,
        favicon_url: faviconUrl,
      });
      setSaved(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    } finally {
      setBusy(false);
    }
  }, [name, color, email, phone, logoUrl, faviconUrl, dict]);

  return (
    <div>
      <h1 className="text-display-lg text-ink">{m.brandingTitle}</h1>

      <div className="mt-8 grid gap-8 lg:grid-cols-2">
        <div className="flex flex-col gap-5">
          <Input id="br-name" label={dict.onboard.brandName} value={name} onChange={(e) => setName(e.target.value)} required />
          <Input id="br-email" type="email" label={dict.onboard.brandEmail} value={email} onChange={(e) => setEmail(e.target.value)} />
          <Input
            id="br-phone"
            type="tel"
            label={dict.onboard.brandPhone}
            hint={dict.onboard.brandPhoneHint}
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            placeholder="01XXXXXXXXX"
            required
          />
          <Input
            id="br-color"
            type="color"
            label={dict.onboard.brandColor}
            value={color}
            onChange={(e) => setColor(e.target.value)}
            className="h-12 w-24 p-1"
          />

          <div className="flex flex-col gap-3">
            <p className="text-caption text-ink-secondary">{m.brandingLogo}</p>
            <div className="flex items-center gap-3">
              {logoUrl ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={logoUrl} alt="logo" className="h-10 w-10 rounded-md object-contain" />
              ) : null}
              <input
                ref={logoInput}
                type="file"
                accept="image/png,image/jpeg,image/webp,image/svg+xml"
                className="hidden"
                onChange={(e) => {
                  const f = e.target.files?.[0];
                  if (f) void upload(f, "logo");
                }}
              />
              <Button variant="secondary" size="sm" onClick={() => logoInput.current?.click()}>
                {m.brandingUpload}
              </Button>
            </div>

            <p className="text-caption text-ink-secondary">{m.brandingFavicon}</p>
            <div className="flex items-center gap-3">
              {faviconUrl ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={faviconUrl} alt="favicon" className="h-8 w-8 rounded object-contain" />
              ) : null}
              <input
                ref={faviconInput}
                type="file"
                accept="image/png,image/x-icon,image/svg+xml"
                className="hidden"
                onChange={(e) => {
                  const f = e.target.files?.[0];
                  if (f) void upload(f, "favicon");
                }}
              />
              <Button variant="secondary" size="sm" onClick={() => faviconInput.current?.click()}>
                {m.brandingUpload}
              </Button>
            </div>
          </div>

          {error ? <p className="text-caption text-ruby">{error}</p> : null}
          {saved ? <p className="text-caption text-primary-deep">{dict.common.saved}</p> : null}
          <Button onClick={save} disabled={busy || !name || !phone} className="self-start">
            {busy ? dict.common.saving : dict.common.save}
          </Button>
        </div>

        {/* Live checkout preview */}
        <div>
          <p className="text-caption text-ink-mute">{m.brandingPreview}</p>
          <div className="mt-3 rounded-lg border border-hairline bg-canvas-soft p-6">
            <div className="mx-auto w-full max-w-xs rounded-lg border border-hairline bg-canvas p-5 shadow-level-1">
              <div className="flex items-center gap-2">
                {logoUrl ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={logoUrl} alt="" className="h-8 w-8 rounded object-contain" />
                ) : (
                  <div className="h-8 w-8 rounded-md" style={{ backgroundColor: color }} />
                )}
                <p className="text-heading-sm text-ink">{name || "Your business"}</p>
              </div>
              <p className="tnum mt-6 text-display-md text-ink">৳500</p>
              <button
                type="button"
                className="mt-6 w-full rounded-pill py-2 text-button-md text-on-primary"
                style={{ backgroundColor: color }}
              >
                {dict.checkout.paidButton}
              </button>
              <p className="mt-4 text-center text-micro text-ink-mute">
                {dict.checkout.poweredBy}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
