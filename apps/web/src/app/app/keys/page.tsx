"use client";

import { useCallback, useEffect, useState } from "react";
import { api, ApiError } from "@/lib/api";
import type { ApiKeyRow } from "@rexio-pay/shared-types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useCopy } from "@/lib/use-copy";
import { useI18n } from "@/lib/i18n";

/**
 * API Keys (REQUIREMENT §14.7): create with a show-once modal, list with
 * prefix only, revoke. The backend returns the raw key exactly once.
 */
export default function KeysPage() {
  const { dict, lang } = useI18n();
  const m = dict.modules;
  const { copied, copy } = useCopy();
  const [rows, setRows] = useState<ApiKeyRow[]>([]);
  const [name, setName] = useState("");
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    try {
      const list = await api.get<{ data: ApiKeyRow[] }>("/api-keys");
      setRows(list.data.filter((k) => !k.revoked_at));
    } catch {
      // shell handles disconnect
    } finally {
      setLoading(false);
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
      const res = await api.post<ApiKeyRow>("/api-keys", { name });
      if (res.key) setCreatedKey(res.key);
      setName("");
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    } finally {
      setBusy(false);
    }
  }

  async function revoke(id: string) {
    setError(null);
    try {
      await api.delete(`/api-keys/${id}`);
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    }
  }

  return (
    <div>
      <h1 className="text-display-lg text-ink">{m.keysTitle}</h1>

      {createdKey ? (
        <div className="mt-6 rounded-lg border border-primary bg-canvas p-6">
          <p className="text-body-md text-ink">{m.keysShowOnce}</p>
          <div className="mt-3 flex items-center gap-3">
            <code className="tnum flex-1 overflow-x-auto rounded-sm bg-canvas-soft px-3 py-2 text-body-tabular text-ink">
              {createdKey}
            </code>
            <Button variant="secondary" size="sm" onClick={() => copy(createdKey, "key")}>
              {copied === "key" ? dict.common.copied : dict.common.copy}
            </Button>
          </div>
          <Button variant="ghost" size="sm" className="mt-3" onClick={() => setCreatedKey(null)}>
            {dict.common.close}
          </Button>
        </div>
      ) : null}

      <form onSubmit={create} className="mt-6 flex flex-wrap items-end gap-4">
        <Input id="key-name" label={m.keysName} value={name} onChange={(e) => setName(e.target.value)} required />
        <Button type="submit" disabled={busy || !name.trim()}>
          {busy ? dict.common.saving : m.keysCreate}
        </Button>
        {error ? <p className="text-caption text-ruby">{error}</p> : null}
      </form>

      {loading ? (
        <p className="mt-8 text-body-md text-ink-mute">{dict.common.loading}</p>
      ) : rows.length === 0 ? (
        <p className="mt-8 text-body-md text-ink-mute">{m.keysEmpty}</p>
      ) : (
        <div className="mt-6 overflow-x-auto rounded-lg border border-hairline bg-canvas">
          <table className="w-full min-w-2xl text-body-tabular">
            <thead>
              <tr className="border-b border-hairline bg-canvas-soft text-left text-ink-mute">
                <th className="px-4 py-3 font-normal">{m.keysName}</th>
                <th className="px-4 py-3 font-normal">{m.keysPrefix}</th>
                <th className="px-4 py-3 font-normal">{m.keysScopes}</th>
                <th className="px-4 py-3 font-normal">{m.keysLastUsed}</th>
                <th className="px-4 py-3 font-normal">{m.profilesActions}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((k) => (
                <tr key={k.id} className="border-b border-hairline last:border-0">
                  <td className="px-4 py-3 text-ink">{k.name}</td>
                  <td className="tnum px-4 py-3 text-ink-mute">{k.key_prefix}…</td>
                  <td className="px-4 py-3">{k.scopes}</td>
                  <td className="px-4 py-3 text-ink-secondary">
                    {k.last_used_at
                      ? new Date(k.last_used_at).toLocaleDateString(lang === "bn" ? "bn-BD" : "en-GB")
                      : "—"}
                  </td>
                  <td className="px-4 py-3">
                    <Button variant="ghost" size="sm" onClick={() => revoke(k.id)}>
                      {m.keysRevoke}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
