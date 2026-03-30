// ─── Routes ───────────────────────────────────────────────────
export const ROUTES = {
  HOME:      "/",
  DASHBOARD: "/dashboard",
  STUDIO:    "/studio",
  ADMIN:     "/admin",
  SPIN:      "/dashboard#spin",
  ABOUT:     "/about",
  BLOG:      "/blog",
  CAREERS:   "/careers",
  PRIVACY:   "/privacy",
  TERMS:     "/terms",
} as const;

// ─── Tier config ──────────────────────────────────────────────
export type Tier = "bronze" | "silver" | "gold" | "platinum" | "diamond";

export const TIER_CONFIG: Record<Tier, {
  label: string; color: string; icon: string; minPoints: number;
  spinMultiplier: number; pointMultiplier: number;
}> = {
  bronze:   { label: "Bronze",   color: "#CD7F32", icon: "🥉", minPoints: 0,     spinMultiplier: 1,  pointMultiplier: 1 },
  silver:   { label: "Silver",   color: "#C0C0C0", icon: "🥈", minPoints: 5000,  spinMultiplier: 2,  pointMultiplier: 1.2 },
  gold:     { label: "Gold",     color: "#FFD700", icon: "🥇", minPoints: 15000, spinMultiplier: 3,  pointMultiplier: 1.5 },
  platinum: { label: "Platinum", color: "#E5E4E2", icon: "💎", minPoints: 40000, spinMultiplier: 5,  pointMultiplier: 2 },
  diamond:  { label: "Diamond",  color: "#B9F2FF", icon: "💠", minPoints: 100000,spinMultiplier: 10, pointMultiplier: 3 },
};

// ─── AI Tool ──────────────────────────────────────────────────
export interface AITool {
  slug:        string;
  name:        string;
  emoji:       string;
  description: string;
  category:    "chat" | "create" | "learn" | "build";
  point_cost:  number;
  is_free:     boolean;
  is_popular?: boolean;
  is_new?:     boolean;
  ui_template: string;
}

// ─── Spin prize ───────────────────────────────────────────────
export interface SpinPrize {
  id:      string;
  label:   string;
  color:   string;
  value:   string;
  type:    "cash" | "airtime" | "data" | "points" | "spin";
  weight:  number;
}

// ─── Helpers ──────────────────────────────────────────────────
export function formatPoints(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000)     return `${(n / 1_000).toFixed(n >= 10_000 ? 0 : 1)}K`;
  return n.toString();
}

export function formatNaira(n: number): string {
  if (n >= 1_000_000) return `₦${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000)     return `₦${(n / 1_000).toFixed(0)}K`;
  return `₦${n.toLocaleString("en-NG")}`;
}
