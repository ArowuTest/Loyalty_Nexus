"use client";

import { useEffect, useState } from 'react';
import { useRouter, usePathname } from 'next/navigation';

/**
 * Enterprise PWA Session Persistence (REQ-1.4)
 * Handles JWT storage in localStorage and automatic redirection.
 */
export const useAuth = () => {
  const router = useRouter();
  const pathname = usePathname();
  const [isAuthenticated, setIsAuthenticated] = useState<boolean | null>(null);

  useEffect(() => {
    const token = localStorage.getItem('nexus_token');
    const isAuth = !!token;
    setIsAuthenticated(isAuth);

    if (!isAuth && pathname !== '/register' && pathname !== '/') {
      router.push('/register');
    }
  }, [pathname, router]);

  const setSession = (token: string) => {
    localStorage.setItem('nexus_token', token);
    setIsAuthenticated(true);
  };

  const logout = () => {
    localStorage.removeItem('nexus_token');
    setIsAuthenticated(false);
    router.push('/register');
  };

  return { isAuthenticated, setSession, logout };
};
