"use client";

import { useCallback, useEffect, useState } from "react";
import { api, ApiError } from "@/lib/api";
import type { Merchant, MerchantSettings, TeamMember } from "@rexio-pay/shared-types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useI18n } from "@/lib/i18n";
import { useI18nSetter } from "@/lib/use-lang-setting";

/**
 * Settings (REQUIREMENT §14.11): plan/billing display, team members (Pro+),
 * claim window / TTL / tolerance / late-match / balance verification
 * (merchants.settings jsonb via PATCH /v1/settings), language toggle.
 */
export default function SettingsPage() {
  const { dict, lang } = useI18n();
  const s = dict.settings;
  const [merchant, setMerchant] = useState<Merchant | null>(null);
  const [settings, setSettings] = useState<MerchantSettings | null>(null);
  const [team, setTeam] = useState<TeamMember[]>([]);
  const [inviteEmail, setInviteEmail] = useState("");
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { setLang } = useI18nSetter();

  const load = useCallback(async () => {
    try {
      const [mRes, sRes, tRes] = await Promise.all([
        api.get<Merchant>("/branding"),
        api.get<{ settings: MerchantSettings }>("/settings"),
        api.get<{ data: TeamMember[] }>("/team"),
      ]);
      setMerchant(mRes);
      setSettings(sRes.settings);
      setTeam(tRes.data);
    } catch {
      // shell handles disconnect
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function saveSettings(patch: Partial<MerchantSettings>) {
    setError(null);
    setSaved(false);
    try {
      const res = await api.patch<{ settings: MerchantSettings }>("/settings", patch);
      setSettings(res.settings);
      setSaved(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    }
  }

  async function invite(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      await api.post("/team", { email: inviteEmail, role: "staff" });
      setInviteEmail("");
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    }
  }

  async function removeMember(userId: string) {
    setError(null);
    try {
      await api.delete(`/team/${userId}`);
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    }
  }

  return (
    <div className="max-w-3xl">
      <h1 className="text-display-lg text-ink">{s.title}</h1>
      {error ? <p className="mt-3 text-caption text-ruby">{error}</p> : null}
      {saved ? <p className="mt-3 text-caption text-primary-deep">{s.savedOk}</p> : null}

      {/* Plan & billing */}
      <section className="mt-8 rounded-lg border border-hairline bg-canvas p-6">
        <h2 className="text-heading-md text-ink">{s.currentPlan}</h2>
        {merchant ? (
          <>
            <p className="mt-2 text-body-md capitalize text-ink">
              {merchant.plan_id} · {merchant.plan_status}
            </p>
            <p className="tnum mt-1 text-caption text-ink-mute">
              {s.sessionsUsed}: {merchant.session_count_current_period}
              {merchant.plan_renews_at
                ? ` · ${s.renews}: ${new Date(merchant.plan_renews_at).toLocaleDateString()}`
                : ""}
            </p>
          </>
        ) : null}
        <p className="mt-3 rounded-md bg-canvas-soft px-4 py-3 text-caption text-ink-secondary">
          {s.billingNote}
        </p>
      </section>

      {/* Matching settings */}
      {settings ? (
        <section className="mt-6 rounded-lg border border-hairline bg-canvas p-6">
          <h2 className="text-heading-md text-ink">{dict.marketing.navProduct}</h2>
          <div className="mt-4 grid gap-5 sm:grid-cols-2">
            <NumberField
              id="set-claim"
              label={s.claimWindow}
              value={settings.claim_window_minutes}
              onCommit={(v) => saveSettings({ claim_window_minutes: v })}
            />
            <div>
              <NumberField
                id="set-ttl"
                label={s.sessionTtl}
                value={settings.session_ttl_minutes}
                onCommit={(v) => saveSettings({ session_ttl_minutes: v })}
              />
              <p className="mt-1 text-micro text-ink-mute">{s.sessionTtlNote}</p>
            </div>
            <NumberField
              id="set-tol"
              label={s.amountTolerance}
              value={settings.amount_tolerance}
              onCommit={(v) => saveSettings({ amount_tolerance: v })}
            />
            <ToggleField
              label={s.autoLate}
              checked={settings.auto_accept_late_match}
              onChange={(v) => saveSettings({ auto_accept_late_match: v })}
            />
            <div>
              <ToggleField
                label={s.balanceVerification}
                checked={settings.balance_verification}
                onChange={(v) => saveSettings({ balance_verification: v })}
              />
              <p className="mt-1 text-micro text-ink-mute">{s.balanceVerificationHint}</p>
            </div>
          </div>
        </section>
      ) : null}

      {/* Team members */}
      <section className="mt-6 rounded-lg border border-hairline bg-canvas p-6">
        <h2 className="text-heading-md text-ink">{s.teamTitle}</h2>
        <form onSubmit={invite} className="mt-4 flex items-end gap-3">
          <Input
            id="team-email"
            type="email"
            label={s.teamEmail}
            hint={s.teamInviteHint}
            value={inviteEmail}
            onChange={(e) => setInviteEmail(e.target.value)}
            required
          />
          <Button type="submit" disabled={!inviteEmail}>
            {s.teamInvite}
          </Button>
        </form>
        {team.length === 0 ? (
          <p className="mt-4 text-body-md text-ink-mute">{dict.modules.settingsTeamEmpty}</p>
        ) : (
          <ul className="mt-4 flex flex-col gap-2">
            {team.map((t) => (
              <li
                key={t.user_id}
                className="flex items-center justify-between rounded-md bg-canvas-soft px-4 py-2 text-body-md"
              >
                <span className="text-ink">{t.role}</span>
                <Button variant="ghost" size="sm" onClick={() => removeMember(t.user_id)}>
                  {s.teamRemove}
                </Button>
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* Language */}
      <section className="mt-6 rounded-lg border border-hairline bg-canvas p-6">
        <h2 className="text-heading-md text-ink">{s.languageTitle}</h2>
        <div className="mt-3 flex gap-3">
          <Button
            variant={lang === "en" ? "primary" : "secondary"}
            size="sm"
            onClick={() => setLang("en")}
          >
            {dict.common.english}
          </Button>
          <Button
            variant={lang === "bn" ? "primary" : "secondary"}
            size="sm"
            onClick={() => setLang("bn")}
          >
            {dict.common.bangla}
          </Button>
        </div>
        <p className="mt-2 text-micro text-ink-mute">
          (users.locale has no write endpoint yet; the choice is stored in your browser.)
        </p>
      </section>
    </div>
  );
}

function NumberField({
  id,
  label,
  value,
  onCommit,
}: {
  id: string;
  label: string;
  value: number;
  onCommit: (v: number) => void;
}) {
  const [local, setLocal] = useState(String(value));
  useEffect(() => setLocal(String(value)), [value]);
  return (
    <Input
      id={id}
      type="number"
      min={0}
      label={label}
      value={local}
      onChange={(e) => setLocal(e.target.value)}
      onBlur={() => {
        const n = Number(local);
        if (Number.isFinite(n) && n >= 0 && n !== value) onCommit(n);
      }}
    />
  );
}

function ToggleField({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label className="flex items-center gap-2 pt-6 text-body-md text-ink-secondary">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="h-4 w-4 accent-[var(--color-primary)]"
      />
      {label}
    </label>
  );
}
