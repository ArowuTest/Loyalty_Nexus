"use client";

import { useEffect, useState, useCallback } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, {
  AIHealthReport, AIProviderStatus, ProviderSwitchEvent, StudioToolHealth, Generation,
} from "@/lib/api";
import {
  RefreshCw, CheckCircle2, AlertTriangle, XCircle, Clock,
  Zap, ArrowRight, Activity, Brain, Wand2, BarChart3,
  AlertCircle, WifiOff, Radio, TrendingDown,
} from "lucide-react";

// ─── Helpers ──────────────────────────────────────────────────────────────────
function timeAgo(isoOrNull: string | null | undefined): string {
  if (!isoOrNull) return "Never";
  const diff = Math.floor((Date.now() - new Date(isoOrNull).getTime()) / 1000);
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

function switchTs(ts: number): string {
  return new Date(ts * 1000).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

function isLive(isoOrNull: string | null | undefined): boolean {
  if (!isoOrNull) return false;
  return (Date.now() - new Date(isoOrNull).getTime()) < 5 * 60 * 1000; // 5 minutes
}

const PROVIDER_META: Record<string, { label: string; model: string; limit: string; color: string; bgColor: string }> = {
  GROQ:        { label: "Groq",        model: "Llama-4-Scout 17B",   limit: "6,000 req/min free",   color: "text-green-600",  bgColor: "bg-green-50 border-green-200" },
  GEMINI_LITE: { label: "Gemini",      model: "Flash-Lite 2.0",      limit: "1,500 req/day free",   color: "text-blue-600",   bgColor: "bg-blue-50 border-blue-200" },
  DEEPSEEK:    { label: "DeepSeek",    model: "V3 (deepseek-chat)",  limit: "Pay-per-use fallback", color: "text-purple-600", bgColor: "bg-purple-50 border-purple-200" },
};

// ─── Status badge ─────────────────────────────────────────────────────────────
function StatusBadge({ status }: { status: AIProviderStatus["status"] }) {
  const config = {
    ok:            { icon: <CheckCircle2 size={13} />, label: "Healthy",       cls: "bg-green-100 text-green-700 border-green-200" },
    error:         { icon: <XCircle size={13} />,      label: "Error",         cls: "bg-red-100 text-red-700 border-red-200" },
    limit_reached: { icon: <AlertTriangle size={13} />,label: "Limit reached", cls: "bg-amber-100 text-amber-700 border-amber-200" },
    unknown:       { icon: <Clock size={13} />,        label: "Not used yet",  cls: "bg-gray-100 text-gray-500 border-gray-200" },
  }[status];
  return (
    <span className={`inline-flex items-center gap-1 text-xs font-semibold px-2.5 py-1 rounded-full border ${config.cls}`}>
      {config.icon} {config.label}
    </span>
  );
}

// ─── Active indicator ─────────────────────────────────────────────────────────
function ActivePulse() {
  return (
    <span className="relative flex h-2.5 w-2.5">
      <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
      <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-green-500" />
    </span>
  );
}

// ─── Provider card ────────────────────────────────────────────────────────────
function ProviderCard({ provider, isActive }: { provider: AIProviderStatus; isActive: boolean }) {
  const meta = PROVIDER_META[provider.name] ?? {
    label: provider.name, model: "—", limit: "—",
    color: "text-gray-600", bgColor: "bg-gray-50 border-gray-200",
  };
  const isDown = provider.status === "error" || provider.status === "limit_reached";

  return (
    <div className={`rounded-2xl border-2 p-5 space-y-4 transition-all ${
      isActive
        ? "border-green-400 ring-2 ring-green-400/20 shadow-md bg-white"
        : isDown
          ? "border-red-200 bg-red-50/30"
          : "border-gray-200 bg-white"
    }`}>
      {/* Header row */}
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-center gap-2.5">
          {isActive && <ActivePulse />}
          <div>
            <div className="flex items-center gap-2">
              <span className={`font-bold text-base ${meta.color}`}>{meta.label}</span>
              {isActive && (
                <span className="text-[10px] bg-green-100 text-green-700 border border-green-200 px-2 py-0.5 rounded-full font-bold uppercase tracking-wider">
                  ACTIVE
                </span>
              )}
            </div>
            <p className="text-gray-500 text-xs mt-0.5">{meta.model}</p>
          </div>
        </div>
        <StatusBadge status={provider.status} />
      </div>

      {/* Metrics grid */}
      <div className="grid grid-cols-2 gap-3">
        <div className={`rounded-xl p-3 border ${meta.bgColor}`}>
          <p className="text-gray-500 text-[10px] uppercase tracking-wider font-medium mb-1">Requests today</p>
          <p className={`font-bold text-2xl ${meta.color}`}>{provider.requests_today.toLocaleString()}</p>
          <p className="text-gray-400 text-[10px] mt-0.5">{meta.limit}</p>
        </div>
        <div className="rounded-xl p-3 border border-gray-100 bg-gray-50">
          <p className="text-gray-500 text-[10px] uppercase tracking-wider font-medium mb-1">Last used</p>
          <p className="font-semibold text-sm text-gray-700">{timeAgo(provider.last_used_at)}</p>
          <p className="text-gray-400 text-[10px] mt-0.5">
            {provider.last_used_at
              ? new Date(provider.last_used_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
              : "—"}
          </p>
        </div>
      </div>

      {/* Error box — improved */}
      {provider.last_error && (
        <div className="flex items-start gap-2 bg-red-50 border border-red-200 rounded-xl p-3">
          <AlertCircle size={14} className="text-red-500 flex-shrink-0 mt-0.5" />
          <div className="min-w-0">
            <p className="text-red-700 text-xs font-semibold mb-1">Last error</p>
            <p className="text-red-600 text-xs font-mono leading-relaxed break-words whitespace-pre-wrap">
              {provider.last_error}
            </p>
          </div>
        </div>
      )}
    </div>
  );
}

// ─── Switch event row ─────────────────────────────────────────────────────────
function SwitchRow({ event, index }: { event: ProviderSwitchEvent; index: number }) {
  const fromMeta = PROVIDER_META[event.from];
  const toMeta   = PROVIDER_META[event.to];
  const reasonLabel: Record<string, string> = {
    rate_limit: "Rate limit hit",
    error:      "Provider error",
    daily_cap:  "Daily cap reached",
    timeout:    "Timeout",
    "":         "Fallback",
  };
  return (
    <div className={`flex items-center gap-3 py-2.5 px-3 rounded-xl ${index % 2 === 0 ? "bg-gray-50" : ""}`}>
      <span className="text-gray-400 text-xs font-mono w-16 flex-shrink-0">{switchTs(event.ts)}</span>
      <span className={`text-xs font-bold ${fromMeta?.color ?? "text-gray-600"}`}>{fromMeta?.label ?? event.from}</span>
      <ArrowRight size={13} className="text-gray-400 flex-shrink-0" />
      <span className={`text-xs font-bold ${toMeta?.color ?? "text-gray-600"}`}>{toMeta?.label ?? event.to}</span>
      <span className="ml-auto text-[10px] bg-amber-100 text-amber-700 border border-amber-200 px-2 py-0.5 rounded-full font-medium">
        {reasonLabel[event.reason] ?? event.reason}
      </span>
    </div>
  );
}

// ─── Studio tool health row ───────────────────────────────────────────────────
function ToolHealthRow({ tool, index }: { tool: StudioToolHealth; index: number }) {
  const live     = isLive(tool.last_used_at);
  const errCount = tool.error_count_today ?? 0;
  const errRate  = tool.requests_today > 0
    ? ((errCount / tool.requests_today) * 100).toFixed(1)
    : "0.0";
  const errRateNum = parseFloat(errRate);

  return (
    <div className={`flex items-center gap-3 py-2.5 px-4 ${index % 2 === 0 ? "bg-gray-50/60" : ""}`}>
      {/* Live indicator */}
      <div className="w-5 flex-shrink-0 flex items-center justify-center">
        {live ? (
          <span className="relative flex h-2 w-2" title="Used in last 5 minutes">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
            <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500" />
          </span>
        ) : (
          <span className="h-2 w-2 rounded-full bg-gray-200 flex-shrink-0" />
        )}
      </div>

      {/* Slug */}
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium text-gray-800 truncate font-mono">{tool.slug}</p>
        <p className="text-[10px] text-gray-400 mt-0.5">
          Provider: <span className="font-medium text-gray-500">{tool.last_provider || "—"}</span>
        </p>
      </div>

      {/* Requests today */}
      <div className="text-right flex-shrink-0 w-16">
        <p className="text-sm font-bold text-gray-700">{tool.requests_today.toLocaleString()}</p>
        <p className="text-[10px] text-gray-400">req today</p>
      </div>

      {/* Error rate */}
      <div className="text-right flex-shrink-0 w-16">
        <p className={`text-sm font-bold ${errRateNum > 10 ? "text-red-500" : errRateNum > 3 ? "text-amber-500" : "text-green-600"}`}>
          {errRate}%
        </p>
        <p className="text-[10px] text-gray-400">err rate</p>
      </div>

      {/* Last error */}
      <div className="text-right flex-shrink-0 w-20">
        <p className="text-xs text-gray-500">{timeAgo(tool.last_error_at)}</p>
        <p className="text-[10px] text-gray-400">last err</p>
      </div>
    </div>
  );
}

// ─── Recent error feed row ────────────────────────────────────────────────────
function ErrorFeedRow({ gen, index }: { gen: Generation; index: number }) {
  return (
    <div className={`px-4 py-3 ${index % 2 === 0 ? "bg-red-50/30" : ""} hover:bg-red-50/50 transition-colors`}>
      <div className="flex items-start justify-between gap-2 mb-1">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-[10px] bg-red-100 text-red-700 border border-red-200 px-2 py-0.5 rounded font-bold font-mono whitespace-nowrap">
            {gen.tool_slug}
          </span>
          <span className="text-[10px] bg-gray-100 text-gray-600 border border-gray-200 px-2 py-0.5 rounded font-medium whitespace-nowrap">
            {gen.provider || "unknown"}
          </span>
        </div>
        <span className="text-[10px] text-gray-400 flex-shrink-0">{timeAgo(gen.created_at)}</span>
      </div>
      {/* Error message */}
      <p className="text-xs text-red-700 font-mono leading-snug mb-1 line-clamp-2">
        {gen.prompt
          ? `"${gen.prompt.slice(0, 60)}${gen.prompt.length > 60 ? "…" : ""}"`
          : <em className="not-italic text-gray-400">No prompt</em>
        }
      </p>
      <p className="text-xs text-gray-500 truncate">
        ⚠ {gen.status === "failed" ? "Generation failed" : gen.status}
      </p>
    </div>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────
export default function AIHealthPage() {
  const [report, setReport]           = useState<AIHealthReport | null>(null);
  const [recentErrors, setRecentErrors] = useState<Generation[]>([]);
  const [loading, setLoading]         = useState(true);
  const [errorsLoading, setErrorsLoading] = useState(false);
  const [error, setError]             = useState<string | null>(null);
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(true);

  const loadErrors = useCallback(async () => {
    setErrorsLoading(true);
    try {
      const r = await adminAPI.getStudioGenerations({ status: "failed", limit: 10 });
      setRecentErrors(r.generations ?? []);
    } catch {
      // non-fatal — errors feed is bonus info
    } finally {
      setErrorsLoading(false);
    }
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await adminAPI.getAIHealth();
      setReport(data);
      setLastRefresh(new Date());
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to load AI health");
    } finally {
      setLoading(false);
    }
    // load recent errors in parallel
    loadErrors();
  }, [loadErrors]);

  useEffect(() => { load(); }, [load]);

  // Auto-refresh every 30s
  useEffect(() => {
    if (!autoRefresh) return;
    const iv = setInterval(load, 30_000);
    return () => clearInterval(iv);
  }, [autoRefresh, load]);

  const hasAlert = report?.providers.some(
    (p) => p.status === "error" || p.status === "limit_reached"
  );

  const activeProvider = report?.providers.find(
    (p) => p.name === report.active_chat_provider
  );
  const activeMeta = activeProvider ? PROVIDER_META[activeProvider.name] : null;

  const sortedTools = [...(report?.studio_tools ?? [])].sort((a, b) => b.requests_today - a.requests_today);

  return (
    <AdminShell>
      <div className="max-w-5xl mx-auto px-4 py-8 space-y-8">

        {/* ── Page header ── */}
        <div className="flex items-start justify-between gap-4 flex-wrap">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-gradient-to-br from-indigo-500 to-purple-600 rounded-2xl flex items-center justify-center shadow">
              <Brain size={20} className="text-white" />
            </div>
            <div>
              <h1 className="text-2xl font-bold text-gray-900">AI Provider Health</h1>
              <p className="text-gray-500 text-sm mt-0.5">
                Live monitoring of LLM providers and Studio tool usage
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2.5">
            {/* Auto-refresh toggle */}
            <button
              onClick={() => setAutoRefresh((v) => !v)}
              className={`text-xs px-3 py-2 rounded-lg border font-medium transition-colors flex items-center gap-1.5 ${
                autoRefresh
                  ? "bg-green-50 text-green-700 border-green-200 hover:bg-green-100"
                  : "bg-gray-50 text-gray-500 border-gray-200 hover:bg-gray-100"
              }`}
            >
              {autoRefresh ? (
                <>
                  <span className="relative flex h-1.5 w-1.5">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
                    <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-green-500" />
                  </span>
                  Auto-refresh on (30s)
                </>
              ) : (
                <><Radio size={12} /> Auto-refresh off</>
              )}
            </button>
            <button
              onClick={load}
              disabled={loading}
              className="flex items-center gap-1.5 px-4 py-2 bg-indigo-600 text-white rounded-xl text-sm font-medium hover:bg-indigo-700 transition-colors disabled:opacity-50"
            >
              <RefreshCw size={14} className={loading ? "animate-spin" : ""} />
              Refresh
            </button>
          </div>
        </div>

        {/* ── Alert banner ── */}
        {hasAlert && (
          <div className="flex items-center gap-3 bg-amber-50 border border-amber-300 rounded-2xl p-4">
            <AlertTriangle size={20} className="text-amber-600 flex-shrink-0" />
            <div>
              <p className="text-amber-800 font-semibold text-sm">Provider issue detected</p>
              <p className="text-amber-700 text-xs mt-0.5">
                One or more AI providers has hit a rate limit or reported an error.
                Chat is automatically routing to the next available provider.
              </p>
            </div>
          </div>
        )}

        {/* ── Connection error ── */}
        {error && (
          <div className="flex items-center gap-3 bg-red-50 border border-red-200 rounded-2xl p-4">
            <WifiOff size={18} className="text-red-500 flex-shrink-0" />
            <p className="text-red-700 text-sm">{error}</p>
          </div>
        )}

        {/* ── Last refresh ── */}
        {lastRefresh && (
          <p className="text-gray-400 text-xs text-right -mt-4">
            Last refreshed: {lastRefresh.toLocaleTimeString()} ·{" "}
            {autoRefresh ? "auto-refreshes every 30s" : "auto-refresh paused"}
          </p>
        )}

        {/* ── Summary stats ── */}
        {report && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            {/* Active provider */}
            <div className="col-span-2 bg-gradient-to-br from-indigo-600 to-purple-700 rounded-2xl p-5 text-white">
              <div className="flex items-center gap-2 mb-3">
                <Activity size={16} className="opacity-80" />
                <span className="text-xs font-medium uppercase tracking-wider opacity-80">Active Chat Provider</span>
              </div>
              <div className="flex items-center gap-3">
                <ActivePulse />
                <div>
                  <p className="text-2xl font-bold">{activeMeta?.label ?? report.active_chat_provider}</p>
                  <p className="text-xs opacity-70 mt-0.5">{activeMeta?.model ?? "—"}</p>
                </div>
              </div>
            </div>

            {/* Total requests */}
            <div className="bg-white border border-gray-200 rounded-2xl p-4">
              <div className="flex items-center gap-2 mb-2">
                <Zap size={14} className="text-indigo-500" />
                <span className="text-gray-500 text-xs font-medium uppercase tracking-wider">Chat req today</span>
              </div>
              <p className="text-3xl font-bold text-gray-800">
                {report.providers.reduce((s, p) => s + p.requests_today, 0).toLocaleString()}
              </p>
            </div>

            {/* Provider switches */}
            <div className="bg-white border border-gray-200 rounded-2xl p-4">
              <div className="flex items-center gap-2 mb-2">
                <ArrowRight size={14} className="text-amber-500" />
                <span className="text-gray-500 text-xs font-medium uppercase tracking-wider">Switches today</span>
              </div>
              <p className="text-3xl font-bold text-gray-800">{report.recent_switches.length}</p>
              <p className="text-gray-400 text-xs mt-0.5">last 50 events shown</p>
            </div>
          </div>
        )}

        {/* ── Provider cards ── */}
        <section className="space-y-3">
          <div className="flex items-center gap-2">
            <Brain size={16} className="text-indigo-600" />
            <h2 className="text-base font-bold text-gray-800">LLM Provider Status</h2>
            <span className="text-xs text-gray-400 ml-auto">Cascade: Groq → Gemini → DeepSeek</span>
          </div>
          {loading && !report ? (
            <div className="grid md:grid-cols-3 gap-3">
              {[...Array(3)].map((_, i) => (
                <div key={i} className="h-44 rounded-2xl border-2 border-gray-100 bg-gray-50 animate-pulse" />
              ))}
            </div>
          ) : (
            <div className="grid md:grid-cols-3 gap-3">
              {(report?.providers ?? []).map((p) => (
                <ProviderCard
                  key={p.name}
                  provider={p}
                  isActive={p.name === report?.active_chat_provider}
                />
              ))}
            </div>
          )}
        </section>

        {/* ── Studio Tool Health table ── */}
        <section className="bg-white border border-gray-200 rounded-2xl overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b border-gray-100">
            <div className="flex items-center gap-2">
              <Wand2 size={15} className="text-purple-500" />
              <h3 className="font-semibold text-gray-800 text-sm">Studio Tool Health</h3>
              <span className="text-[10px] bg-purple-100 text-purple-700 border border-purple-200 px-2 py-0.5 rounded-full font-medium">
                {sortedTools.length} tools
              </span>
            </div>
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-1 text-[10px] text-gray-400">
                <span className="relative flex h-1.5 w-1.5"><span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-green-500" /></span>
                Live = used in last 5m
              </div>
              <div className="flex items-center gap-1 text-xs text-gray-400">
                <BarChart3 size={12} />
                {report?.studio_tools.reduce((s, t) => s + t.requests_today, 0) ?? 0} req today
              </div>
            </div>
          </div>

          {/* Column headers */}
          <div className="flex items-center gap-3 px-4 py-2 bg-gray-50 border-b border-gray-100 text-[10px] font-bold text-gray-400 uppercase tracking-wider">
            <div className="w-5 flex-shrink-0" />
            <div className="flex-1">Tool slug</div>
            <div className="w-16 text-right">Req today</div>
            <div className="w-16 text-right">Err rate</div>
            <div className="w-20 text-right">Last error</div>
          </div>

          <div className="divide-y divide-gray-50/80 max-h-80 overflow-y-auto">
            {sortedTools.length === 0 ? (
              <div className="py-10 text-center">
                <Wand2 size={24} className="mx-auto text-gray-300 mb-2" />
                <p className="text-gray-400 text-sm">No studio tool usage today</p>
              </div>
            ) : (
              sortedTools.map((tool, i) => <ToolHealthRow key={tool.slug} tool={tool} index={i} />)
            )}
          </div>
        </section>

        {/* ── Recent Generation Errors feed ── */}
        <section className="bg-white border border-gray-200 rounded-2xl overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b border-gray-100">
            <div className="flex items-center gap-2">
              <TrendingDown size={15} className="text-red-500" />
              <h3 className="font-semibold text-gray-800 text-sm">Recent Generation Failures</h3>
              {recentErrors.length > 0 && (
                <span className="text-[10px] bg-red-100 text-red-700 border border-red-200 px-2 py-0.5 rounded-full font-bold">
                  {recentErrors.length} shown
                </span>
              )}
            </div>
            <button
              onClick={loadErrors}
              disabled={errorsLoading}
              className="text-xs text-gray-400 hover:text-indigo-600 transition-colors flex items-center gap-1"
            >
              <RefreshCw size={11} className={errorsLoading ? "animate-spin" : ""} />
              Reload
            </button>
          </div>

          <div className="divide-y divide-gray-100 max-h-80 overflow-y-auto">
            {errorsLoading && (
              <div className="py-8 text-center">
                <RefreshCw size={18} className="mx-auto text-gray-300 mb-2 animate-spin" />
                <p className="text-gray-400 text-xs">Loading failures…</p>
              </div>
            )}
            {!errorsLoading && recentErrors.length === 0 && (
              <div className="py-10 text-center">
                <CheckCircle2 size={24} className="mx-auto text-green-400 mb-2" />
                <p className="text-gray-400 text-sm">No generation failures in the feed</p>
                <p className="text-gray-300 text-xs mt-1">All recent generations completed successfully ✓</p>
              </div>
            )}
            {!errorsLoading && recentErrors.map((gen, i) => (
              <ErrorFeedRow key={gen.id} gen={gen} index={i} />
            ))}
          </div>
        </section>

        {/* ── Two-column: Switch Log + Tools ── */}
        <div className="grid md:grid-cols-2 gap-4">

          {/* Provider switch log */}
          <section className="bg-white border border-gray-200 rounded-2xl overflow-hidden">
            <div className="flex items-center justify-between px-4 py-3 border-b border-gray-100">
              <div className="flex items-center gap-2">
                <ArrowRight size={15} className="text-amber-500" />
                <h3 className="font-semibold text-gray-800 text-sm">Provider Switches</h3>
              </div>
              <span className="text-gray-400 text-xs">Most recent first</span>
            </div>
            <div className="divide-y divide-gray-50 px-2 py-1 max-h-72 overflow-y-auto">
              {report?.recent_switches.length === 0 ? (
                <div className="py-8 text-center">
                  <CheckCircle2 size={24} className="mx-auto text-green-400 mb-2" />
                  <p className="text-gray-400 text-sm">No provider switches</p>
                  <p className="text-gray-300 text-xs">Groq has been handling everything ✓</p>
                </div>
              ) : (
                (report?.recent_switches ?? []).map((ev, i) => (
                  <SwitchRow key={i} event={ev} index={i} />
                ))
              )}
              {!report && !loading && (
                <p className="text-center text-gray-400 text-sm py-6">No data</p>
              )}
            </div>
          </section>

          {/* Studio tool request totals */}
          <section className="bg-white border border-gray-200 rounded-2xl overflow-hidden">
            <div className="flex items-center justify-between px-4 py-3 border-b border-gray-100">
              <div className="flex items-center gap-2">
                <BarChart3 size={15} className="text-indigo-500" />
                <h3 className="font-semibold text-gray-800 text-sm">Studio Req Volume (Today)</h3>
              </div>
              <div className="flex items-center gap-1 text-xs text-gray-400">
                <Activity size={12} />
                {report?.studio_tools.reduce((s, t) => s + t.requests_today, 0) ?? 0} total
              </div>
            </div>
            <div className="divide-y divide-gray-50 max-h-72 overflow-y-auto px-4 py-2 space-y-1">
              {sortedTools.length === 0 ? (
                <div className="py-8 text-center">
                  <Wand2 size={24} className="mx-auto text-gray-300 mb-2" />
                  <p className="text-gray-400 text-sm">No studio tool usage today</p>
                </div>
              ) : sortedTools.map((tool) => {
                const max = sortedTools[0]?.requests_today || 1;
                const pct = Math.max(2, (tool.requests_today / max) * 100);
                return (
                  <div key={tool.slug} className="flex items-center gap-3 py-1.5">
                    <span className="text-xs font-mono text-gray-700 w-32 truncate flex-shrink-0">{tool.slug}</span>
                    <div className="flex-1 bg-gray-100 rounded-full h-2 overflow-hidden">
                      <div
                        className="h-full rounded-full bg-gradient-to-r from-indigo-500 to-purple-500"
                        style={{ width: `${pct}%`, transition: "width 0.5s ease" }}
                      />
                    </div>
                    <span className="text-xs font-bold text-gray-700 w-10 text-right flex-shrink-0">
                      {tool.requests_today.toLocaleString()}
                    </span>
                  </div>
                );
              })}
            </div>
          </section>

        </div>

        {/* ── Explanation ── */}
        <div className="bg-gray-50 border border-gray-200 rounded-2xl p-5 space-y-3">
          <h3 className="font-semibold text-gray-700 text-sm flex items-center gap-2">
            <AlertCircle size={15} className="text-indigo-500" /> How the AI cascade works
          </h3>
          <div className="grid md:grid-cols-3 gap-3 text-xs text-gray-600">
            <div className="space-y-1">
              <p className="font-semibold text-green-700">1. Groq (Primary)</p>
              <p>~300 tokens/second. Handles all chat by default. Free at 6,000 req/min. When it hits its rate limit, the system switches automatically.</p>
            </div>
            <div className="space-y-1">
              <p className="font-semibold text-blue-700">2. Gemini Flash-Lite (Fallback)</p>
              <p>Handles overflow when Groq hits limits. Free at 1,500 req/day. When the daily quota is reached, traffic flows to DeepSeek.</p>
            </div>
            <div className="space-y-1">
              <p className="font-semibold text-purple-700">3. DeepSeek (Overflow)</p>
              <p>Pay-per-use last resort. Triggered only after both free providers are exhausted. Costs ~$0.10/month at typical user volumes.</p>
            </div>
          </div>
          <p className="text-xs text-gray-500 border-t border-gray-200 pt-3">
            <strong>When to act:</strong> If Groq shows &ldquo;limit_reached&rdquo; consistently every day, consider adding a second Groq API key.
            If DeepSeek is being used heavily, your user growth has exceeded the free tiers — upgrade Groq to a paid plan (~$10/month covers 10M tokens).
          </p>
        </div>

      </div>
    </AdminShell>
  );
}
