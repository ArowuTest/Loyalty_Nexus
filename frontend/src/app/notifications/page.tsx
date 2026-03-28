"use client";

import { useEffect, useState, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import AppShell from "@/components/layout/AppShell";
import api from "@/lib/api";
import { cn } from "@/lib/utils";
import {
  Bell, BellOff, CheckCheck, Loader2, RefreshCw,
  Zap, Trophy, Flame, Sparkles, Globe, Gift, Megaphone,
} from "lucide-react";
import toast, { Toaster } from "react-hot-toast";

// ── Types ──────────────────────────────────────────────────────────────────
interface Notification {
  id: string;
  type: string;
  title: string;
  body: string;
  is_read: boolean;
  created_at: string;
  metadata?: Record<string, unknown>;
}

interface NotifResponse {
  notifications: Notification[];
  unread_count: number;
  cursor?: string;
}

// ── Type config — icon + colour per notification type ──────────────────────
const TYPE_CONFIG: Record<string, {
  icon: React.ReactNode;
  accent: string;
  bg: string;
  border: string;
}> = {
  spin_win: {
    icon: <Trophy size={18} />,
    accent: "text-yellow-400",
    bg:     "bg-yellow-500/10",
    border: "border-yellow-500/20",
  },
  streak_warn: {
    icon: <Flame size={18} />,
    accent: "text-orange-400",
    bg:     "bg-orange-500/10",
    border: "border-orange-500/20",
  },
  studio_ready: {
    icon: <Sparkles size={18} />,
    accent: "text-purple-400",
    bg:     "bg-purple-500/10",
    border: "border-purple-500/20",
  },
  wars_result: {
    icon: <Globe size={18} />,
    accent: "text-green-400",
    bg:     "bg-green-500/10",
    border: "border-green-500/20",
  },
  bonus_pulse: {
    icon: <Zap size={18} />,
    accent: "text-nexus-400",
    bg:     "bg-nexus-500/10",
    border: "border-nexus-500/20",
  },
  draw_result: {
    icon: <Gift size={18} />,
    accent: "text-pink-400",
    bg:     "bg-pink-500/10",
    border: "border-pink-500/20",
  },
  marketing: {
    icon: <Megaphone size={18} />,
    accent: "text-sky-400",
    bg:     "bg-sky-500/10",
    border: "border-sky-500/20",
  },
};

const DEFAULT_CONFIG = {
  icon:   <Bell size={18} />,
  accent: "text-white/40",
  bg:     "bg-white/5",
  border: "border-white/10",
};

function getTypeConfig(type: string) {
  return TYPE_CONFIG[type] ?? DEFAULT_CONFIG;
}

// ── Time formatting ────────────────────────────────────────────────────────
function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const m = Math.floor(diff / 60000);
  const h = Math.floor(diff / 3600000);
  const d = Math.floor(diff / 86400000);
  if (m < 1)  return "Just now";
  if (m < 60) return `${m}m ago`;
  if (h < 24) return `${h}h ago`;
  if (d < 7)  return `${d}d ago`;
  return new Date(iso).toLocaleDateString("en-NG", { day: "numeric", month: "short" });
}

// ── Notification Card ──────────────────────────────────────────────────────
function NotifCard({
  notif,
  onRead,
}: {
  notif: Notification;
  onRead: (id: string) => void;
}) {
  const cfg = getTypeConfig(notif.type);

  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, x: -20 }}
      onClick={() => !notif.is_read && onRead(notif.id)}
      className={cn(
        "nexus-card p-4 flex items-start gap-3 transition-all cursor-default",
        !notif.is_read && "border-nexus-500/20 bg-nexus-600/5 cursor-pointer",
      )}
    >
      {/* Icon */}
      <div className={cn(
        "w-10 h-10 rounded-2xl flex items-center justify-center flex-shrink-0 border",
        cfg.bg, cfg.border, cfg.accent,
      )}>
        {cfg.icon}
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className="flex items-start justify-between gap-2">
          <p className={cn(
            "text-sm leading-snug",
            notif.is_read ? "text-white/70 font-normal" : "text-white font-semibold",
          )}>
            {notif.title}
          </p>
          <span className="text-white/25 text-[10px] flex-shrink-0 mt-0.5">
            {timeAgo(notif.created_at)}
          </span>
        </div>
        <p className="text-white/45 text-xs mt-1 leading-relaxed line-clamp-2">
          {notif.body}
        </p>
      </div>

      {/* Unread dot */}
      {!notif.is_read && (
        <div className="w-2 h-2 rounded-full bg-nexus-400 flex-shrink-0 mt-1" />
      )}
    </motion.div>
  );
}

// ── Empty state ────────────────────────────────────────────────────────────
function EmptyState() {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="flex flex-col items-center justify-center py-20 text-center"
    >
      <div className="w-20 h-20 rounded-3xl bg-nexus-600/10 border border-nexus-600/20 flex items-center justify-center mb-5">
        <BellOff size={32} className="text-nexus-400/40" />
      </div>
      <p className="text-white font-semibold text-lg mb-2">All caught up</p>
      <p className="text-white/40 text-sm max-w-xs leading-relaxed">
        You have no notifications right now. We'll let you know when you win a spin,
        when your AI generation is ready, or when your streak is about to expire.
      </p>
    </motion.div>
  );
}

