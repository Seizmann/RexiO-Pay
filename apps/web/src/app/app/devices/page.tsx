"use client";

import { useCallback, useEffect, useState } from "react";
import { api, ApiError, isPlanLimitError } from "@/lib/api";
import type { Device } from "@rexio-pay/shared-types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { StatusBadge } from "@/components/ui/badge";
import { useI18n } from "@/lib/i18n";

/**
 * Devices (REQUIREMENT §14.4): list + pair via QR + disable + re-pair.
 * The QR payload matches the Android app contract from Session 3:
 * canonical JSON {"server_url","merchant_id","pairing_token"}.
 */
export default function DevicesPage() {
  const { dict, lang } = useI18n();
  const m = dict.modules;
  const [rows, setRows] = useState<Device[]>([]);
  const [pairing, setPairing] = useState<Device | null>(null);
  const [qrDataUrl, setQrDataUrl] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("My phone");
  const [limitHit, setLimitHit] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    try {
      const list = await api.get<{ data: Device[] }>("/devices");
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

  async function createDevice(e: React.FormEvent) {
    e.preventDefault();
    setCreating(true);
    setError(null);
    try {
      const created = await api.post<Device>("/devices", { name });
      setPairing(created);
      await showQr(created);
      await load();
    } catch (err) {
      if (isPlanLimitError(err)) {
        setLimitHit(true);
      } else {
        setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
      }
    } finally {
      setCreating(false);
    }
  }

  async function showQr(device: Device) {
    if (!device.pairing_token) return;
    const payload = {
      server_url: process.env.NEXT_PUBLIC_API_BASE_URL ?? "https://api.pay.rexio.pro",
      merchant_id: device.pairing_token.split("|")[0] ?? "",
      pairing_token: device.pairing_token,
    };
    // merchant_id comes from /branding; the create response has no merchant id,
    // so fetch it once per QR render.
    try {
      const merchant = await api.get<{ id: string }>("/branding");
      payload.merchant_id = merchant.id;
    } catch {
      // keep empty; QR will show a token-only payload
    }
    const QRCode = (await import("qrcode")).default;
    setQrDataUrl(await QRCode.toDataURL(JSON.stringify(payload), { width: 256, margin: 1 }));
  }

  async function repair(d: Device) {
    setError(null);
    try {
      const updated = await api.post<Device>(`/devices/${d.id}/repair`);
      setPairing({ ...d, pairing_token: updated.pairing_token });
      await showQr({ ...d, pairing_token: updated.pairing_token });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    }
  }

  async function disable(d: Device) {
    setError(null);
    try {
      await api.post(`/devices/${d.id}/disable`);
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : dict.common.errorGeneric);
    }
  }

  return (
    <div>
      <h1 className="text-display-lg text-ink">{m.devicesTitle}</h1>
      {limitHit ? (
        <p className="mt-3 rounded-md bg-canvas-cream px-4 py-3 text-caption text-ink">
          {dict.common.errorPlanLimit}
        </p>
      ) : null}
      {error ? <p className="mt-3 text-caption text-ruby">{error}</p> : null}

      <form onSubmit={createDevice} className="mt-6 flex flex-wrap items-end gap-4">
        <Input id="dev-name" label={m.keysName} value={name} onChange={(e) => setName(e.target.value)} className="w-64" />
        <Button type="submit" disabled={creating || !name.trim()}>
          {creating ? dict.common.loading : m.devicesAdd}
        </Button>
      </form>

      {pairing?.pairing_token && qrDataUrl ? (
        <div className="mt-6 flex flex-col items-start gap-3 rounded-lg border border-hairline bg-canvas p-6">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={qrDataUrl} alt={m.devicesAdd} width={256} height={256} className="rounded-md border border-hairline" />
          <p className="text-caption text-ink-mute">{dict.onboard.deviceScanWith}</p>
          <Button variant="secondary" size="sm" onClick={() => repair(pairing)}>
            {m.devicesRepair}
          </Button>
        </div>
      ) : null}

      {loading ? (
        <p className="mt-8 text-body-md text-ink-mute">{dict.common.loading}</p>
      ) : rows.length === 0 ? (
        <p className="mt-8 text-body-md text-ink-mute">{m.devicesEmpty}</p>
      ) : (
        <div className="mt-6 overflow-x-auto rounded-lg border border-hairline bg-canvas">
          <table className="w-full min-w-3xl text-body-tabular">
            <thead>
              <tr className="border-b border-hairline bg-canvas-soft text-left text-ink-mute">
                <th className="px-4 py-3 font-normal">{m.keysName}</th>
                <th className="px-4 py-3 font-normal">{m.txStatus}</th>
                <th className="px-4 py-3 font-normal">{m.devicesLastBeat}</th>
                <th className="px-4 py-3 font-normal">{m.devicesLastSync}</th>
                <th className="px-4 py-3 font-normal">{m.devicesSigFails}</th>
                <th className="px-4 py-3 font-normal">{m.profilesActions}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((d) => (
                <tr key={d.id} className="border-b border-hairline last:border-0">
                  <td className="px-4 py-3 text-ink">
                    {d.name}
                    {d.model ? <span className="text-ink-mute"> · {d.model}</span> : null}
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge
                      tone={d.status === "active" ? "succeeded" : d.status === "offline" ? "danger" : "muted"}
                      label={dict.status[d.status as keyof typeof dict.status] ?? d.status}
                    />
                  </td>
                  <td className="px-4 py-3 text-ink-secondary">
                    {d.last_heartbeat_at
                      ? new Date(d.last_heartbeat_at).toLocaleTimeString(lang === "bn" ? "bn-BD" : "en-GB")
                      : "—"}
                  </td>
                  <td className="px-4 py-3 text-ink-secondary">
                    {d.last_sms_synced_at
                      ? new Date(d.last_sms_synced_at).toLocaleTimeString(lang === "bn" ? "bn-BD" : "en-GB")
                      : "—"}
                  </td>
                  <td className="tnum px-4 py-3">{d.failed_sig_count}</td>
                  <td className="flex gap-2 px-4 py-3">
                    <Button variant="ghost" size="sm" onClick={() => repair(d)}>
                      {m.devicesRepair}
                    </Button>
                    {d.status !== "disabled" ? (
                      <Button variant="ghost" size="sm" onClick={() => disable(d)}>
                        {m.devicesDisable}
                      </Button>
                    ) : null}
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
