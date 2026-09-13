"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, { AIProviderConfig, AIStageBinding, AIToolRouteResponse, AIToolStage, StudioTool } from "@/lib/api";
import { Activity, CheckCircle2, Plus, RefreshCw, Save, ShieldCheck, Trash2, Zap } from "lucide-react";

const panel: React.CSSProperties = { background: "rgba(95,114,249,.05)", border: "1px solid rgba(95,114,249,.16)", borderRadius: 14, padding: 18 };
const input: React.CSSProperties = { background: "rgba(255,255,255,.04)", border: "1px solid rgba(95,114,249,.25)", color: "#e2e8ff", borderRadius: 8, padding: "8px 10px" };

const MICROS_PER_DOLLAR = 1_000_000;

function defaultLimits(queue: AIToolStage["queue_class"]) {
  switch (queue) {
    case "HEAVY_ASYNC": return { maxConcurrent: 4, rpm: 30 };
    case "ASYNC": return { maxConcurrent: 12, rpm: 120 };
    case "BACKGROUND": return { maxConcurrent: 4, rpm: 60 };
    default: return { maxConcurrent: 32, rpm: 300 };
  }
}

function providerCostTier(p?: AIProviderConfig): AIStageBinding["cost_tier"] {
  const cost = p?.cost_micros || 0;
  if (cost === 0) return "FREE";
  return cost < 10_000 ? "LOW_COST" : "PREMIUM";
}

