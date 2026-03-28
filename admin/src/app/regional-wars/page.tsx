"use client";
import { useState, useEffect, useCallback } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, { RegionalStat, WarSecondaryDraw, WarSecondaryDrawWinner } from "@/lib/api";

// ─── Types ────────────────────────────────────────────────────────────────────

interface RegionalWar {
  id: string;
  period: string;
  status: "ACTIVE" | "COMPLETED";
  total_prize_kobo: number;
  starts_at: string;
  ends_at: string;
  resolved_at?: string;
}

interface WarWinner {
  id: string;
  war_id: string;
  state: string;
  rank: number;
  total_points: number;
  prize_kobo: number;
  status: string;
}

interface RegionalWarsData {
  leaderboard: RegionalStat[];
  history?: RegionalWar[];
  prize_pool_kobo?: number;
  winning_bonus_pp?: number;
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

const naira = (kobo: number) =>
  "₦" + (kobo / 100).toLocaleString("en-NG", { minimumFractionDigits: 0, maximumFractionDigits: 0 });

const RANK_MEDAL = ["🥇", "🥈", "🥉"];

const STATE_COLORS = ["#f9c74f", "#90e0ef", "#c77dff", "#f4a261", "#52b788", "#6d6875"];
function stateColor(i: number) { return STATE_COLORS[i % STATE_COLORS.length]; }

// ─── Sub-components ───────────────────────────────────────────────────────────

function StatusPill({ status }: { status: string }) {
  const cfg: Record<string, [string, string]> = {
    ACTIVE:           ["🟢 Active",     "rgba(16,185,129,0.15)"],
    COMPLETED:        ["✅ Completed",   "rgba(95,114,249,0.15)"],
    PENDING_PAYMENT:  ["⏳ Unpaid",      "rgba(249,199,79,0.15)"],
    PAID:             ["✅ Paid",        "rgba(16,185,129,0.15)"],
    FAILED:           ["❌ Failed",      "rgba(239,68,68,0.15)"],
  };
  const [label, bg] = cfg[status] ?? [status, "rgba(255,255,255,0.05)"];
  return (
    <span style={{ padding: "2px 10px", borderRadius: 20, background: bg, fontSize: 12, fontWeight: 600, color: "#e2e8ff", whiteSpace: "nowrap" }}>
      {label}
    </span>
  );
}

// ─── Secondary Draw Panel ─────────────────────────────────────────────────────

function SecondaryDrawPanel({
  war,
  winners,
  existingDraws,
  onDrawComplete,
}: {
  war: RegionalWar;
  winners: WarWinner[];
  existingDraws: WarSecondaryDraw[];
  onDrawComplete: () => void;
}) {
  const [selectedState, setSelectedState] = useState(winners[0]?.state ?? "");
  const [winnerCount, setWinnerCount] = useState(3);
  const [prizeKobo, setPrizeKobo] = useState(100000); // ₦1,000 default
  const [running, setRunning] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  // Pay modal
  const [payWinner, setPayWinner] = useState<WarSecondaryDrawWinner | null>(null);
  const [momoInput, setMomoInput] = useState("");
  const [paying, setPaying] = useState(false);

  const alreadyDrawn = existingDraws.find(d => d.state === selectedState && d.status === "COMPLETED");

  const handleRun = async () => {
    setErr(null);
    if (!selectedState) { setErr("Select a state"); return; }
    if (alreadyDrawn) { setErr(`Secondary draw for ${selectedState} already completed`); return; }
    if (!confirm(`Run secondary draw for ${selectedState}?\n${winnerCount} winner(s) × ${naira(prizeKobo)} = ${naira(winnerCount * prizeKobo)} total`)) return;
    setRunning(true);
    try {
      await adminAPI.runSecondaryDraw(war.id, {
        state: selectedState,
        winner_count: winnerCount,
        prize_per_winner_kobo: prizeKobo,
      });
      onDrawComplete();
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : "Draw failed");
    } finally {
      setRunning(false);
    }
  };

