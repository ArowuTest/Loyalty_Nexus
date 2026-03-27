"use client";

import useSWR from "swr";
import { useState } from "react";
import AppShell from "@/components/layout/AppShell";
import api from "@/lib/api";
import { Bell, Check, CheckCheck, Megaphone, Trophy, Zap, Gift, AlertCircle } from "lucide-react";
import { cn } from "@/lib/utils";
import { motion, AnimatePresence } from "framer-motion";

// ─── Types ────────────────────────────────────────────────────────────────────

interface Notification {
  id: string;
  type: string;
  title: string;
  body: string;
  is_read: boolean;
  created_at: string;
}

interface NotifResponse {
  notifications: Notification[];
  unread_count: number;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

const TYPE_ICON: Record<string, React.ReactNode> = {
  spin_win:          <Trophy size={16} className="text-brand-gold" />,
  draw_result:       <Gift size={16} className="text-green-400" />,
  studio_ready:      <Zap size={16} className="text-nexus-400" />,
  wars_result:       <Trophy size={16} className="text-green-400" />,
  system:            <Bell size={16} className="text-[rgb(130_140_180)]" />,
  marketing:         <Megaphone size={16} className="text-amber-400" />,
  subscription_warn: <AlertCircle size={16} className="text-amber-400" />,
};

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60_000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function NotificationsPage() {
  const [markingAll, setMarkingAll] = useState(false);

  const { data, mutate, isLoading } = useSWR<NotifResponse>(
    "/notifications",
    () => api.getNotifications() as Promise<NotifResponse>,
    { refreshInterval: 30_000 }
  );

  const notifications = data?.notifications ?? [];
  const unreadCount   = data?.unread_count ?? 0;

  const markRead = async (id: string) => {
    try {
      await api.markNotificationRead(id);
      mutate(d => d ? {
        ...d,
        notifications: d.notifications.map(n => n.id === id ? { ...n, is_read: true } : n),
        unread_count: Math.max(0, d.unread_count - 1),
      } : d, false);
    } catch { /* ignore */ }
  };

  const markAllRead = async () => {
    setMarkingAll(true);
    try {
      await api.markAllNotificationsRead();
      mutate(d => d ? {
        ...d,
        notifications: d.notifications.map(n => ({ ...n, is_read: true })),
        unread_count: 0,
      } : d, false);
    } catch { /* ignore */ } finally {
      setMarkingAll(false);
    }
  };

  return (
    <AppShell>
      <div className="max-w-2xl mx-auto px-4 py-6 space-y-4">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Bell size={22} className="text-brand-gold" />
            <div>
              <h1 className="text-xl font-black text-white">Notifications</h1>
              {unreadCount > 0 && (
                <p className="text-xs text-[rgb(130_140_180)]">{unreadCount} unread</p>
              )}
            </div>
          </div>
          {unreadCount > 0 && (
            <button
              onClick={markAllRead}
              disabled={markingAll}
              className="flex items-center gap-1.5 text-xs font-bold text-nexus-400 hover:text-nexus-300
                         px-3 py-1.5 rounded-xl bg-nexus-500/10 border border-nexus-500/20 transition-colors disabled:opacity-50"
            >
              <CheckCheck size={13} />
              {markingAll ? "Marking…" : "Mark all read"}
            </button>
          )}
        </div>

        {/* Loading skeletons */}
        {isLoading && (
          <div className="space-y-3">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="nexus-card p-4 flex gap-3 animate-pulse">
                <div className="w-10 h-10 rounded-full bg-white/5 shrink-0" />
                <div className="flex-1 space-y-2">
                  <div className="h-3 bg-white/5 rounded w-3/4" />
                  <div className="h-3 bg-white/5 rounded w-1/2" />
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Empty state */}
        {!isLoading && notifications.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-center space-y-3">
            <Bell size={40} className="text-white/10" />
            <p className="text-white/40 font-semibold">All caught up!</p>
            <p className="text-white/20 text-sm">No notifications yet. Spin, earn, win!</p>
          </div>
        )}

        {/* Notification list */}
        {!isLoading && notifications.length > 0 && (
          <AnimatePresence initial={false}>
            <div className="space-y-2">
              {notifications.map((notif, i) => (
                <motion.div
                  key={notif.id}
                  initial={{ opacity: 0, y: 6 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: i * 0.03 }}
                  className={cn(
                    "nexus-card p-4 flex items-start gap-3 cursor-pointer transition-all",
                    !notif.is_read && "border-nexus-500/30 bg-nexus-900/20"
                  )}
                  onClick={() => !notif.is_read && markRead(notif.id)}
                >
                  <div className={cn(
                    "w-10 h-10 rounded-2xl flex items-center justify-center shrink-0",
                    notif.is_read ? "bg-white/5" : "bg-nexus-500/20"
                  )}>
                    {TYPE_ICON[notif.type] ?? <Bell size={16} className="text-[rgb(130_140_180)]" />}
                  </div>

                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between gap-2">
                      <p className={cn(
                        "text-sm font-semibold leading-tight",
                        notif.is_read ? "text-white/70" : "text-white"
                      )}>
                        {notif.title}
                      </p>
                      <span className="text-[10px] text-[rgb(130_140_180)] shrink-0 mt-0.5">
                        {timeAgo(notif.created_at)}
                      </span>
                    </div>
                    <p className={cn(
                      "text-xs mt-0.5 leading-relaxed",
                      notif.is_read ? "text-white/30" : "text-[rgb(130_140_180)]"
                    )}>
                      {notif.body}
                    </p>
                  </div>

                  {!notif.is_read && (
                    <div className="w-2 h-2 rounded-full bg-nexus-400 shrink-0 mt-1" />
                  )}
                  {notif.is_read && (
                    <Check size={14} className="text-white/20 shrink-0 mt-1" />
                  )}
                </motion.div>
              ))}
            </div>
          </AnimatePresence>
        )}
      </div>
    </AppShell>
  );
}
