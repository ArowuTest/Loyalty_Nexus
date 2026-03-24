// lib/google-wallet.ts
import jwt from 'jsonwebtoken';
import { GoogleAuth } from 'google-auth-library';

const BASE = 'https://walletobjects.googleapis.com/walletobjects/v1';

/**
 * Lazily resolve Google Wallet env vars at call time (not module load time).
 * Throws a clear error if any required variable is missing — prevents
 * a raw TypeError crash that would take down unrelated routes.
 */
function getWalletConfig() {
  const ISSUER_ID    = process.env.GOOGLE_WALLET_ISSUER_ID;
  const CLIENT_EMAIL = process.env.GOOGLE_WALLET_CLIENT_EMAIL;
  const RAW_KEY      = process.env.GOOGLE_WALLET_PRIVATE_KEY;

  if (!ISSUER_ID || !CLIENT_EMAIL || !RAW_KEY) {
    const missing = [
      !ISSUER_ID    && 'GOOGLE_WALLET_ISSUER_ID',
      !CLIENT_EMAIL && 'GOOGLE_WALLET_CLIENT_EMAIL',
      !RAW_KEY      && 'GOOGLE_WALLET_PRIVATE_KEY',
    ].filter(Boolean).join(', ');
    throw new Error(`Google Wallet non configuré (variables manquantes : ${missing})`);
  }

  return {
    ISSUER_ID,
    CLIENT_EMAIL,
    PRIVATE_KEY: RAW_KEY.replace(/\\n/g, '\n'),
  };
}

/* ── Backward-compat interface (kept for existing callers) ────────────────── */

interface CreateCardParams {
  customerId:     string;
  firstName:      string;
  totalPoints:    number;
  restaurantName: string;
  /** Use restaurantId (UUID, immutable). restaurantSlug was removed — slugs are mutable. */
  restaurantId:   string;
  primaryColor:   string;
  logoUrl:        string | null;
}

/* ── Shared types ─────────────────────────────────────────────────────────── */

export interface GooglePassData {
  objectId:        string;
  classId:         string;
  customerId:      string;
  displayName:     string;
  totalPoints:     number;
  stampsCount:     number;
  stampsTotal:     number;
  rewardThreshold: number;
  rewardMessage:   string;
  qrToken:         string;
  /** 8-char uppercase code stored in wallet_passes.short_code; shown under QR for manual entry */
  shortCode?:      string;
  restaurantName:  string;
  primaryColor:    string;
  passKind:        'stamps' | 'points' | 'event';
}

/* ── Internal: lazy-initialised GoogleAuth instance ──────────────────────── */

let _auth: GoogleAuth | null = null;

function getAuth(): GoogleAuth {
  if (!_auth) {
    const { CLIENT_EMAIL, PRIVATE_KEY } = getWalletConfig();
    _auth = new GoogleAuth({
      credentials: {
        client_email: CLIENT_EMAIL,
        private_key:  PRIVATE_KEY,
      },
      scopes: ['https://www.googleapis.com/auth/wallet_object.issuer'],
    });
  }
  return _auth;
}

/* ── Internal: authenticated Google API fetch ────────────────────────────── */

