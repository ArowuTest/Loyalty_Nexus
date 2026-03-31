"use client";
import { useEffect, useState } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI from "@/lib/api";
import { UserPlus, ShieldOff, Crown, DollarSign, Settings, Wrench } from "lucide-react";

interface AdminUser {
  id: string;
  email: string;
  full_name: string;
  role: "super_admin" | "finance" | "operations" | "content";
  is_active: boolean;
  last_login_at: string | null;
  created_at: string;
}

const ROLE_CONFIG = {
  super_admin: { label: "Super Admin",   icon: Crown,       color: "text-yellow-400", bg: "bg-yellow-500/10 border-yellow-500/30" },
  finance:     { label: "Finance",       icon: DollarSign,  color: "text-green-400",  bg: "bg-green-500/10 border-green-500/30" },
  operations:  { label: "Operations",    icon: Settings,    color: "text-blue-400",   bg: "bg-blue-500/10 border-blue-500/30" },
  content:     { label: "Content",       icon: Wrench,      color: "text-purple-400", bg: "bg-purple-500/10 border-purple-500/30" },
};

export default function TeamPage() {
  const [admins, setAdmins]     = useState<AdminUser[]>([]);
  const [loading, setLoading]   = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm]         = useState({ email: "", password: "", full_name: "", role: "operations" as AdminUser["role"] });
  const [error, setError]       = useState("");
  const [saving, setSaving]     = useState(false);

  const myRole = typeof window !== "undefined" ? localStorage.getItem("admin_role") : "";

  const load = () => {
    adminAPI.req<{ admins: AdminUser[] }>("GET", "/admin/auth/admins")
      .then(r => setAdmins(r.admins))
      .catch(() => setError("Insufficient permissions — super_admin required"))
      .finally(() => setLoading(false));
  };
  useEffect(() => { load(); }, []);

  const createAdmin = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true); setError("");
    try {
      await adminAPI.req("POST", "/admin/auth/admins", form);
      setShowForm(false);
      setForm({ email: "", password: "", full_name: "", role: "operations" });
      load();
    } catch (err: any) {
      setError(err.message || "Failed to create admin");
    } finally {
      setSaving(false);
    }
  };

  const deactivate = async (id: string, name: string) => {
    if (!confirm(`Deactivate ${name}?`)) return;
    try {
      await adminAPI.req("DELETE", `/admin/auth/admins/${id}`, undefined);
      setAdmins(a => a.map(x => x.id === id ? { ...x, is_active: false } : x));
    } catch { setError("Failed to deactivate"); }
  };

  return (
    <AdminShell>
      <div className="p-6 max-w-4xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-black text-white">Admin Team</h1>
            <p className="text-sm text-gray-500 mt-0.5">Manage admin accounts and RBAC roles</p>
          </div>
          {myRole === "super_admin" && (
            <button
              onClick={() => setShowForm(v => !v)}
              className="flex items-center gap-2 px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-bold"
            >
              <UserPlus size={15} /> Add Admin
            </button>
          )}
        </div>

        {error && (
          <div className="bg-red-500/10 border border-red-500/30 rounded-xl px-4 py-3 text-red-400 text-sm">{error}</div>
        )}

        {/* Create form */}
        {showForm && (
          <form onSubmit={createAdmin} className="bg-gray-900 border border-gray-700 rounded-2xl p-5 space-y-4">
            <h2 className="font-bold text-white">New Admin Account</h2>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <label className="text-xs text-gray-400 font-bold uppercase">Full Name</label>
                <input value={form.full_name} onChange={e => setForm(f => ({...f, full_name: e.target.value}))}
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-indigo-500"
                  placeholder="Jane Smith" />
              </div>
              <div className="space-y-1">
                <label className="text-xs text-gray-400 font-bold uppercase">Email</label>
                <input type="email" required value={form.email} onChange={e => setForm(f => ({...f, email: e.target.value}))}
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-indigo-500"
                  placeholder="jane@company.com" />
              </div>
              <div className="space-y-1">
                <label className="text-xs text-gray-400 font-bold uppercase">Password</label>
                <input type="password" required minLength={8} value={form.password} onChange={e => setForm(f => ({...f, password: e.target.value}))}
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-indigo-500"
                  placeholder="Min 8 characters" />
              </div>
              <div className="space-y-1">
                <label className="text-xs text-gray-400 font-bold uppercase">Role</label>
                <select value={form.role} onChange={e => setForm(f => ({...f, role: e.target.value as AdminUser["role"]}))}
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-indigo-500">
                  <option value="operations">Operations</option>
                  <option value="finance">Finance</option>
                  <option value="content">Content</option>
                  <option value="super_admin">Super Admin</option>
                </select>
              </div>
            </div>
            <div className="flex gap-2 pt-2">
              <button type="button" onClick={() => setShowForm(false)} className="px-4 py-2 rounded-lg text-gray-400 text-sm hover:text-white">Cancel</button>
              <button type="submit" disabled={saving} className="px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-bold disabled:opacity-50">
                {saving ? "Creating…" : "Create Admin"}
              </button>
            </div>
          </form>
        )}

        {/* Admin list */}
        {loading ? (
          <p className="text-gray-500 text-sm">Loading…</p>
        ) : (
          <div className="space-y-3">
            {admins.map(admin => {
              const rc = ROLE_CONFIG[admin.role] || ROLE_CONFIG.operations;
              const Icon = rc.icon;
              return (
                <div key={admin.id} className={`flex items-center justify-between bg-gray-900 border rounded-2xl px-5 py-4 ${!admin.is_active ? "opacity-40" : ""}`}>
                  <div className="flex items-center gap-4">
                    <div className="w-10 h-10 rounded-full bg-gray-800 flex items-center justify-center text-lg font-black text-white">
                      {admin.full_name?.[0] || (admin.email?.[0] ?? "?").toUpperCase()}
                    </div>
                    <div>
                      <p className="text-white font-semibold text-sm">{admin.full_name || "—"}</p>
                      <p className="text-gray-500 text-xs">{admin.email}</p>
                      <p className="text-gray-600 text-xs mt-0.5">
                        {admin.last_login_at ? `Last login: ${new Date(admin.last_login_at).toLocaleDateString()}` : "Never logged in"}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className={`flex items-center gap-1.5 text-xs font-bold px-3 py-1 rounded-full border ${rc.bg} ${rc.color}`}>
                      <Icon size={11} /> {rc.label}
                    </span>
                    {myRole === "super_admin" && admin.is_active && (
                      <button onClick={() => deactivate(admin.id, admin.full_name || admin.email)}
                        className="p-2 text-gray-500 hover:text-red-400 transition-colors" title="Deactivate">
                        <ShieldOff size={15} />
                      </button>
                    )}
                    {!admin.is_active && <span className="text-xs text-red-500 font-bold">DEACTIVATED</span>}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </AdminShell>
  );
}
