"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

/**
 * /studio/business-plan — redirect to the main studio page with the
 * 'bizplan' tool pre-selected. The KnowledgeDoc template handles the
 * structured form, generation and output display.
 */
export default function BusinessPlanPage() {
  const router = useRouter();

  useEffect(() => {
    router.replace("/studio?tool=bizplan");
  }, [router]);

  return (
    <div className="min-h-screen bg-[#0a0b14] flex items-center justify-center">
      <div className="flex flex-col items-center gap-3 text-white/40">
        <div className="w-8 h-8 border-2 border-white/20 border-t-yellow-400 rounded-full animate-spin" />
        <p className="text-sm">Loading Business Plan…</p>
      </div>
    </div>
  );
}
