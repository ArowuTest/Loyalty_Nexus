/**
 * Keep-warm endpoint — called by Vercel Cron every 10 minutes.
 *
 * Pings the Render backend /health to wake the free-tier instance on demand.
 * Also called by Vercel Cron once per day (Hobby plan limit).
 * For higher-frequency pinging, call this endpoint from an external service
 * such as UptimeRobot (free, supports 5-minute intervals).
 *
 * Route: GET /api/keepwarm
 * Schedule: see vercel.json → crons (daily at 06:00 UTC)
 */
export const runtime = "edge";

const BACKEND_URL =
  process.env.NEXT_PUBLIC_API_URL?.replace("/api/v1", "") ??
  "https://loyalty-nexus-api.onrender.com";

export async function GET() {
  const start = Date.now();

  try {
    const res = await fetch(`${BACKEND_URL}/health`, {
      method: "GET",
      // Short timeout — we just want to wake the instance, not wait for a full response.
      signal: AbortSignal.timeout(10_000),
      cache: "no-store",
    });

    const elapsed = Date.now() - start;
    const body = res.ok ? await res.json().catch(() => ({})) : {};

    return Response.json(
      {
        ok: res.ok,
        status: res.status,
        backend: body,
        elapsed_ms: elapsed,
        pinged_at: new Date().toISOString(),
      },
      { status: 200 }
    );
  } catch (err) {
    const elapsed = Date.now() - start;
    return Response.json(
      {
        ok: false,
        error: err instanceof Error ? err.message : String(err),
        elapsed_ms: elapsed,
        pinged_at: new Date().toISOString(),
      },
      { status: 200 } // Always 200 so Vercel Cron doesn't alert on backend downtime
    );
  }
}
