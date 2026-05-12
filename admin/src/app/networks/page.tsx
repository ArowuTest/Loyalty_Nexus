"use client";
import AdminShell from "@/components/layout/AdminShell";
import { useEffect, useState, useCallback } from "react";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

interface Network {
  id: string;
  network_code: string;
  network_name: string;
  logo_url: string;
  brand_color: string;
  is_active: boolean;
  airtime_enabled: boolean;
  data_enabled: boolean;
  min_amount_kobo: number;
  max_amount_kobo: number;
  sort_order: number;
}

const NETWORK_COLORS: Record<string, string> = {
  MTN: "#FFCC00", GLO: "#00A651", AIRTEL: "#FF0000", "9MOBILE": "#00A859",
};
const NETWORK_EMOJIS: Record<string, string> = {
  MTN: "🟡", GLO: "🟢", AIRTEL: "🔴", "9MOBILE": "🟢",
};

export default function NetworksPage() {
  const [networks, setNetworks]   = useState<Network[]>([]);
  const [loading, setLoading]     = useState(true);
  const [saving, setSaving]       = useState<string | null>(null);
  const [error, setError]         = useState("");
  const [success, setSuccess]     = useState("");

  const token = typeof window !== "undefined"
    ? localStorage.getItem("admin_token") ?? ""
    : "";

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const r = await fetch(`${API}/admin/networks`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const d = await r.json();
      setNetworks(d.networks ?? []);
    } catch {
      setError("Failed to load network configurations.");
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => { load(); }, [load]);

  const toggle = async (code: string, field: "is_active" | "airtime_enabled" | "data_enabled", current: boolean) => {
    setSaving(code + "_" + field);
    setError(""); setSuccess("");
    try {
      const r = await fetch(`${API}/admin/networks/${code}`, {
        method: "PATCH",
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
        body: JSON.stringify({ [field]: !current }),
      });
      if (!r.ok) throw new Error("Update failed");
      setSuccess(`${code} ${field.replace(/_/g, " ")} updated`);
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Update failed");
    } finally {
      setSaving(null);
    }
  };

  return (
    <AdminShell title="Network Operators" subtitle="Enable/disable telecom networks and their services. Changes take effect instantly — no deployment needed.">
      {error   && <div className="mb-4 p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-sm">{error}</div>}
      {success && <div className="mb-4 p-3 rounded-lg bg-green-500/10 border border-green-500/20 text-green-400 text-sm">✓ {success}</div>}

      {loading ? (
        <div className="flex items-center gap-2 text-gray-400 py-8">
          <div className="w-4 h-4 rounded-full border-2 border-gray-400 border-t-transparent animate-spin" />
          Loading network configurations…
        </div>
      ) : (
        <div className="space-y-4">
          {/* Info banner */}
          <div className="rounded-xl bg-blue-500/8 border border-blue-500/20 p-4 text-sm text-blue-300">
            <strong className="text-blue-200">Admin-only toggle:</strong> Networks with{" "}
            <span className="font-bold text-white">Active = OFF</span> are completely hidden from
            the recharge page. Toggle{" "}
            <span className="font-bold text-white">Airtime/Data</span> to control which service
            types are visible per network. MTN is enabled at launch; others are ready to activate
            when you onboard new telco partners — no code deploy required.
          </div>

          {/* Networks table */}
          <div className="rounded-2xl border border-white/[0.08] overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-white/[0.06] bg-white/[0.02]">
                  {["Network", "Active", "Airtime", "Data", "Min Amount", "Max Amount"].map(h => (
                    <th key={h} className="text-left px-4 py-3 text-xs font-bold text-white/40 uppercase tracking-wider">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-white/[0.04]">
                {networks.map(net => (
                  <tr key={net.network_code} className="hover:bg-white/[0.02] transition-colors">
                    {/* Network name */}
                    <td className="px-4 py-4">
                      <div className="flex items-center gap-3">
                        <span className="text-2xl">{NETWORK_EMOJIS[net.network_code] ?? "📱"}</span>
                        <div>
                          <div className="font-bold text-white text-sm" style={{ color: NETWORK_COLORS[net.network_code] }}>
                            {net.network_code}
                          </div>
                          <div className="text-xs text-white/40">{net.network_name}</div>
                        </div>
                      </div>
                    </td>

                    {/* Active toggle */}
                    <td className="px-4 py-4">
                      <Toggle
                        value={net.is_active}
                        loading={saving === net.network_code + "_is_active"}
                        onChange={() => toggle(net.network_code, "is_active", net.is_active)}
                        color={NETWORK_COLORS[net.network_code]}
                      />
                    </td>

                    {/* Airtime toggle */}
                    <td className="px-4 py-4">
                      <Toggle
                        value={net.airtime_enabled}
                        loading={saving === net.network_code + "_airtime_enabled"}
                        onChange={() => toggle(net.network_code, "airtime_enabled", net.airtime_enabled)}
                        disabled={!net.is_active}
                      />
                    </td>

                    {/* Data toggle */}
                    <td className="px-4 py-4">
                      <Toggle
                        value={net.data_enabled}
                        loading={saving === net.network_code + "_data_enabled"}
                        onChange={() => toggle(net.network_code, "data_enabled", net.data_enabled)}
                        disabled={!net.is_active}
                      />
                    </td>

                    {/* Min/max amounts */}
                    <td className="px-4 py-4 text-sm text-white/60">₦{Math.floor(net.min_amount_kobo / 100).toLocaleString()}</td>
                    <td className="px-4 py-4 text-sm text-white/60">₦{Math.floor(net.max_amount_kobo / 100).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </AdminShell>
  );
}

function Toggle({ value, loading, onChange, disabled = false, color = "#F5A623" }: {
  value: boolean; loading: boolean; onChange: () => void; disabled?: boolean; color?: string;
}) {
  return (
    <button
      onClick={onChange}
      disabled={loading || disabled}
      className={`relative w-11 h-6 rounded-full transition-all ${
        disabled ? "opacity-30 cursor-not-allowed" :
        value ? "bg-green-500/20 border border-green-500/40" : "bg-white/[0.05] border border-white/[0.10]"
      }`}
    >
      {loading ? (
        <div className="absolute inset-0 flex items-center justify-center">
          <div className="w-3 h-3 rounded-full border border-white/40 border-t-transparent animate-spin" />
        </div>
      ) : (
        <div className={`absolute top-0.5 w-5 h-5 rounded-full transition-all ${
          value ? "left-[22px] bg-green-400" : "left-0.5 bg-white/20"
        }`} />
      )}
    </button>
  );
}
