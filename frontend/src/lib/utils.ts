import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatNaira(kobo: number): string {
  return `₦${(kobo / 100).toLocaleString("en-NG", { minimumFractionDigits: 0, maximumFractionDigits: 0 })}`;
}

export function formatPoints(points: number): string {
  if (points >= 1000) return `${(points / 1000).toFixed(1)}k pts`;
  return `${points} pts`;
}

export function truncatePhone(phone: string): string {
  if (phone.length >= 11) return `${phone.slice(0, 4)}****${phone.slice(-4)}`;
  return phone;
}

export const TIER_COLORS: Record<string, string> = {
  BRONZE:   "text-amber-600 bg-amber-50 border-amber-200",
  SILVER:   "text-slate-500 bg-slate-50 border-slate-200",
  GOLD:     "text-yellow-600 bg-yellow-50 border-yellow-200",
  PLATINUM: "text-purple-600 bg-purple-50 border-purple-200",
};

export const TIER_THRESHOLDS = [
  { tier: "BRONZE",   min: 0,    max: 499,  label: "Bronze" },
  { tier: "SILVER",   min: 500,  max: 1499, label: "Silver" },
  { tier: "GOLD",     min: 1500, max: 4999, label: "Gold" },
  { tier: "PLATINUM", min: 5000, max: Infinity, label: "Platinum" },
];