export default function AIRoutingPage() {
  const [tools, setTools] = useState<StudioTool[]>([]);
  const [providers, setProviders] = useState<AIProviderConfig[]>([]);
  const [slug, setSlug] = useState("");
  const [route, setRoute] = useState<AIToolRouteResponse | null>(null);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  const loadBase = useCallback(async () => {
    const [toolRes, providerRes] = await Promise.all([adminAPI.getStudioTools(), adminAPI.getAIProviders()]);
    setTools(toolRes.tools);
    setProviders(providerRes.providers);
    setSlug(s => s || toolRes.tools[0]?.slug || "");
  }, []);

  const loadRoute = useCallback(async (toolSlug: string) => {
    if (!toolSlug) return;
    setBusy(true);
    try { setRoute(await adminAPI.getAIToolRoute(toolSlug)); setMessage(""); }
    catch (e) { setMessage(e instanceof Error ? e.message : "Failed to load route"); }
    finally { setBusy(false); }
  }, []);
  useEffect(() => { loadBase().catch(e => setMessage(String(e))); }, [loadBase]);
  useEffect(() => { if (slug) loadRoute(slug); }, [slug, loadRoute]);

  const activeTool = useMemo(() => tools.find(t => t.slug === slug), [tools, slug]);

  const validate = async () => {
    setBusy(true);
    try { await adminAPI.validateAIToolRoute(slug); setMessage("Route valid: every required stage has an active provider."); }
    catch (e) { setMessage(e instanceof Error ? e.message : "Route validation failed"); }
    finally { setBusy(false); }
  };

  const saveStage = async (stage: AIToolStage, patch: Partial<AIToolStage>) => {
    setBusy(true);
    try { await adminAPI.updateAIStage(stage.id, patch); await loadRoute(slug); setMessage("Stage policy updated."); }
    catch (e) { setMessage(e instanceof Error ? e.message : "Stage update failed"); }
    finally { setBusy(false); }
  };

  const saveBinding = async (binding: AIStageBinding, patch: Partial<AIStageBinding>) => {
    setBusy(true);
    try { await adminAPI.updateAIStageBinding(binding.id, patch); await loadRoute(slug); setMessage("Provider route updated."); }
    catch (e) { setMessage(e instanceof Error ? e.message : "Binding update failed"); }
    finally { setBusy(false); }
  };

  const removeBinding = async (id: string) => {
    if (!confirm("Remove this provider from the tool stage?")) return;
    setBusy(true);
    try { await adminAPI.deleteAIStageBinding(id); await loadRoute(slug); setMessage("Provider removed from route."); }
    catch (e) { setMessage(e instanceof Error ? e.message : "Delete failed"); }
    finally { setBusy(false); }
  };
  const addProvider = async (stage: AIToolStage, providerId: string) => {
    if (!providerId) return;
    setBusy(true);
    try {
      const p = providers.find(x => x.id === providerId);
      const limits = defaultLimits(stage.queue_class);
      await adminAPI.createAIStageBinding(stage.id, {
        provider_id: providerId,
        priority: 100,
        cost_tier: providerCostTier(p),
        max_concurrent: limits.maxConcurrent,
        requests_per_minute: limits.rpm,
        timeout_ms: 120000,
        max_retries: 0,
        allow_paid_fallback: true,
        circuit_failure_threshold: 5,
        circuit_open_seconds: 60,
      });
      await loadRoute(slug);
      setMessage("Provider added with surge guardrails. Review priority, budget and circuit policy before production use.");
    } catch (e) { setMessage(e instanceof Error ? e.message : "Add provider failed"); }
    finally { setBusy(false); }
  };

  return <AdminShell>
    <div style={{ padding: 28, color: "#e2e8ff", maxWidth: 1400, margin: "0 auto" }}>
      <div style={{ display: "flex", justifyContent: "space-between", gap: 16, alignItems: "flex-start", marginBottom: 22 }}>
        <div>
          <h1 style={{ margin: 0, fontSize: 28 }}>AI Routing Control Plane</h1>
          <p style={{ color: "#828cb4", marginTop: 7 }}>Per-tool, per-stage provider/model routing with free-first policies and surge protection.</p>
        </div>
        <div style={{ display: "flex", gap: 8 }}>
          <button onClick={() => loadRoute(slug)} style={input}><RefreshCw size={14} /> Refresh</button>
          <button onClick={validate} style={{ ...input, background: "#5f72f9", border: 0 }}><ShieldCheck size={14} /> Validate Route</button>
        </div>
      </div>

      <div style={{ ...panel, display: "grid", gridTemplateColumns: "minmax(260px,420px) 1fr", gap: 18, marginBottom: 18 }}>
        <div>
          <label style={{ color: "#828cb4", fontSize: 12 }}>STUDIO TOOL</label>
          <select value={slug} onChange={e => setSlug(e.target.value)} style={{ ...input, width: "100%", marginTop: 6 }}>
            {tools.map(t => <option key={t.id} value={t.slug}>{t.name} — {t.slug}</option>)}
          </select>
        </div>
        <div style={{ display: "flex", gap: 22, alignItems: "center", flexWrap: "wrap" }}>
          <span><strong>{activeTool?.name || "—"}</strong><br/><small style={{ color: "#828cb4" }}>Execution profile: {activeTool?.execution_profile || "legacy/unset"}</small></span>
          <span><Activity size={15} style={{ verticalAlign: "middle" }}/> {route?.stages.length || 0} stage(s)</span>
          <span><Zap size={15} style={{ verticalAlign: "middle" }}/> {route?.stages.reduce((n,s) => n + s.candidates.filter(c => c.binding.is_active).length, 0) || 0} active candidates</span>
        </div>
      </div>

      {message && <div style={{ ...panel, marginBottom: 18, borderColor: message.toLowerCase().includes("failed") ? "#f87171" : "#34d399" }}>{message}</div>}
      {busy && <div style={{ color: "#828cb4", marginBottom: 12 }}>Applying routing configuration…</div>}

      <div style={{ display: "grid", gap: 18 }}>
        {route?.stages.map(({ stage, candidates }) => <StageCard key={stage.id} stage={stage} candidates={candidates} providers={providers} onSaveStage={saveStage} onSaveBinding={saveBinding} onDelete={removeBinding} onAdd={addProvider} />)}
      </div>
      {!busy && route && route.stages.length === 0 && <div style={panel}>No routing stages exist for this tool. Keep it unavailable until a valid stage is configured.</div>}
    </div>
  </AdminShell>;
}

