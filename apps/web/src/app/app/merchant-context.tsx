"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { Merchant } from "@rexio-pay/shared-types";
import { createClient } from "@/lib/supabase/client";
import { Button } from "@/components/ui/button";
import { ConnectKeyForm } from "@/components/connect-key-form";
import { useI18n } from "@/lib/i18n";

/**
 * Loads the merchant profile via GET /v1/branding (the de-facto get-me,
 * it returns the full merchant row incl. plan and settings).
 * Renders the connect-key screen when the key cookie is missing/invalid.
 */
export function useMerchant() {
  const [merchant, setMerchant] = useState<Merchant | null>(null);
  const [state, setState] = useState<"loading" | "connected" | "disconnected">(
    "loading",
  );

  const load = () => {
    setState("loading");
    api
      .get<Merchant>("/branding")
      .then((m) => {
        setMerchant(m);
        setState("connected");
      })
      .catch(() => setState("disconnected"));
  };

  useEffect(load, []);

  return { merchant, state, reload: load };
}

/** Gate shown across the dashboard until a valid key cookie exists. */
export function ConnectGate() {
  const { dict } = useI18n();
  const supabase = createClient();
  return (
    <div className="mx-auto w-full max-w-md rounded-lg border border-hairline bg-canvas p-8 shadow-level-1">
      <h1 className="text-heading-lg text-ink">{dict.connect.title}</h1>
      <p className="mt-3 text-body-md text-ink-secondary">{dict.connect.body}</p>
      <div className="mt-6">
        <ConnectKeyForm onConnected={() => window.location.reload()} />
      </div>
      <Button
        variant="ghost"
        size="sm"
        className="mt-6"
        onClick={() => void supabase.auth.signOut()}
      >
        {dict.common.signOut}
      </Button>
    </div>
  );
}
