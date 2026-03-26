"use client";

import { useEffect, use } from "react";
import { useRouter } from "next/navigation";

/**
 * /studio/tool/[id] — redirect to the main studio page.
 * The main studio page handles tool selection via its drawer UI.
 * We pass the tool id as a query param so the studio page can
 * auto-open the drawer for the requested tool.
 */
export default function ToolPage({ params }: { params: Promise<{ id: string }> }) {
  const router = useRouter();
  const { id } = use(params);

  useEffect(() => {
    router.replace(`/studio?tool=${encodeURIComponent(id)}`);
  }, [id, router]);

  return (
    <div className="min-h-screen bg-[#0a0b14] flex items-center justify-center">
      <div className="flex flex-col items-center gap-3 text-white/40">
        <div className="w-8 h-8 border-2 border-white/20 border-t-nexus-400 rounded-full animate-spin" />
        <p className="text-sm">Loading tool…</p>
      </div>
    </div>
  );
}
