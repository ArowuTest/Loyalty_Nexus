"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import adminAPI from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [phone, setPhone] = useState("");
  const [otp, setOtp] = useState("");
  const [step, setStep] = useState<"phone"|"otp">("phone");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const send = async () => {
    setLoading(true); setError("");
    try {
      await adminAPI.req("POST", "/auth/otp/send", { phone_number: phone, purpose: "login" });
      setStep("otp");
    } catch(e: unknown) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  };

  const verify = async () => {
    setLoading(true); setError("");
    try {
      const result = await adminAPI.req<{token: string}>("POST", "/auth/otp/verify", { phone_number: phone, code: otp, purpose: "login" });
      adminAPI.setToken(result.token);
      router.push("/dashboard");
    } catch(e: unknown) { setError(e instanceof Error ? e.message : "Invalid OTP"); }
    finally { setLoading(false); }
  };

  return (
    <div style={{ minHeight: "100vh", display: "flex", alignItems: "center", justifyContent: "center", padding: 24 }}>
      <div className="card" style={{ width: "100%", maxWidth: 380, padding: 32 }}>
        <div style={{ textAlign: "center", marginBottom: 28 }}>
          <div style={{ fontSize: 40, marginBottom: 8 }}>⚡</div>
          <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff" }}>Admin Cockpit</h1>
          <p style={{ color: "#828cb4", fontSize: 13, marginTop: 4 }}>Loyalty Nexus Operations</p>
        </div>
        {step === "phone" ? (
          <>
            <label style={{ color: "#828cb4", fontSize: 12, display: "block", marginBottom: 6 }}>Admin phone number</label>
            <input className="input" type="tel" placeholder="080X XXX XXXX" value={phone} onChange={e => setPhone(e.target.value)} onKeyDown={e => e.key === "Enter" && send()} style={{ marginBottom: 16 }} />
            <button className="btn-primary" style={{ width: "100%" }} onClick={send} disabled={loading}>{loading ? "Sending…" : "Send OTP →"}</button>
          </>
        ) : (
          <>
            <label style={{ color: "#828cb4", fontSize: 12, display: "block", marginBottom: 6 }}>Enter OTP</label>
            <input className="input" type="number" placeholder="——" value={otp} onChange={e => setOtp(e.target.value)} style={{ marginBottom: 16, textAlign: "center", fontSize: 24, letterSpacing: 12 }} />
            <button className="btn-primary" style={{ width: "100%", marginBottom: 8 }} onClick={verify} disabled={loading}>{loading ? "Verifying…" : "Enter Dashboard →"}</button>
            <button className="btn-outline" style={{ width: "100%", fontSize: 13 }} onClick={() => setStep("phone")}>← Change number</button>
          </>
        )}
        {error && <p style={{ color: "#f43f5e", fontSize: 13, marginTop: 12, textAlign: "center" }}>{error}</p>}
      </div>
    </div>
  );
}