function StageCard({ stage, candidates, providers, onSaveStage, onSaveBinding, onDelete, onAdd }: {
  stage: AIToolStage;
  candidates: AIToolRouteResponse["stages"][number]["candidates"];
  providers: AIProviderConfig[];
  onSaveStage: (s: AIToolStage, p: Partial<AIToolStage>) => Promise<void>;
  onSaveBinding: (b: AIStageBinding, p: Partial<AIStageBinding>) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  onAdd: (s: AIToolStage, p: string) => Promise<void>;
}) {
  const [policy, setPolicy] = useState(stage.routing_policy);
  const [queue, setQueue] = useState(stage.queue_class);
  const [maxQueue, setMaxQueue] = useState(stage.max_queue_seconds || 0);
  const [hourlyBudget, setHourlyBudget] = useState((stage.paid_hourly_budget_micros || 0) / MICROS_PER_DOLLAR);
  const [dailyBudget, setDailyBudget] = useState((stage.paid_daily_budget_micros || 0) / MICROS_PER_DOLLAR);
  const [newProvider, setNewProvider] = useState("");

  useEffect(() => {
    setPolicy(stage.routing_policy);
    setQueue(stage.queue_class);
    setMaxQueue(stage.max_queue_seconds || 0);
    setHourlyBudget((stage.paid_hourly_budget_micros || 0) / MICROS_PER_DOLLAR);
    setDailyBudget((stage.paid_daily_budget_micros || 0) / MICROS_PER_DOLLAR);
  }, [stage]);

  const used = new Set(candidates.map(c => c.provider.id));
  const available = providers.filter(p => !used.has(p.id));
  const saveStagePolicy = () => onSaveStage(stage, {
    routing_policy: policy,
    queue_class: queue,
    max_queue_seconds: Math.max(0, maxQueue || 0),
    paid_hourly_budget_micros: Math.round(Math.max(0, hourlyBudget || 0) * MICROS_PER_DOLLAR),
    paid_daily_budget_micros: Math.round(Math.max(0, dailyBudget || 0) * MICROS_PER_DOLLAR),
  });

  return <section style={panel}>
    <div style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap" }}>
      <div>
        <h2 style={{ margin: 0, fontSize: 18 }}>{stage.stage_key}</h2>
        <div style={{ color: "#828cb4", fontSize: 12 }}>{stage.capability}</div>
      </div>
      <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
        <select value={policy} onChange={e => setPolicy(e.target.value as AIToolStage["routing_policy"])} style={input}>
          {["FREE_FIRST","BALANCED","QUALITY_FIRST","FREE_ONLY","PREMIUM_ONLY"].map(v => <option key={v}>{v}</option>)}
        </select>
        <select value={queue} onChange={e => setQueue(e.target.value as AIToolStage["queue_class"])} style={input}>
          {["REALTIME","INTERACTIVE","ASYNC","HEAVY_ASYNC","BACKGROUND"].map(v => <option key={v}>{v}</option>)}
        </select>
        <label style={{ color: "#828cb4", fontSize: 11 }}>Queue wait s
          <input type="number" min={0} max={3600} value={maxQueue} onChange={e => setMaxQueue(Number(e.target.value))} style={{ ...input, width: 86, marginLeft: 5 }}/>
        </label>
        <label style={{ color: "#828cb4", fontSize: 11 }}>Paid $/hr
          <input type="number" min={0} step="0.01" value={hourlyBudget} onChange={e => setHourlyBudget(Number(e.target.value))} style={{ ...input, width: 95, marginLeft: 5 }}/>
        </label>
        <label style={{ color: "#828cb4", fontSize: 11 }}>Paid $/day
          <input type="number" min={0} step="0.01" value={dailyBudget} onChange={e => setDailyBudget(Number(e.target.value))} style={{ ...input, width: 100, marginLeft: 5 }}/>
        </label>
        <button style={input} onClick={saveStagePolicy}><Save size={14}/></button>
      </div>
    </div>
    <div style={{ color: "#687198", fontSize: 11, marginTop: 7 }}>
      Paid budget 0 = unlimited. Configured paid budgets fail closed if the distributed budget backend is unavailable.
    </div>

    <div style={{ marginTop: 16, overflowX: "auto" }}>
      <table style={{ width: "100%", borderCollapse: "collapse", minWidth: 1500 }}>
        <thead><tr style={{ textAlign: "left", color: "#828cb4", fontSize: 11 }}>
          <th>PRIORITY</th><th>PROVIDER / MODEL</th><th>COST</th><th>PAID FALLBACK</th>
          <th>MAX CONCURRENT</th><th>RPM</th><th>TIMEOUT</th><th>RETRIES</th>
          <th>CIRCUIT FAILS</th><th>COOLDOWN s</th><th>ACTIVE</th><th></th>
        </tr></thead>
        <tbody>{candidates.map(({ binding, provider }) =>
          <BindingRow key={binding.id} binding={binding} provider={provider} onSave={onSaveBinding} onDelete={onDelete} />
        )}</tbody>
      </table>
    </div>

    <div style={{ display: "flex", gap: 8, marginTop: 16 }}>
      <select value={newProvider} onChange={e => setNewProvider(e.target.value)} style={{ ...input, minWidth: 280 }}>
        <option value="">Add provider/model candidate…</option>
        {available.map(p => <option key={p.id} value={p.id}>{p.name} — {p.model_id || p.slug}</option>)}
      </select>
      <button disabled={!newProvider} onClick={() => { const id = newProvider; setNewProvider(""); onAdd(stage, id); }} style={{ ...input, opacity: newProvider ? 1 : .5 }}>
        <Plus size={14}/> Add
      </button>
    </div>
  </section>;
}
function BindingRow({ binding, provider, onSave, onDelete }: {
  binding: AIStageBinding;
  provider: AIProviderConfig;
  onSave: (b: AIStageBinding, p: Partial<AIStageBinding>) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}) {
  const [draft, setDraft] = useState(binding);
  useEffect(() => setDraft(binding), [binding]);
  const num = (k: keyof AIStageBinding, v: string) => setDraft(d => ({ ...d, [k]: Number(v) }));

  return <tr style={{ borderTop: "1px solid rgba(95,114,249,.12)" }}>
    <td><input type="number" min={1} value={draft.priority} onChange={e => num("priority", e.target.value)} style={{ ...input, width: 75 }}/></td>
    <td style={{ padding: "12px 8px" }}>
      <strong>{provider.name}</strong><br/>
      <code style={{ color: "#a5b4fc", fontSize: 11 }}>{provider.model_id || provider.slug}</code>
    </td>
    <td>
      <select value={draft.cost_tier} onChange={e => setDraft(d => ({ ...d, cost_tier: e.target.value as AIStageBinding["cost_tier"] }))} style={input}>
        {["FREE","LOW_COST","PREMIUM"].map(v => <option key={v}>{v}</option>)}
      </select>
    </td>
    <td>
      <button onClick={() => setDraft(d => ({ ...d, allow_paid_fallback: !d.allow_paid_fallback }))} style={input}>
        {draft.allow_paid_fallback ? "Allowed" : "Blocked"}
      </button>
    </td>
    <td><input type="number" min={0} max={10000} value={draft.max_concurrent} onChange={e => num("max_concurrent", e.target.value)} style={{ ...input, width: 90 }}/></td>
    <td><input type="number" min={0} max={100000} value={draft.requests_per_minute} onChange={e => num("requests_per_minute", e.target.value)} style={{ ...input, width: 90 }}/></td>
    <td><input type="number" min={1000} max={600000} step={1000} value={draft.timeout_ms} onChange={e => num("timeout_ms", e.target.value)} style={{ ...input, width: 105 }}/></td>
    <td><input type="number" min={0} max={3} value={draft.max_retries} onChange={e => num("max_retries", e.target.value)} style={{ ...input, width: 70 }}/></td>
    <td><input type="number" min={1} max={100} value={draft.circuit_failure_threshold} onChange={e => num("circuit_failure_threshold", e.target.value)} style={{ ...input, width: 85 }}/></td>
    <td><input type="number" min={1} max={86400} value={draft.circuit_open_seconds} onChange={e => num("circuit_open_seconds", e.target.value)} style={{ ...input, width: 95 }}/></td>
    <td>
      <button onClick={() => setDraft(d => ({ ...d, is_active: !d.is_active }))} style={input}>
        {draft.is_active ? <CheckCircle2 size={16}/> : "Off"}
      </button>
    </td>
    <td style={{ display: "flex", gap: 5, padding: "10px 0" }}>
      <button onClick={() => onSave(binding, draft)} style={input}><Save size={14}/></button>
      <button onClick={() => onDelete(binding.id)} style={{ ...input, color: "#f87171" }}><Trash2 size={14}/></button>
    </td>
  </tr>;
}