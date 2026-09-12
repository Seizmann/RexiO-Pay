import { createBrowserClient } from "@supabase/ssr";

/** Browser Supabase client (singleton). Reads via RLS with the user's JWT. */
export function createClient() {
  return createBrowserClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!,
  );
}