// ── Main Page ──────────────────────────────────────────────────────────────
export default function NotificationsPage() {
  const [notifs, setNotifs]       = useState<Notification[]>([]);
  const [unreadCount, setUnread]  = useState(0);
  const [loading, setLoading]     = useState(true);
  const [marking, setMarking]     = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.getNotifications() as NotifResponse;
      setNotifs(res.notifications ?? []);
      setUnread(res.unread_count ?? 0);
    } catch {
      // If the endpoint doesn't exist yet, show empty state gracefully
      setNotifs([]);
      setUnread(0);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleRead = useCallback(async (id: string) => {
    // Optimistic update
    setNotifs(prev =>
      prev.map(n => n.id === id ? { ...n, is_read: true } : n)
    );
    setUnread(prev => Math.max(0, prev - 1));
    try {
      await api.markNotificationRead(id);
    } catch { /* silent — optimistic already applied */ }
  }, []);

  const handleMarkAllRead = useCallback(async () => {
    if (unreadCount === 0) return;
    setMarking(true);
    try {
      await api.markAllNotificationsRead();
      setNotifs(prev => prev.map(n => ({ ...n, is_read: true })));
      setUnread(0);
      toast.success("All notifications marked as read");
    } catch {
      toast.error("Failed to mark all as read");
    } finally {
      setMarking(false);
    }
  }, [unreadCount]);

  // Group notifications: Today / Earlier
  const today = new Date();
  today.setHours(0, 0, 0, 0);

  const todayNotifs   = notifs.filter(n => new Date(n.created_at) >= today);
  const earlierNotifs = notifs.filter(n => new Date(n.created_at) < today);

  return (
    <AppShell>
      <Toaster position="top-center" toastOptions={{
        style: { background: "#1c2038", color: "#fff", border: "1px solid rgba(255,255,255,0.08)" },
      }} />

      <div className="max-w-2xl mx-auto px-4 py-6 pb-28 space-y-4">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="relative">
              <Bell size={22} className="text-white" />
              {unreadCount > 0 && (
                <span className="absolute -top-1.5 -right-1.5 min-w-[18px] h-[18px] rounded-full bg-nexus-500 text-white text-[10px] font-bold flex items-center justify-center px-1">
                  {unreadCount > 99 ? "99+" : unreadCount}
                </span>
              )}
            </div>
            <div>
              <h1 className="text-2xl font-bold font-display text-white">Notifications</h1>
              <p className="text-white/40 text-xs mt-0.5">
                {unreadCount > 0 ? `${unreadCount} unread` : "All caught up"}
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={load}
              className="w-9 h-9 rounded-xl bg-white/5 border border-white/10 flex items-center justify-center hover:bg-white/10 transition-colors"
            >
              <RefreshCw size={15} className={cn("text-white/50", loading && "animate-spin")} />
            </button>
            {unreadCount > 0 && (
              <button
                onClick={handleMarkAllRead}
                disabled={marking}
                className="flex items-center gap-1.5 text-xs font-semibold px-3 py-2 rounded-xl bg-nexus-600/20 text-nexus-300 border border-nexus-500/30 hover:bg-nexus-600/30 transition-all disabled:opacity-50"
              >
                {marking ? (
                  <Loader2 size={13} className="animate-spin" />
                ) : (
                  <CheckCheck size={13} />
                )}
                Mark all read
              </button>
            )}
          </div>
        </div>

        {/* Content */}
        {loading ? (
          <div className="space-y-3">
            {[1, 2, 3, 4].map(i => (
              <div key={i} className="nexus-card p-4 flex items-start gap-3 animate-pulse">
                <div className="w-10 h-10 rounded-2xl bg-white/5 flex-shrink-0" />
                <div className="flex-1 space-y-2">
                  <div className="h-3.5 bg-white/8 rounded-lg w-2/3" />
                  <div className="h-3 bg-white/5 rounded-lg w-full" />
                  <div className="h-3 bg-white/5 rounded-lg w-3/4" />
                </div>
              </div>
            ))}
          </div>
        ) : notifs.length === 0 ? (
          <EmptyState />
        ) : (
          <div className="space-y-5">
            <AnimatePresence mode="popLayout">
              {/* Today */}
              {todayNotifs.length > 0 && (
                <div key="today" className="space-y-2">
                  <p className="text-white/30 text-xs font-semibold uppercase tracking-widest px-1">Today</p>
                  {todayNotifs.map(n => (
                    <NotifCard key={n.id} notif={n} onRead={handleRead} />
                  ))}
                </div>
              )}

              {/* Earlier */}
              {earlierNotifs.length > 0 && (
                <div key="earlier" className="space-y-2">
                  <p className="text-white/30 text-xs font-semibold uppercase tracking-widest px-1">Earlier</p>
                  {earlierNotifs.map(n => (
                    <NotifCard key={n.id} notif={n} onRead={handleRead} />
                  ))}
                </div>
              )}
            </AnimatePresence>
          </div>
        )}
      </div>
    </AppShell>
  );
}
