"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { createClient } from "@/lib/supabase/client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useI18n } from "@/lib/i18n";

/**
 * Merchant signup. Creates the Supabase auth user only.
 * Gap flag #1: the backend does not create a merchant record on signup yet;
 * after confirming, the merchant connects their API key at /app/connect.
 */
export default function SignUpPage() {
  const { dict } = useI18n();
  const router = useRouter();
  const supabase = createClient();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const { error } = await supabase.auth.signUp({
        email,
        password,
        options: {
          emailRedirectTo: `${location.origin}/auth/callback?next=/app`,
        },
      });
      if (error) throw error;
      // Session exists immediately unless the project requires email confirmation.
      router.push("/app");
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : dict.common.errorGeneric);
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="mx-auto flex min-h-dvh w-full max-w-md flex-col justify-center px-6 py-16">
      <h1 className="text-display-lg text-ink">{dict.auth.signUpTitle}</h1>
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
        <Input
          id="password"
          type="password"
          label={dict.auth.password}
          hint={dict.auth.passwordRule}
          error={password.length > 0 && password.length < 8 ? dict.auth.passwordShort : undefined}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          minLength={8}
          autoComplete="new-password"
        />
        {error ? <p className="text-caption text-ruby">{error}</p> : null}
        <Button type="submit" disabled={busy || !email || password.length < 8}>
          {busy ? dict.common.loading : dict.auth.signUp}
        </Button>
      </form>
      <p className="mt-10 text-body-md text-ink-mute">
        {dict.auth.haveAccount}{" "}
        <a href="/login" className="text-primary hover:underline">
          {dict.auth.signIn}
        </a>
      </p>
    </main>
  );
}
