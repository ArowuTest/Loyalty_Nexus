"use client";

import { useState } from "react";
import { motion } from "framer-motion";
import { useRouter } from "next/navigation";
import AppShell from "@/components/layout/AppShell";
import { useStore } from "@/store/useStore";
import api from "@/lib/api";
import toast, { Toaster } from "react-hot-toast";
import {
  Settings, Phone, Wallet, Bell, Shield, LogOut, ChevronRight, ExternalLink
} from "lucide-react";

export default function SettingsPage() {
  const router = useRouter();
  const { logout, user } = useStore();
  const [momoNumber, setMomoNumber] = useState("");
  const [linking, setLinking] = useState(false);
  const [showMoMo, setShowMoMo] = useState(false);

  const handleLinkMoMo = async () => {
    if (momoNumber.length < 11) { toast.error("Enter valid MoMo number"); return; }
    setLinking(true);
    try {
      const result = await api.requestMoMoLink(momoNumber) as { account_name: string };
      toast.success(`MoMo linked: ${result.account_name}`);
      await api.verifyMoMo(momoNumber);
      setShowMoMo(false);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "Failed to link MoMo");
    } finally {
      setLinking(false);
    }
  };

  const MENU_ITEMS = [
    {
      section: "Account",
      items: [
        { icon: Phone, label: "Phone Number", sub: user?.phone_number || "—", action: null },
        { icon: Wallet, label: "Link MoMo Wallet", sub: "Required for cash prizes", action: () => setShowMoMo(true) },
      ]
    },
    {
      section: "Preferences",
      items: [
        { icon: Bell, label: "Notifications", sub: "SMS and push alerts", action: null },
        { icon: Shield, label: "Privacy Policy", sub: "How we use your data", action: null },
      ]
    }
  ];

  return (
    <AppShell>
      <Toaster position="top-center" toastOptions={{ style: { background: "#1c2038", color: "#fff" } }} />

      <div className="max-w-5xl mx-auto px-4 md:px-6 py-6 space-y-5">
        <div className="flex items-center gap-3">
          <Settings className="text-nexus-400" size={24} />
          <h1 className="text-2xl font-bold font-display text-white">Settings</h1>
        </div>

        {MENU_ITEMS.map(({ section, items }) => (
          <div key={section}>
            <p className="text-[rgb(130_140_180)] text-xs uppercase tracking-widest mb-2 px-1">{section}</p>
            <div className="nexus-card overflow-hidden divide-y divide-nexus-600/10">
              {items.map((item, i) => (
                <motion.button
                  key={i}
                  onClick={item.action || undefined}
                  disabled={!item.action}
                  className="w-full flex items-center gap-3 p-4 text-left hover:bg-white/5 transition-colors disabled:cursor-default"
                  whileTap={item.action ? { scale: 0.98 } : {}}
                >
                  <div className="w-9 h-9 rounded-xl bg-nexus-600/20 flex items-center justify-center">
                    <item.icon size={16} className="text-nexus-400" />
                  </div>
                  <div className="flex-1">
                    <p className="text-white text-sm font-medium">{item.label}</p>
                    <p className="text-[rgb(130_140_180)] text-xs">{item.sub}</p>
                  </div>
                  {item.action && <ChevronRight size={14} className="text-[rgb(130_140_180)]" />}
                </motion.button>
              ))}
            </div>
          </div>
        ))}

        <button
          onClick={() => { logout(); api.clearToken(); router.push("/"); }}
          className="w-full nexus-card p-4 flex items-center gap-3 text-red-400 hover:bg-red-400/5 transition-colors"
        >
          <div className="w-9 h-9 rounded-xl bg-red-400/10 flex items-center justify-center">
            <LogOut size={16} />
          </div>
          <span className="font-medium">Sign Out</span>
        </button>
      </div>

      {/* MoMo link modal */}
      {showMoMo && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-end justify-center p-4" onClick={() => setShowMoMo(false)}>
          <motion.div
            initial={{ y: 50 }}
            animate={{ y: 0 }}
            className="nexus-card w-full max-w-md p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="text-lg font-bold text-white mb-1">Link MoMo Wallet</h3>
            <p className="text-[rgb(130_140_180)] text-sm mb-4">
              Cash prizes will be sent directly to your MoMo wallet.
            </p>
            <input
              type="tel"
              placeholder="080X XXX XXXX (MoMo number)"
              value={momoNumber}
              onChange={(e) => setMomoNumber(e.target.value)}
              className="nexus-input mb-4"
            />
            <div className="flex gap-2">
              <button onClick={() => setShowMoMo(false)} className="nexus-btn-outline flex-1">Cancel</button>
              <button onClick={handleLinkMoMo} disabled={linking} className="nexus-btn-primary flex-1">
                {linking ? "Linking…" : "Link Wallet"}
              </button>
            </div>
          </motion.div>
        </div>
      )}
    </AppShell>
  );
}
