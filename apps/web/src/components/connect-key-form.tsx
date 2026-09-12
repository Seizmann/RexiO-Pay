"use client";

import { useState } from "react";
import { ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useI18n } from "@/lib/i18n";

/**
 * One-time merchant API key connection (Session 4 gap flag #1 workaround).
 * Posts the pasted key to /api/connect-key, which validates it against the
 * backend and stores it in an httpOnly cookie. The key never enters browser
 * state after this call.
 */
export function ConnectKeyForm({ onConnected }: { onConnected?: () => void }) {
  const { dict } = useI18n();
  const [apiKey, setApiKey] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [merchantName, setMerchantName] = useState<string | null>(null);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const res = await fetch("/api/connect-key", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ api_key: apiKey.trim() }),
      });
      const body = (await res.json()) as {
        merchant_name?: string;
        error?: { message: string };
      };
      if (!res.ok) {
        throw new ApiError(res.status, "unauthorized", body.error?.message ?? dict.connect.invalid);
      }
      setMerchantName(body.merchant_name ?? "");
      setApiKey("");
      onConnected?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : dict.connect.invalid);
    } finally {
      setBusy(false);
    }
  }

  if (merchantName !== null) {
    return (
      <p className="text-body-md text-ink-secondary">
        {dict.connect.connected} <strong>{merchantName}</strong>. {dict.common.saved}.
      </p>
    );
  }

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-5">
      <Input
        id="api_key"
        label={dict.connect.label}
        placeholder={dict.connect.placeholder}
        value={apiKey}
        onChange={(e) => setApiKey(e.target.value)}
        required
        autoComplete="off"
      />
      {error ? <p className="text-caption text-ruby">{error}</p> : null}
      <Button type="submit" disabled={busy || !apiKey}>
        {busy ? dict.connect.connecting : dict.connect.connect}
      </Button>
      <p className="text-micro text-ink-mute">{dict.connect.howTo}</p>
    </form>
  );
}
