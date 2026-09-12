"use client";

import { useCallback, useEffect, useState } from "react";
import { api, ApiError } from "@/lib/api";
import type { DomainWhitelistRow } from "@rexio-pay/shared-types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useI18n } from "@/lib/i18n";

/** Domain Whitelist (REQUIREMENT §14.10): CRUD; duplicates 409 handled. */
export default function DomainsPage() {
  const { dict } = useI18n();
  const m = dict.modules;
  const [rows, setRows] = useState<DomainWhitelistRow[]>([]);
  const [domain, setDomain] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    try {
      const list = await api.get<{ data: DomainWhitelistRow[] }>("/domain-whitelist");
      setRows(list.data);
    } catch {
      // shell handles disconnect
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function add(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      await api.post("/domain-whitelist", { domain: domain.trim() });
      setDomain("");
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    }
  }

  async function remove(id: string) {
    setError(null);
    try {
      await api.delete(`/domain-whitelist/${id}`);
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    }
  }

  return (
    <div>
      <h1 className="text-display-lg text-ink">{m.domainsTitle}</h1>
      <p className="mt-3 text-body-md text-ink-secondary">{dict.onboard.domainsIntro}</p>

      <form onSubmit={add} className="mt-6 flex items-end gap-4">
        <Input
          id="dm"
          label={m.domainsTitle}
          placeholder={dict.onboard.domainsPlaceholder}
          value={domain}
          onChange={(e) => setDomain(e.target.value)}
          required
        />
        <Button type="submit" disabled={!domain.trim()}>
          {dict.onboard.domainsAdd}
        </Button>
      </form>
      {error ? <p className="mt-3 text-caption text-ruby">{error}</p> : null}

      {loading ? (
        <p className="mt-8 text-body-md text-ink-mute">{dict.common.loading}</p>
      ) : rows.length === 0 ? (
        <p className="mt-8 text-body-md text-ink-mute">{m.domainsEmpty}</p>
      ) : (
        <ul className="mt-6 flex flex-col gap-2">
          {rows.map((r) => (
            <li
              key={r.id}
              className="flex items-center justify-between rounded-lg border border-hairline bg-canvas px-4 py-3"
            >
              <span className="tnum text-body-md text-ink">{r.domain}</span>
              <Button variant="ghost" size="sm" onClick={() => remove(r.id)}>
                {m.domainsRemove}
              </Button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