  const handlePay = async () => {
    if (!payWinner) return;
    if (momoInput.length < 10) { alert("Enter a valid MoMo number"); return; }
    setPaying(true);
    try {
      await adminAPI.markSecondaryWinnerPaid(payWinner.id, momoInput);
      setPayWinner(null);
      setMomoInput("");
      onDrawComplete();
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : "Payment failed");
    } finally {
      setPaying(false);
    }
  };

  return (
    <div style={{ marginTop: 32 }}>
      <h2 style={{ fontSize: 17, fontWeight: 700, color: "#e2e8ff", marginBottom: 16 }}>
        🎲 Secondary Draw
      </h2>
      <p style={{ color: "#828cb4", fontSize: 13, marginBottom: 20, lineHeight: 1.6 }}>
        Run a cryptographically-secure draw (CSPRNG Fisher-Yates) for users in a winning state.
        All active users in the state who earned points during the war window are eligible.
        Winners receive MoMo cash prizes paid manually by admin. One draw per state.
      </p>

      {/* Draw launcher */}
      <div className="card" style={{ padding: 20, marginBottom: 20 }}>
        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(160px, 1fr))", gap: 16, marginBottom: 16 }}>

          {/* State selector */}
          <div>
            <label style={{ fontSize: 11, color: "#828cb4", display: "block", marginBottom: 6 }}>Winning State</label>
            <select
              value={selectedState}
              onChange={e => setSelectedState(e.target.value)}
              style={{ width: "100%", background: "#1c2038", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13 }}
            >
              {winners.map(w => (
                <option key={w.state} value={w.state}>
                  {RANK_MEDAL[w.rank - 1] ?? "#" + w.rank} {w.state}
                </option>
              ))}
            </select>
          </div>

          {/* Winner count */}
          <div>
            <label style={{ fontSize: 11, color: "#828cb4", display: "block", marginBottom: 6 }}>Number of Winners (1–10)</label>
            <input
              type="number" min={1} max={10} value={winnerCount}
              onChange={e => setWinnerCount(Math.min(10, Math.max(1, Number(e.target.value))))}
              style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13 }}
            />
          </div>

          {/* Prize per winner */}
          <div>
            <label style={{ fontSize: 11, color: "#828cb4", display: "block", marginBottom: 6 }}>Prize / Winner (kobo)</label>
            <input
              type="number" min={0} step={10000} value={prizeKobo}
              onChange={e => setPrizeKobo(Number(e.target.value))}
              style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13 }}
            />
            <p style={{ fontSize: 11, color: "#5f72f9", marginTop: 4 }}>{naira(prizeKobo)} per winner • Total: {naira(winnerCount * prizeKobo)}</p>
          </div>
        </div>

        {/* State status */}
        {selectedState && alreadyDrawn && (
          <div style={{ marginBottom: 12, padding: "10px 14px", background: "rgba(95,114,249,0.1)", borderRadius: 8, color: "#828cb4", fontSize: 12 }}>
            ✅ Secondary draw for <strong style={{ color: "#e2e8ff" }}>{selectedState}</strong> was already run on {new Date(alreadyDrawn.executed_at!).toLocaleString()}
            — {alreadyDrawn.winner_count} winner(s) selected.
          </div>
        )}

        {err && (
          <div style={{ marginBottom: 12, padding: "10px 14px", background: "rgba(239,68,68,0.1)", borderRadius: 8, color: "#fca5a5", fontSize: 12 }}>
            ⚠️ {err}
          </div>
        )}

        <button
          onClick={handleRun}
          disabled={running || !!alreadyDrawn || war.status !== "COMPLETED"}
          style={{
            padding: "10px 24px", borderRadius: 8, fontWeight: 700, fontSize: 13, border: "none", cursor: "pointer",
            background: (running || !!alreadyDrawn || war.status !== "COMPLETED") ? "rgba(95,114,249,0.3)" : "#5f72f9",
            color: "#fff", opacity: (running || !!alreadyDrawn || war.status !== "COMPLETED") ? 0.5 : 1,
          }}
        >
          {running ? "Running draw…" : "🎲 Run Secondary Draw"}
        </button>
        {war.status !== "COMPLETED" && (
          <p style={{ fontSize: 11, color: "#828cb4", marginTop: 8 }}>War must be resolved before running secondary draw.</p>
        )}
      </div>

      {/* Results tables per draw */}
      {existingDraws.length > 0 && (
        <div className="space-y-4">
          {existingDraws.map(draw => (
            <div key={draw.id} className="card" style={{ overflow: "hidden" }}>
              <div style={{ padding: "14px 20px", borderBottom: "1px solid rgba(95,114,249,0.1)", display: "flex", alignItems: "center", gap: 12, flexWrap: "wrap" }}>
                <span style={{ fontWeight: 700, color: "#e2e8ff", fontSize: 15 }}>🏙 {draw.state}</span>
                <StatusPill status={draw.status} />
                <span style={{ color: "#828cb4", fontSize: 12 }}>
                  {draw.winner_count} winner(s) · {naira(draw.prize_per_winner_kobo)} each · {draw.participant_count} eligible users
                </span>
                {draw.executed_at && (
                  <span style={{ color: "#828cb4", fontSize: 12, marginLeft: "auto" }}>
                    Drawn {new Date(draw.executed_at).toLocaleString()}
                  </span>
                )}
              </div>
              <table style={{ width: "100%", borderCollapse: "collapse" }}>
                <thead>
                  <tr style={{ borderBottom: "1px solid rgba(95,114,249,0.08)" }}>
                    {["#", "Phone", "Prize", "Payment", "Action"].map(h => (
                      <th key={h} style={{ padding: "10px 16px", textAlign: "left", color: "#828cb4", fontSize: 12 }}>{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {(draw.winners ?? []).map((winner) => (
                    <tr key={winner.id} style={{ borderBottom: "1px solid rgba(95,114,249,0.04)" }}>
                      <td style={{ padding: "10px 16px", color: "#e2e8ff", fontWeight: 700, fontSize: 16 }}>
                        {RANK_MEDAL[winner.position - 1] ?? `#${winner.position}`}
                      </td>
                      <td style={{ padding: "10px 16px", color: "#e2e8ff", fontFamily: "monospace", fontSize: 13 }}>
                        {winner.phone_number}
                      </td>
                      <td style={{ padding: "10px 16px", color: "#f9c74f", fontWeight: 700, fontSize: 13 }}>
                        {naira(winner.prize_kobo)}
                      </td>
                      <td style={{ padding: "10px 16px" }}>
                        <StatusPill status={winner.payment_status} />
                        {winner.momo_number && (
                          <span style={{ display: "block", color: "#828cb4", fontSize: 11, marginTop: 2 }}>
                            MoMo: {winner.momo_number}
                          </span>
                        )}
                      </td>
                      <td style={{ padding: "10px 16px" }}>
                        {winner.payment_status === "PENDING_PAYMENT" && (
                          <button
                            onClick={() => { setPayWinner(winner); setMomoInput(winner.momo_number ?? ""); }}
                            style={{
                              padding: "6px 14px", borderRadius: 7, background: "rgba(16,185,129,0.2)",
                              border: "1px solid rgba(16,185,129,0.3)", color: "#34d399", fontSize: 12,
                              fontWeight: 600, cursor: "pointer",
                            }}
                          >
                            💵 Pay
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ))}
        </div>
      )}

      {/* Pay Modal */}
      {payWinner && (
        <div
          style={{ position: "fixed", inset: 0, background: "rgba(0,0,0,0.7)", zIndex: 200, display: "flex", alignItems: "center", justifyContent: "center", padding: 20 }}
          onClick={e => e.target === e.currentTarget && setPayWinner(null)}
        >
          <div className="card" style={{ width: "100%", maxWidth: 420, padding: 24 }}>
            <h3 style={{ fontSize: 17, fontWeight: 700, color: "#e2e8ff", marginBottom: 4 }}>💵 Mark Winner Paid</h3>
            <p style={{ color: "#828cb4", fontSize: 13, marginBottom: 20 }}>
              {payWinner.phone_number} — {naira(payWinner.prize_kobo)} MoMo Cash
            </p>
            <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 6 }}>MoMo Number Used for Payment</label>
            <input
              value={momoInput}
              onChange={e => setMomoInput(e.target.value)}
              placeholder="0801 234 5678"
              style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "10px 14px", color: "#e2e8ff", fontSize: 14, marginBottom: 16, boxSizing: "border-box" }}
            />
            <div style={{ display: "flex", gap: 10 }}>
              <button
                onClick={handlePay}
                disabled={paying}
                style={{ flex: 1, padding: "10px 0", borderRadius: 8, background: "#10b981", border: "none", color: "#fff", fontWeight: 700, fontSize: 13, cursor: "pointer", opacity: paying ? 0.6 : 1 }}
              >
                {paying ? "Saving…" : "Confirm Payment"}
              </button>
              <button
                onClick={() => setPayWinner(null)}
                style={{ padding: "10px 16px", borderRadius: 8, background: "rgba(255,255,255,0.05)", border: "1px solid rgba(255,255,255,0.1)", color: "#828cb4", fontSize: 13, cursor: "pointer" }}
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function RegionalWarsPage() {
  const [data, setData]             = useState<RegionalWarsData | null>(null);
  const [loading, setLoading]       = useState(true);
  const [activeWar, setActiveWar]   = useState<RegionalWar | null>(null);
  const [warWinners, setWarWinners] = useState<WarWinner[]>([]);
  const [secondaryDraws, setSecondaryDraws] = useState<WarSecondaryDraw[]>([]);
  const [selectedWarId, setSelectedWarId]   = useState<string | null>(null);

  // Prize pool editor
  const [editPrize, setEditPrize]   = useState(false);
  const [prizeInput, setPrizeInput] = useState("");
  const [savingPrize, setSavingPrize] = useState(false);

  // Resolve
  const [resolving, setResolving]   = useState(false);
  const [resolveErr, setResolveErr] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const r = await adminAPI.getRegionalWars() as RegionalWarsData;
      setData(r);
      const history: RegionalWar[] = (r as { history?: RegionalWar[] }).history ?? [];
      // Active war is the most recent ACTIVE one
      const active = history.find((w) => w.status === "ACTIVE") ?? null;
      setActiveWar(active);
      setPrizeInput(String((r.prize_pool_kobo ?? 50_000_000) / 100));
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  // Load secondary draws + war winners when a completed war is selected
  const loadWarDetail = useCallback(async (warId: string) => {
    setSelectedWarId(warId);
    try {
      const [winnersR, drawsR] = await Promise.all([
        adminAPI.req<{ winners: WarWinner[] }>("GET", `/admin/wars/${warId}/winners`) as Promise<{ winners: WarWinner[] }>,
        adminAPI.getSecondaryDraws(warId),
      ]);
      setWarWinners(winnersR.winners ?? []);
      setSecondaryDraws(drawsR.draws ?? []);
    } catch (e) {
      console.error("loadWarDetail", e);
    }
  }, []);

  const savePrizePool = async () => {
    if (!activeWar) return;
    setSavingPrize(true);
    try {
      await adminAPI.req("PUT", "/admin/wars/prize-pool", {
        period: activeWar.period,
        total_prize_kobo: Number(prizeInput) * 100,
      });
      setEditPrize(false);
      await load();
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSavingPrize(false);
    }
  };

  const resolveWar = async () => {
    if (!activeWar) return;
    if (!confirm(`Resolve war for period ${activeWar.period}?\nThis is irreversible — top-3 states will be locked in as winners.`)) return;
    setResolving(true);
    setResolveErr(null);
    try {
      await adminAPI.resolveWar(activeWar.period);
      await load();
      // Auto-select the now-completed war for secondary draw
      await loadWarDetail(activeWar.id);
    } catch (e: unknown) {
      setResolveErr(e instanceof Error ? e.message : "Resolve failed");
    } finally {
      setResolving(false);
    }
  };

  const history: RegionalWar[] = (data as unknown as { history?: RegionalWar[] })?.history ?? [];

  return (
    <AdminShell>
      <div style={{ maxWidth: 960, margin: "0 auto", paddingBottom: 60 }}>

        {/* ── Header ── */}
        <div style={{ marginBottom: 24 }}>
          <h1 style={{ fontSize: 22, fontWeight: 800, color: "#e2e8ff" }}>🌍 Regional Wars</h1>
          <p style={{ color: "#828cb4", fontSize: 13, marginTop: 4 }}>
            Monthly state competition. Top-3 states split the prize pool. Admin resolves the war
            and triggers secondary draws for winning state users.
          </p>
        </div>

        {loading ? (
          <div style={{ display: "flex", justifyContent: "center", padding: "80px 0" }}>
            <div style={{ width: 36, height: 36, border: "3px solid #5f72f9", borderTopColor: "transparent", borderRadius: "50%", animation: "spin 0.8s linear infinite" }} />
          </div>
        ) : (
          <>
            {/* ── Active War Card ── */}
            {activeWar && (
              <div className="card" style={{ padding: 20, marginBottom: 24 }}>
                <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", flexWrap: "wrap", gap: 12 }}>
                  <div>
                    <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
                      <h2 style={{ fontSize: 18, fontWeight: 700, color: "#e2e8ff" }}>
                        Active War — {activeWar.period}
                      </h2>
                      <StatusPill status="ACTIVE" />
                    </div>
                    <p style={{ color: "#828cb4", fontSize: 12, marginTop: 4 }}>
                      {new Date(activeWar.starts_at).toLocaleDateString()} →{" "}
                      {new Date(activeWar.ends_at).toLocaleDateString()}
                    </p>
                  </div>

                  {/* Prize pool */}
                  <div style={{ textAlign: "right" }}>
                    {editPrize ? (
                      <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
                        <span style={{ color: "#828cb4", fontSize: 13 }}>₦</span>
                        <input
                          type="number" value={prizeInput}
                          onChange={e => setPrizeInput(e.target.value)}
                          style={{ width: 120, background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.3)", borderRadius: 7, padding: "6px 10px", color: "#e2e8ff", fontSize: 14 }}
                        />
                        <button onClick={savePrizePool} disabled={savingPrize}
                          style={{ padding: "6px 14px", borderRadius: 7, background: "#10b981", border: "none", color: "#fff", fontWeight: 600, fontSize: 12, cursor: "pointer" }}>
                          {savingPrize ? "…" : "Save"}
                        </button>
                        <button onClick={() => setEditPrize(false)}
                          style={{ padding: "6px 10px", borderRadius: 7, background: "transparent", border: "1px solid rgba(255,255,255,0.1)", color: "#828cb4", fontSize: 12, cursor: "pointer" }}>
                          ✕
                        </button>
                      </div>
                    ) : (
                      <div>
                        <p style={{ color: "#f9c74f", fontSize: 20, fontWeight: 800 }}>
                          {naira(activeWar.total_prize_kobo)}
                        </p>
                        <p style={{ color: "#828cb4", fontSize: 11 }}>Prize pool</p>
                        <button onClick={() => setEditPrize(true)}
                          style={{ marginTop: 4, padding: "4px 12px", borderRadius: 6, background: "transparent", border: "1px solid rgba(95,114,249,0.3)", color: "#5f72f9", fontSize: 11, cursor: "pointer" }}>
                          Edit
                        </button>
                      </div>
                    )}
                  </div>
                </div>

                {/* Resolve button */}
                <div style={{ marginTop: 16, paddingTop: 16, borderTop: "1px solid rgba(95,114,249,0.1)" }}>
                  {resolveErr && (
                    <p style={{ color: "#fca5a5", fontSize: 12, marginBottom: 10 }}>⚠️ {resolveErr}</p>
                  )}
                  <button onClick={resolveWar} disabled={resolving}
                    style={{
                      padding: "10px 24px", borderRadius: 8, background: resolving ? "rgba(249,199,79,0.2)" : "#f9c74f",
                      border: "none", color: "#000", fontWeight: 700, fontSize: 13, cursor: "pointer",
                      opacity: resolving ? 0.7 : 1,
                    }}>
                    {resolving ? "Resolving…" : "🏁 Resolve War & Lock Winners"}
                  </button>
                  <p style={{ fontSize: 11, color: "#828cb4", marginTop: 8 }}>
                    Locks top-3 states, awards Pulse Point bonuses to winning-state members, marks war COMPLETED.
                    Run after the month ends.
                  </p>
                </div>
              </div>
            )}

            {/* ── Leaderboard ── */}
            <div className="card" style={{ marginBottom: 24, overflow: "hidden" }}>
              <div style={{ padding: "14px 20px", borderBottom: "1px solid rgba(95,114,249,0.1)" }}>
                <h2 style={{ fontSize: 15, fontWeight: 700, color: "#e2e8ff" }}>📊 Current Leaderboard</h2>
              </div>
              <table style={{ width: "100%", borderCollapse: "collapse" }}>
                <thead>
                  <tr style={{ borderBottom: "1px solid rgba(95,114,249,0.08)" }}>
                    {["Rank", "State", "Total Points", "Active Members", "Prize Share"].map(h => (
                      <th key={h} style={{ padding: "10px 16px", textAlign: "left", color: "#828cb4", fontSize: 12 }}>{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {(data?.leaderboard ?? []).map((row, i) => (
                    <tr key={row.state} style={{ borderBottom: "1px solid rgba(95,114,249,0.04)" }}>
                      <td style={{ padding: "10px 16px", fontSize: 20 }}>
                        {RANK_MEDAL[i] ?? <span style={{ color: "#828cb4", fontSize: 14 }}>#{row.rank ?? i + 1}</span>}
                      </td>
                      <td style={{ padding: "10px 16px" }}>
                        <span style={{ background: `${stateColor(i)}22`, color: stateColor(i), padding: "3px 10px", borderRadius: 12, fontWeight: 700, fontSize: 13 }}>
                          {row.state}
                        </span>
                      </td>
                      <td style={{ padding: "10px 16px", color: "#f9c74f", fontWeight: 700 }}>
                        {row.total_points.toLocaleString()} pts
                      </td>
                      <td style={{ padding: "10px 16px", color: "#828cb4", fontSize: 13 }}>
                        {row.active_members.toLocaleString()}
                      </td>
                      <td style={{ padding: "10px 16px", color: i < 3 ? "#10b981" : "#828cb4", fontSize: 13 }}>
                        {i === 0 ? "50%" : i === 1 ? "30%" : i === 2 ? "20%" : "—"}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* ── War History ── */}
            <div className="card" style={{ marginBottom: 24, overflow: "hidden" }}>
              <div style={{ padding: "14px 20px", borderBottom: "1px solid rgba(95,114,249,0.1)" }}>
                <h2 style={{ fontSize: 15, fontWeight: 700, color: "#e2e8ff" }}>🏆 War History</h2>
              </div>
              {history.length === 0 ? (
                <p style={{ padding: "30px 20px", color: "#828cb4", fontSize: 13 }}>No completed wars yet.</p>
              ) : (
                <table style={{ width: "100%", borderCollapse: "collapse" }}>
                  <thead>
                    <tr style={{ borderBottom: "1px solid rgba(95,114,249,0.08)" }}>
                      {["Period", "Prize Pool", "Status", "Resolved", ""].map(h => (
                        <th key={h} style={{ padding: "10px 16px", textAlign: "left", color: "#828cb4", fontSize: 12 }}>{h}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {history.map((war) => (
                      <tr key={war.id}
                        onClick={() => loadWarDetail(war.id)}
                        style={{
                          borderBottom: "1px solid rgba(95,114,249,0.04)",
                          cursor: "pointer",
                          background: selectedWarId === war.id ? "rgba(95,114,249,0.07)" : "transparent",
                        }}>
                        <td style={{ padding: "10px 16px", color: "#e2e8ff", fontWeight: 600 }}>{war.period}</td>
                        <td style={{ padding: "10px 16px", color: "#f9c74f", fontWeight: 700 }}>{naira(war.total_prize_kobo)}</td>
                        <td style={{ padding: "10px 16px" }}><StatusPill status={war.status} /></td>
                        <td style={{ padding: "10px 16px", color: "#828cb4", fontSize: 12 }}>
                          {war.resolved_at ? new Date(war.resolved_at).toLocaleDateString() : "—"}
                        </td>
                        <td style={{ padding: "10px 16px", color: "#5f72f9", fontSize: 12 }}>
                          {war.status === "COMPLETED" ? "View →" : ""}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>

            {/* ── Selected War Detail — Secondary Draw Panel ── */}
            {selectedWarId && warWinners.length > 0 && (
              <>
                {/* War winners summary */}
                <div className="card" style={{ padding: 20, marginBottom: 4 }}>
                  <h2 style={{ fontSize: 15, fontWeight: 700, color: "#e2e8ff", marginBottom: 14 }}>
                    🥇 State Winners — {history.find(w => w.id === selectedWarId)?.period}
                  </h2>
                  <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
                    {warWinners.map((w) => (
                      <div key={w.id} style={{ flex: "1 1 160px", background: "rgba(255,255,255,0.03)", borderRadius: 12, padding: "14px 16px", border: "1px solid rgba(95,114,249,0.15)" }}>
                        <div style={{ fontSize: 28, marginBottom: 4 }}>{RANK_MEDAL[w.rank - 1]}</div>
                        <p style={{ fontWeight: 700, color: "#e2e8ff", fontSize: 16 }}>{w.state}</p>
                        <p style={{ color: "#f9c74f", fontWeight: 700 }}>{naira(w.prize_kobo)}</p>
                        <p style={{ color: "#828cb4", fontSize: 12 }}>{w.total_points.toLocaleString()} pts</p>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Secondary draw panel */}
                <SecondaryDrawPanel
                  war={history.find(w => w.id === selectedWarId)!}
                  winners={warWinners}
                  existingDraws={secondaryDraws}
                  onDrawComplete={() => loadWarDetail(selectedWarId)}
                />
              </>
            )}
          </>
        )}
      </div>

      <style>{`
        @keyframes spin { to { transform: rotate(360deg); } }
      `}</style>
    </AdminShell>
  );
}