async function gFetch(
  method: 'GET' | 'POST' | 'PUT' | 'PATCH',
  path: string,
  body?: unknown,
): Promise<{ ok: boolean; status: number; data: unknown; error?: string }> {
  try {
    const token = await getAuth().getAccessToken();
    const res   = await fetch(`${BASE}${path}`, {
      method,
      headers: {
        Authorization:  `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
    const data = await res.json().catch(() => null);
    return { ok: res.ok, status: res.status, data };
  } catch (err) {
    const error = err instanceof Error ? err.message : String(err);
    return { ok: false, status: 0, data: null, error };
  }
}

/* ── Class management ─────────────────────────────────────────────────────── */

export async function ensureLoyaltyClass(params: {
  classId:        string;
  restaurantName: string;
  primaryColor:   string;
  passKind:       'stamps' | 'points' | 'event';
  logoUrl?:       string | null;
}): Promise<{ ok: boolean; created: boolean }> {
  const { classId, restaurantName, primaryColor, logoUrl } = params;

  // GET first — skip creation if class already exists
  const check = await gFetch('GET', `/loyaltyClass/${encodeURIComponent(classId)}`);
  if (check.ok) return { ok: true, created: false };
  if (check.status !== 404) {
    // Unexpected error (401 = bad credentials, 403 = wrong issuer/scope, 5xx = Google outage).
    // Do NOT attempt creation — we can't know if the class already exists.
    console.error(
      `[GWallet] ensureLoyaltyClass GET failed classId=${classId}` +
      ` HTTP ${check.status}` +
      (check.error ? ` error=${check.error}` : ` body=${JSON.stringify(check.data).slice(0, 300)}`),
    );
    return { ok: false, created: false };
  }

  const hexColor  = primaryColor.startsWith('#') ? primaryColor : '#4f6bed';
  const classBody: Record<string, unknown> = {
    id:                 classId,
    issuerName:         restaurantName,
    programName:        'Carte fidélité',
    reviewStatus:       'UNDER_REVIEW',
    hexBackgroundColor: hexColor,
    countryCode:        'FR',
  };

  if (logoUrl) {
    classBody.programLogo = {
      sourceUri:          { uri: logoUrl },
      contentDescription: { defaultValue: { language: 'fr-FR', value: restaurantName } },
    };
  }

  const create = await gFetch('POST', '/loyaltyClass', classBody);
  return { ok: create.ok, created: create.ok };
}

export async function updateLoyaltyClass(params: {
  classId:        string;
  restaurantName: string;
  primaryColor:   string;
  logoUrl?:       string | null;
}): Promise<{ ok: boolean }> {
  const { classId, restaurantName, primaryColor, logoUrl } = params;
  const hexColor = primaryColor.startsWith('#') ? primaryColor : '#4f6bed';

  const patch: Record<string, unknown> = {
    issuerName:         restaurantName,
    hexBackgroundColor: hexColor,
  };
  if (logoUrl) {
    patch.programLogo = {
      sourceUri:          { uri: logoUrl },
      contentDescription: { defaultValue: { language: 'fr-FR', value: restaurantName } },
    };
  }

  const result = await gFetch('PATCH', `/loyaltyClass/${encodeURIComponent(classId)}`, patch);
  return { ok: result.ok };
}

/* ── Object management ────────────────────────────────────────────────────── */

export async function createLoyaltyObject(data: GooglePassData): Promise<{ ok: boolean }> {
  const body   = buildLoyaltyObject(data);
  const result = await gFetch('POST', '/loyaltyObject', body);
  return { ok: result.ok };
}

export async function updateLoyaltyObject(
  objectId: string,
  patch: {
    totalPoints?:   number;
    stampsCount?:   number;
    stampsTotal?:   number;
    rewardMessage?: string;
    state?:         'ACTIVE' | 'EXPIRED' | 'INACTIVE';
    /** When provided, patches the barcode on the Google Wallet object.
     *  Use this during a Sync to align the QR value with the stored short_code. */
    barcode?:       { value: string; alternateText: string };
    /**
     * Controls which Google Wallet field carries the loyalty balance.
     * - 'stamps' → primary loyaltyPoints = stamps count ("X / Y Tampons").
     *              Matches what buildLoyaltyObject() sets at issuance for stamps passes.
     * - 'points' or undefined → primary loyaltyPoints = total points ("X pts").
     *
     * Omit secondaryLoyaltyPoints entirely — we never use it; adding it after
     * issuance would create an unexpected second row on the card face.
     */
    passKind?:      'stamps' | 'points' | 'event';
  },
): Promise<{ ok: boolean; status: number; data: unknown; error?: string }> {
  const body: Record<string, unknown> = {};

  if (patch.state !== undefined) {
    body.state = patch.state;
  }

  // Track which fields we're explicitly updating so we can build the updateMask.
  // Google Wallet PATCH is additive by default: absent fields stay unchanged.
  // With ?updateMask=..., fields listed in the mask but absent from the body are
  // CLEARED to their default value — this is how we remove secondaryLoyaltyPoints.
  const maskFields: string[] = [];

  if (patch.passKind === 'stamps') {
    // Stamps mode: primary loyaltyPoints shows stamp progress — same layout as issuance.
    if (patch.stampsCount !== undefined) {
      body.loyaltyPoints = {
        balance: { string: `${patch.stampsCount} / ${patch.stampsTotal ?? 10}` },
        label:   'Tampons',
      };
      maskFields.push('loyaltyPoints');
    }
    // Always include secondaryLoyaltyPoints in the mask WITHOUT adding it to the body.
    // This tells Google to clear it — removing any stale "X pts" counter from old syncs
    // that incorrectly used secondaryLoyaltyPoints before the passKind fix.
    maskFields.push('secondaryLoyaltyPoints');
  } else {
    // Points mode (default): primary loyaltyPoints shows total points.
    if (patch.totalPoints !== undefined) {
      body.loyaltyPoints = {
        balance: { string: `${patch.totalPoints} pts` },
        label:   'Points fidélité',
      };
      maskFields.push('loyaltyPoints');
    }
  }

  if (patch.state !== undefined) {
    maskFields.push('state');
  }
  if (patch.rewardMessage !== undefined) {
    body.textModulesData = [
      { header: 'Récompense', body: patch.rewardMessage, id: 'reward' },
    ];
    maskFields.push('textModulesData');
  }
  if (patch.barcode !== undefined) {
    body.barcode = {
      type:          'QR_CODE',
      value:         patch.barcode.value,
      alternateText: patch.barcode.alternateText,
    };
    maskFields.push('barcode');
  }

  const qs     = maskFields.length > 0 ? `?updateMask=${maskFields.join(',')}` : '';
  const result = await gFetch('PATCH', `/loyaltyObject/${encodeURIComponent(objectId)}${qs}`, body);

  if (result.ok) {
    // Extract the confirmed loyaltyPoints value from Google's response body.
    // A 200 with a mismatched balance here means our patch payload was silently ignored
    // (e.g. class still UNDER_REVIEW, invalid field combination, or stale cached object).
    const responseObj  = result.data as Record<string, unknown> | null;
    const lp           = responseObj?.loyaltyPoints as Record<string, unknown> | null;
    const confirmedBal = (lp?.balance as Record<string, unknown> | null)?.string ?? '(no balance field)';
    console.log(
      `[GWallet] PATCH ok objectId=${objectId}` +
      ` mask=[${maskFields.join(',')}]` +
      ` sent=${JSON.stringify({ stampsCount: patch.stampsCount, stampsTotal: patch.stampsTotal, totalPoints: patch.totalPoints, passKind: patch.passKind })}` +
      ` confirmed loyaltyPoints.balance.string="${confirmedBal}"`,
    );
  } else {
    console.error(
      `[GWallet] PATCH failed objectId=${objectId}` +
      ` mask=[${maskFields.join(',')}]` +
      ` HTTP ${result.status}` +
      (result.error ? ` error=${result.error}` : ` body=${JSON.stringify(result.data).slice(0, 300)}`),
    );
  }

  return { ok: result.ok, status: result.status, data: result.data, error: result.error };
}

export async function revokeLoyaltyObject(objectId: string): Promise<{ ok: boolean }> {
  const result = await updateLoyaltyObject(objectId, { state: 'EXPIRED' });
  return { ok: result.ok };
}

/**
 * Recovers a broken Google Wallet loyalty object.
 *
 * Recovery strategy (in order):
 *   1. GET the existing object in Google.
 *      - 200 + state ACTIVE  → 'already_active' (idempotent, no action needed)
 *      - 200 + other state   → PATCH state to ACTIVE → 'patched'
 *      - 404                 → CREATE the object from scratch → 'created'
 *      - any other error     → 'failed'
 *
 * IMPORTANT: caller must run ensureLoyaltyClass() successfully before calling this.
 * The referenced class must exist in Google's system before any recovery attempt.
 */
export async function recoverLoyaltyObject(
  data: GooglePassData,
): Promise<{ ok: boolean; strategy: 'already_active' | 'patched' | 'created' | 'failed'; error?: string }> {
  const { objectId } = data;

  const check = await gFetch('GET', `/loyaltyObject/${encodeURIComponent(objectId)}`);

  if (check.ok) {
    const existing = check.data as Record<string, unknown> | null;

    if (existing?.state === 'ACTIVE') {
      // Object is already active — class now exists so installation should succeed.
      return { ok: true, strategy: 'already_active' };
    }

    // Object exists but is in a non-active state (e.g. INACTIVE, EXPIRED) — restore it.
    const patch = await gFetch('PATCH', `/loyaltyObject/${encodeURIComponent(objectId)}`, { state: 'ACTIVE' });
    if (!patch.ok) {
      return {
        ok:       false,
        strategy: 'failed',
        error:    patch.error ?? `PATCH state=ACTIVE failed (HTTP ${patch.status})`,
      };
    }
    return { ok: true, strategy: 'patched' };
  }

  if (check.status === 404) {
    // Object was never created — class now guaranteed to exist, create it fresh.
    const create = await createLoyaltyObject(data);
    if (!create.ok) {
      return { ok: false, strategy: 'failed', error: 'createLoyaltyObject failed' };
    }
    return { ok: true, strategy: 'created' };
  }

  // Unexpected error from Google API
  return {
    ok:       false,
    strategy: 'failed',
    error:    check.error ?? `GET /loyaltyObject failed (HTTP ${check.status})`,
  };
}

/* ── Internal: JWT / pass body builders ──────────────────────────────────── */

function buildLoyaltyObject(data: GooglePassData): Record<string, unknown> {
  const hexColor = data.primaryColor.startsWith('#') ? data.primaryColor : '#4f6bed';

  // The short_code is the human-readable manual-entry code.
  // Fall back to first 8 hex chars of qrToken for legacy passes without a stored short_code.
  const manualCode = data.shortCode ?? data.qrToken.replace(/-/g, '').slice(0, 8).toUpperCase();

  const obj: Record<string, unknown> = {
    id:      data.objectId,
    classId: data.classId,
    state:   'ACTIVE',
    accountId:   data.customerId,
    accountName: data.displayName,
    hexBackgroundColor: hexColor,
    barcode: {
      type:          'QR_CODE',
      // When shortCode is set, the QR encodes the short_code (8 chars).
      // Both camera scan and manual entry resolve via the same step-3 short_code lookup.
      // Legacy passes (no shortCode) still encode qrToken so step-1 lookup remains valid.
      value:         data.shortCode ?? data.qrToken,
      alternateText: manualCode,
    },
    textModulesData: [
      { header: 'Programme', body: `Fidélité ${data.restaurantName}`, id: 'program' },
      { header: 'Récompense', body: data.rewardMessage || 'Récompense offerte !', id: 'reward' },
      { header: 'Code manuel', body: manualCode, id: 'manual_code' },
    ],
  };

  if (data.passKind === 'stamps') {
    obj.loyaltyPoints = {
      balance: { string: `${data.stampsCount} / ${data.stampsTotal}` },
      label:   'Tampons',
    };
  } else {
    obj.loyaltyPoints = {
      balance: { string: `${data.totalPoints} pts` },
      label:   'Points fidélité',
    };
  }

  return obj;
}

/**
 * Compute the deterministic Google Wallet classId for a restaurant + passKind.
 * Uses restaurantId (UUID, immutable) — never the slug (mutable).
 * One class per (restaurant × passKind): different program UX per type.
 */
export function computeClassId(restaurantId: string, passKind: string): string {
  const { ISSUER_ID } = getWalletConfig();
  return `${ISSUER_ID}.r${restaurantId.replace(/-/g, '')}_${passKind}`;
}

export function generateSaveJwt(data: GooglePassData): string {
  const { CLIENT_EMAIL, PRIVATE_KEY } = getWalletConfig();
  const claims = {
    iss:     CLIENT_EMAIL,
    aud:     'google',
    origins: [process.env.NEXT_PUBLIC_APP_URL ?? 'http://localhost:3000'],
    typ:     'savetowallet',
    payload: { loyaltyObjects: [buildLoyaltyObject(data)] },
  };
  const token = jwt.sign(claims, PRIVATE_KEY, { algorithm: 'RS256' });
  return `https://pay.google.com/gp/v/save/${token}`;
}

/* ── Main issuance API ────────────────────────────────────────────────────── */

export async function issueGooglePass(params: {
  passId:         string;
  restaurantId:   string;
  customerId:     string;
  firstName:      string;
  lastName:       string;
  totalPoints:    number;
  stampsCount:    number;
  qrToken:        string;
  /** 8-char short code stored in wallet_passes.short_code, shown under QR for manual entry */
  shortCode?:     string;
  restaurantName: string;
  primaryColor:   string;
  logoUrl:        string | null;
  passKind:       'stamps' | 'points' | 'event';
  configJson:     Record<string, unknown>;
}): Promise<{
  saveUrl:  string;
  objectId: string;
  classId:  string;
  synced:   boolean;
}> {
  const {
    passId, restaurantId, customerId, firstName, lastName,
    totalPoints, stampsCount, qrToken, shortCode, restaurantName,
    primaryColor, logoUrl, passKind, configJson,
  } = params;

  const { ISSUER_ID } = getWalletConfig();
  const classId  = `${ISSUER_ID}.r${restaurantId.replace(/-/g, '')}_${passKind}`;
  const objectId = `${ISSUER_ID}.p${passId.replace(/-/g, '')}`;

  const data: GooglePassData = {
    objectId,
    classId,
    customerId,
    displayName:     `${firstName} ${lastName}`,
    totalPoints,
    stampsCount,
    stampsTotal:     Number(configJson.stamps_total     ?? 10),
    rewardThreshold: Number(configJson.reward_threshold ?? 100),
    rewardMessage:   String(configJson.reward_message   ?? 'Récompense offerte !'),
    qrToken,
    shortCode,
    restaurantName,
    primaryColor,
    passKind,
  };

  // Sequential: LoyaltyClass MUST be fully registered before LoyaltyObject creation.
  // Google rejects object installs when the referenced class doesn't exist yet.
  const saveUrl = generateSaveJwt(data);

  const classResult = await ensureLoyaltyClass({
    classId, restaurantName, primaryColor, passKind, logoUrl,
  });

  if (!classResult.ok) {
    console.error('[issueGooglePass] ensureLoyaltyClass failed — skipping object creation. classId:', classId);
    return { saveUrl, objectId, classId, synced: false };
  }

  const objectResult = await createLoyaltyObject(data);
  const synced = objectResult.ok;

  return { saveUrl, objectId, classId, synced };
}

/* ── Backward-compat wrapper (keeps /api/wallet/[customerId] working) ─────── */

export async function generateWalletUrl(params: CreateCardParams): Promise<string> {
  const { customerId, firstName, totalPoints, restaurantName, restaurantId, primaryColor } = params;

  // Use restaurantId-based naming (same convention as issueGooglePass) so classIds
  // remain stable even if the restaurant's slug is later renamed.
  // Default passKind 'points' for this legacy JWT-only path (no REST API call).
  const { ISSUER_ID } = getWalletConfig();
  const classId  = computeClassId(restaurantId, 'points');
  const objectId = `${ISSUER_ID}.p${customerId.replace(/-/g, '')}`;

  const data: GooglePassData = {
    objectId,
    classId,
    customerId,
    displayName:     firstName,
    totalPoints,
    stampsCount:     0,
    stampsTotal:     10,
    rewardThreshold: 100,
    rewardMessage:   'Récompense offerte !',
    qrToken:         customerId,
    restaurantName,
    primaryColor,
    passKind:        'points',
  };

  return generateSaveJwt(data);
}
/**
 * KPI Engine — runtime resolver.
 *
 * getRestaurantKPIs(restaurantId) returns the full list of KPIs a restaurant
 * has access to, based on their plan, with availability flags and the
 * restaurant's own settings (e.g. average_ticket).
 *
 * NOTE: This module does NOT compute KPI values — it only resolves
 * which KPIs are available and what settings are present.
 * Actual computation belongs in a future kpi-calculator layer.
 */

import { supabaseAdmin } from '@/lib/supabase-admin';

/* ── Types ─────────────────────────────────────────────────────────────────── */

export interface KpiRow {
  id:          string;
  key:         string;
  name:        string;
  description: string;
  category:    'growth' | 'retention' | 'revenue' | 'engagement';
  is_active:   boolean;
}

export interface ResolvedKpi {
  kpi:     KpiRow;
  /** true = plan grants access AND plan_kpis.enabled = true AND kpi.is_active = true */
  enabled: boolean;
}

export interface RestaurantKpiContext {
  restaurantId: string;
  planId:       string | null;
  planKey:      string;
  kpis:         ResolvedKpi[];
  /** All restaurant_settings values for this restaurant, keyed by settings key */
  settings:     Record<string, string>;
}

/* ── Main resolver ─────────────────────────────────────────────────────────── */

export async function getRestaurantKPIs(restaurantId: string): Promise<RestaurantKpiContext> {
  // 1. Load restaurant (plan_id + plan key)
  const { data: restaurant } = await supabaseAdmin
    .from('restaurants')
    .select('plan_id, plan')
    .eq('id', restaurantId)
    .maybeSingle();

  const planId  = restaurant?.plan_id ?? null;
  const planKey = restaurant?.plan   ?? 'free';

  // 2. Load restaurant_settings
  const { data: settingsRows } = await supabaseAdmin
    .from('restaurant_settings')
    .select('key, value')
    .eq('restaurant_id', restaurantId);

  const settings = Object.fromEntries(
    (settingsRows ?? []).map((s) => [s.key, s.value])
  );

  // 3. Load all active KPIs
  const { data: allKpis } = await supabaseAdmin
    .from('kpis')
    .select('id, key, name, description, category, is_active')
    .eq('is_active', true)
    .order('category')
    .order('name');

  if (!allKpis || allKpis.length === 0) {
    return { restaurantId, planId, planKey, kpis: [], settings };
  }

  // 4. Load plan_kpis for this plan (if plan_id exists)
  let enabledKpiIds = new Set<string>();
  if (planId) {
    const { data: planKpis } = await supabaseAdmin
      .from('plan_kpis')
      .select('kpi_id, enabled')
      .eq('plan_id', planId)
      .eq('enabled', true);

    enabledKpiIds = new Set((planKpis ?? []).map((pk) => pk.kpi_id));
  }

  // 5. Resolve: a KPI is enabled iff the plan has it enabled
  const kpis: ResolvedKpi[] = allKpis.map((kpi) => ({
    kpi:     kpi as KpiRow,
    enabled: enabledKpiIds.has(kpi.id),
  }));

  return { restaurantId, planId, planKey, kpis, settings };
}

/* ── Convenience: enabled KPIs only ───────────────────────────────────────── */

export async function getEnabledKpis(restaurantId: string): Promise<KpiRow[]> {
  const ctx = await getRestaurantKPIs(restaurantId);
  return ctx.kpis.filter((r) => r.enabled).map((r) => r.kpi);
}
import { supabaseAdmin } from '@/lib/supabase-admin';

export async function auditLog(params: {
  restaurantId: string;
  actorId?: string;
  action: string;
  targetType?: string;
  targetId?: string;
  metadata?: Record<string, unknown>;
}) {
  // Fire-and-forget insert — never block the main operation
  void supabaseAdmin.from('audit_log').insert({
    restaurant_id: params.restaurantId,
    actor_id: params.actorId ?? null,
    action: params.action,
    target_type: params.targetType ?? null,
    target_id: params.targetId ?? null,
    metadata: params.metadata ?? null,
  }).then(({ error }) => {
    if (error) console.error('[audit] insert failed:', error.message);
  });
}
