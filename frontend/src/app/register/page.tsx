"use client";
// The registration / login flow lives on the root page (/).
// This redirect ensures any deep-links to /register still work.
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useStore } from "@/store/useStore";

export default function RegisterRedirect() {
  const router = useRouter();
  const { token } = useStore();
  useEffect(() => {
    // If already authenticated, go to dashboard; otherwise go to landing/login
    router.replace(token ? "/dashboard" : "/");
  }, [router, token]);
  return null;
}
