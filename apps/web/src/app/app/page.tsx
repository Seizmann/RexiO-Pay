import { ConnectKeyForm } from "@/components/connect-key-form";

/**
 * Temporary /app landing until the dashboard shell lands (Step 5).
 * Confirms the Supabase session works and offers the connect-key step.
 */
export default function AppIndexPage() {
  return (
    <main className="mx-auto flex min-h-dvh w-full max-w-xl flex-col justify-center px-6 py-16">
      <h1 className="text-display-lg text-ink">Dashboard</h1>
      <p className="mt-4 text-body-md text-ink-mute">
        Sign-in works. Connect your merchant API key to finish setup.
      </p>
      <div className="mt-8">
        <ConnectKeyForm />
      </div>
    </main>
  );
}
