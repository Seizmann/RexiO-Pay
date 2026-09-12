"use client";

import { useCallback, useEffect, useState } from "react";
import { api, ApiError } from "@/lib/api";
import type { Announcement } from "@rexio-pay/shared-types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useI18n } from "@/lib/i18n";

/** Announcements CRUD (REQUIREMENT §17): broadcast banners shown to merchants. */
export default function AdminAnnouncementsPage() {
  const { dict, lang } = useI18n();
  const a = dict.admin;
  const [rows, setRows] = useState<Announcement[]>([]);
  const [form, setForm] = useState({
    title: "",
    body: "",
    severity: "info",
    expires_at: "",
  });
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const res = await api.get<{ data: Announcement[] }>("/rexio-admin/announcements");
      setRows(res.data);
    } catch {
      // show empty
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await api.post("/rexio-admin/announcements", {
        title: form.title,
        body: form.body,
        severity: form.severity,
        expires_at: form.expires_at ? new Date(form.expires_at).toISOString() : undefined,
      });
      setForm({ title: "", body: "", severity: "info", expires_at: "" });
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    } finally {
      setBusy(false);
    }
  }

  async function remove(id: string) {
    setError(null);
    try {
      await api.delete(`/rexio-admin/announcements/${id}`);
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    }
  }

  return (
    <div>
      <h1 className="text-display-lg text-ink">{a.announcements}</h1>

      <form onSubmit={create} className="mt-6 flex flex-col gap-4 rounded-lg border border-hairline bg-canvas p-6">
        <Input id="an-title" label={a.announcementTitle} value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} required />
        <label className="flex flex-col gap-1.5">
          <span className="text-caption text-ink-secondary">{a.announcementBody}</span>
          <textarea
            value={form.body}
            onChange={(e) => setForm({ ...form, body: e.target.value })}
            required
            rows={3}
            className="rounded-sm border border-hairline-input bg-canvas px-3 py-2 text-body-md text-ink focus:border-primary focus:outline-none"
          />
        </label>
        <div className="flex flex-wrap items-end gap-4">
          <label className="flex flex-col gap-1.5">
            <span className="text-caption text-ink-secondary">{a.announcementSeverity}</span>
            <select
              value={form.severity}
              onChange={(e) => setForm({ ...form, severity: e.target.value })}
              className="h-10 rounded-sm border border-hairline-input bg-canvas px-3 text-body-md"
            >
              <option value="info">info</option>
              <option value="success">success</option>
              <option value="warning">warning</option>
              <option value="error">error</option>
            </select>
          </label>
          <Input
            id="an-expires"
            type="date"
            label={a.announcementExpires}
            value={form.expires_at}
            onChange={(e) => setForm({ ...form, expires_at: e.target.value })}
            className="w-44"
          />
          <Button type="submit" disabled={busy || !form.title || !form.body}>
            {busy ? dict.common.saving : a.announcementCreate}
          </Button>
        </div>
        {error ? <p className="text-caption text-ruby">{error}</p> : null}
      </form>

      {rows.length === 0 ? (
        <p className="mt-8 text-body-md text-ink-mute">{dict.common.empty}</p>
      ) : (
        <ul className="mt-6 flex flex-col gap-3">
          {rows.map((r) => (
            <li key={r.id} className="rounded-lg border border-hairline bg-canvas p-4">
              <div className="flex items-center justify-between gap-4">
                <div>
                  <p className="text-body-md text-ink">{r.title}</p>
                  <p className="mt-1 text-caption text-ink-secondary">{r.body}</p>
                  <p className="mt-1 text-micro text-ink-mute">
                    {r.severity} ·{" "}
                    {new Date(r.created_at).toLocaleDateString(lang === "bn" ? "bn-BD" : "en-GB")}
                    {r.expires_at
                      ? ` · ${dict.settings.renews}: ${new Date(r.expires_at).toLocaleDateString()}`
                      : ""}
                  </p>
                </div>
                <Button variant="ghost" size="sm" onClick={() => remove(r.id)}>
                  {a.announcementDelete}
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
