"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import AppShell from "@/components/layout/AppShell";
import { useStore } from "@/store/useStore";
import { api } from "@/lib/api";
import { User, Mail, Phone, CheckCircle, AlertCircle, Loader2, ArrowLeft } from "lucide-react";
import Link from "next/link";

export default function ProfilePage() {
  const router = useRouter();
  const { user, setUser } = useStore();

  const [displayName, setDisplayName] = useState(user?.display_name ?? "");
  const [email, setEmail]             = useState(user?.email ?? "");
  const [saving, setSaving]           = useState(false);
  const [status, setStatus]           = useState<"idle" | "success" | "error">("idle");
  const [errorMsg, setErrorMsg]       = useState("");

  // Sync fields if user store updates (e.g. after page reload)
  useEffect(() => {
    setDisplayName(user?.display_name ?? "");
    setEmail(user?.email ?? "");
  }, [user?.display_name, user?.email]);

  const handleSave = async () => {
    setSaving(true);
    setStatus("idle");
    setErrorMsg("");

    // Basic email validation
    if (email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      setStatus("error");
      setErrorMsg("Please enter a valid email address.");
      setSaving(false);
      return;
    }

    try {
      const updated = await api.updateProfile({
        display_name: displayName.trim() || null,
        email:        email.trim()       || null,
      });
      // Merge updated fields back into the store
      if (user) {
        setUser({
          ...user,
          display_name: updated.display_name ?? (displayName.trim() || undefined),
          email:        updated.email        ?? (email.trim()        || undefined),
        });
      }
      setStatus("success");
    } catch (err: unknown) {
      setStatus("error");
      setErrorMsg(err instanceof Error ? err.message : "Something went wrong.");
    } finally {
      setSaving(false);
    }
  };

  const tierColor: Record<string, string> = {
    BRONZE: "#CD7F32", SILVER: "#C0C0C0", GOLD: "#F5A623", PLATINUM: "#E5E4E2", DIAMOND: "#B9F2FF",
  };
  const tier  = (user?.tier ?? "BRONZE").toUpperCase();
  const color = tierColor[tier] ?? "#CD7F32";

  function getInitials() {
    if (displayName.trim()) {
      const parts = displayName.trim().split(/\s+/);
      if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
      return parts[0].slice(0, 2).toUpperCase();
    }
    return (user?.phone_number ?? "").slice(-2) || "?";
  }

  return (
    <AppShell>
      <div className="max-w-lg mx-auto px-4 py-8">

        {/* Back link */}
        <Link
          href="/dashboard"
          className="inline-flex items-center gap-2 text-white/40 hover:text-white text-[13px] mb-6 transition-colors"
        >
          <ArrowLeft size={14} />
          Back to Dashboard
        </Link>

        {/* Page header */}
        <div className="mb-8">
          <h1 className="text-2xl font-black text-white">Edit Profile</h1>
          <p className="text-white/40 text-[13px] mt-1">Update your display name and email address.</p>
        </div>

        {/* Avatar preview */}
        <div className="flex items-center gap-4 mb-8">
          <div
            className="w-16 h-16 rounded-full flex items-center justify-center text-xl font-black flex-shrink-0"
            style={{ background: `${color}22`, border: `2px solid ${color}55`, color }}
          >
            {getInitials()}
          </div>
          <div>
            <p className="text-white font-bold text-[15px]">
              {displayName.trim() || user?.phone_number || ""}
            </p>
            <p className="text-white/40 text-[12px] mt-0.5">
              {email || "No email set"}
            </p>
          </div>
        </div>

        {/* Form card */}
        <div
          className="rounded-2xl p-6 space-y-5"
          style={{
            background: "rgba(255,255,255,0.03)",
            border: "1px solid rgba(255,255,255,0.07)",
          }}
        >
          {/* Phone — read-only */}
          <div>
            <label className="block text-[12px] font-bold text-white/50 uppercase tracking-wider mb-2">
              Phone Number
            </label>
            <div
              className="flex items-center gap-3 px-4 py-3 rounded-xl"
              style={{ background: "rgba(255,255,255,0.03)", border: "1px solid rgba(255,255,255,0.06)" }}
            >
              <Phone size={15} className="text-white/30 flex-shrink-0" />
              <span className="text-white/50 text-[14px] font-mono">
                {user?.phone_number ?? "—"}
              </span>
              <span className="ml-auto text-[11px] text-white/25 italic">Cannot change</span>
            </div>
          </div>

          {/* Display name */}
          <div>
            <label className="block text-[12px] font-bold text-white/50 uppercase tracking-wider mb-2">
              Display Name
            </label>
            <div
              className="flex items-center gap-3 px-4 py-3 rounded-xl transition-all focus-within:border-[rgba(245,166,35,0.4)]"
              style={{ background: "rgba(255,255,255,0.04)", border: "1px solid rgba(255,255,255,0.09)" }}
            >
              <User size={15} className="text-white/40 flex-shrink-0" />
              <input
                type="text"
                value={displayName}
                onChange={(e) => { setDisplayName(e.target.value); setStatus("idle"); }}
                placeholder="e.g. Amara Okafor"
                maxLength={60}
                className="flex-1 bg-transparent text-white text-[14px] outline-none placeholder:text-white/25"
              />
            </div>
            <p className="text-white/25 text-[11px] mt-1.5">
              This is how your name appears across the platform.
            </p>
          </div>

          {/* Email */}
          <div>
            <label className="block text-[12px] font-bold text-white/50 uppercase tracking-wider mb-2">
              Email Address
            </label>
            <div
              className="flex items-center gap-3 px-4 py-3 rounded-xl transition-all focus-within:border-[rgba(245,166,35,0.4)]"
              style={{ background: "rgba(255,255,255,0.04)", border: "1px solid rgba(255,255,255,0.09)" }}
            >
              <Mail size={15} className="text-white/40 flex-shrink-0" />
              <input
                type="email"
                value={email}
                onChange={(e) => { setEmail(e.target.value); setStatus("idle"); }}
                placeholder="you@example.com"
                maxLength={120}
                className="flex-1 bg-transparent text-white text-[14px] outline-none placeholder:text-white/25"
              />
            </div>
            <p className="text-white/25 text-[11px] mt-1.5">
              Used for prize notifications and account recovery.
            </p>
          </div>

          {/* Status message */}
          {status === "success" && (
            <div
              className="flex items-center gap-2 px-4 py-3 rounded-xl text-[13px]"
              style={{ background: "rgba(34,197,94,0.08)", border: "1px solid rgba(34,197,94,0.2)", color: "#4ade80" }}
            >
              <CheckCircle size={15} className="flex-shrink-0" />
              Profile updated successfully.
            </div>
          )}
          {status === "error" && (
            <div
              className="flex items-center gap-2 px-4 py-3 rounded-xl text-[13px]"
              style={{ background: "rgba(239,68,68,0.08)", border: "1px solid rgba(239,68,68,0.2)", color: "#f87171" }}
            >
              <AlertCircle size={15} className="flex-shrink-0" />
              {errorMsg}
            </div>
          )}

          {/* Save button */}
          <button
            onClick={handleSave}
            disabled={saving}
            className="w-full py-3.5 rounded-xl font-black text-[14px] transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
            style={{
              background: saving ? "rgba(245,166,35,0.3)" : "linear-gradient(135deg, #F5A623, #E8950F)",
              color: "#0d0e14",
              boxShadow: saving ? "none" : "0 4px 20px rgba(245,166,35,0.25)",
            }}
          >
            {saving ? (
              <>
                <Loader2 size={16} className="animate-spin" />
                Saving…
              </>
            ) : (
              "Save Changes"
            )}
          </button>
        </div>
      </div>
    </AppShell>
  );
}
