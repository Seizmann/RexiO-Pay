"use client";

import { Suspense, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { createClient } from "@/lib/supabase/client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useI18n } from "@/lib/i18n";

/**
 * Merchant login: magic link (default) or email + password.
 * REQUIREMENT §19 step 1 allows either; both are supported here.
 */
function LoginForm() {
  const { dict } = useI18n();
  const router = useRouter();
  const params = useSearchParams();
  const supabase = createClient();

  const [mode, setMode] = useState<"magic" | "password">("magic");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const nextPath = params.get("next") ?? "/app";

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      if (mode === "magic") {
        const { error } = await supabase.auth.signInWithOtp({
          email,
          options: {
            emailRedirectTo: `${location.origin}/auth/callback?next=${encodeURIComponent(nextPath)}`,
          },
        });
        if (error) throw error;
        setNotice(dict.auth.magicLinkSent);
      } else {
        const { error } = await supabase.auth.signInWithPassword({
          email,
          password,
        });
        if (error) throw error;
        router.push(nextPath);
        router.refresh();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : dict.common.errorGeneric);
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <form onSubmit={onSubmit} className="mt-8 flex flex-col gap-5">
        <Input
          id="email"
          type="email"
          label={dict.auth.email}
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          autoComplete="email"
        />
        {mode === "password" ? (
          <Input
            id="password"
            type="password"
            label={dict.auth.password}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            minLength={8}
            autoComplete="current-password"
          />
        ) : null}
        {error ? <p className="text-caption text-ruby">{error}</p> : null}
        {notice ? (
          <p className="text-caption text-ink-secondary">{notice}</p>
        ) : null}
        <Button type="submit" disabled={busy || !email}>
          {busy
            ? dict.common.loading
            : mode === "magic"
              ? dict.auth.magicLink
              : dict.auth.signIn}
        </Button>
      </form>
      <button
        type="button"
        onClick={() => setMode(mode === "magic" ? "password" : "magic")}
        className="mt-6 self-start text-body-md text-primary hover:underline"
      >
        {mode === "magic" ? dict.auth.orUsePassword : dict.auth.havePassword}
      </button>
    </>
  );
}

export default function LoginPage() {
  const { dict } = useI18n();

  return (
    <main className="mx-auto flex min-h-dvh w-full max-w-md flex-col justify-center px-6 py-16">
      <h1 className="text-display-lg text-ink">{dict.auth.signInTitle}</h1>
      <Suspense
        fallback={<div className="mt-8 text-body-md text-ink-mute">{dict.common.loading}</div>}
      >
        <LoginForm />
      </Suspense>
      <p className="mt-10 text-body-md text-ink-mute">
        {dict.auth.noAccount}{" "}
        <a href="/signup" className="text-primary hover:underline">
          {dict.auth.signUp}
        </a>
      </p>
    </main>
  );
}
