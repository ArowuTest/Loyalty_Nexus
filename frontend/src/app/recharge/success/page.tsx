"use client";
import { useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense } from "react";
import { Loader2 } from "lucide-react";

// Legacy URL — the new flow redirects to /recharge?payment=success&reference=REF
// Redirect any existing links to the recharge page preserving the reference.
function RedirectContent() {
  const router = useRouter();
  const params = useSearchParams();
  const ref = params.get("ref") || params.get("reference") || "";

  useEffect(() => {
    const dest = ref
      ? `/recharge?payment=success&reference=${encodeURIComponent(ref)}`
      : "/recharge";
    router.replace(dest);
  }, [ref, router]);

  return (
    <div className="min-h-screen bg-[#080808] flex items-center justify-center">
      <Loader2 className="w-8 h-8 text-amber-400 animate-spin" />
    </div>
  );
}

export default function RechargeSuccessLegacy() {
  return (
    <Suspense fallback={
      <div className="min-h-screen bg-[#080808] flex items-center justify-center">
        <Loader2 className="w-8 h-8 text-amber-400 animate-spin" />
      </div>
    }>
      <RedirectContent />
    </Suspense>
  );
}
