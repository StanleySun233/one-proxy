'use client';

import {createContext, ReactNode, useCallback, useContext, useEffect, useMemo, useState} from 'react';

import {AUTH_INVALID_EVENT, login as loginRequest, logout as logoutRequest, Session, SESSION_STORAGE_KEY, updateAccount} from '@/lib/api';
import type {ActiveTenant, TenantMembership} from '@/lib/types/auth';

type AuthContextValue = {
  session: Session | null;
  tenantMemberships: TenantMembership[];
  activeTenant: ActiveTenant;
  ready: boolean;
  login: (account: string, password: string) => Promise<Session>;
  rotatePassword: (password: string) => Promise<Session>;
  switchTenant: (tenantId: string | null) => Session | null;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

function tenantRoleRank(role: TenantMembership['role']) {
  return role === 'tenant_admin' ? 2 : 1;
}

function sortTenantMemberships(tenantMemberships: TenantMembership[]) {
  return [...tenantMemberships].sort((left, right) => {
    const roleDiff = tenantRoleRank(right.role) - tenantRoleRank(left.role);
    if (roleDiff !== 0) {
      return roleDiff;
    }
    return left.tenantName.localeCompare(right.tenantName) || left.tenantId.localeCompare(right.tenantId);
  });
}

function resolveActiveTenantId(tenantMemberships: TenantMembership[], activeTenantId: string | null | undefined) {
  if (activeTenantId && tenantMemberships.some((membership) => membership.tenantId === activeTenantId)) {
    return activeTenantId;
  }

  return tenantMemberships[0]?.tenantId || null;
}

function normalizeSession(session: Session): Session {
  const tenantMemberships = sortTenantMemberships(session.tenantMemberships || []);

  return {
    ...session,
    tenantMemberships,
    activeTenantId: resolveActiveTenantId(tenantMemberships, session.activeTenantId)
  };
}

export function AuthProvider({children}: {children: ReactNode}) {
  const [session, setSession] = useState<Session | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    const stored = window.localStorage.getItem(SESSION_STORAGE_KEY);

    if (stored) {
      try {
        const parsed = normalizeSession(JSON.parse(stored) as Session);
        if (Date.parse(parsed.expiresAt) > Date.now()) {
          setSession(parsed);
          window.localStorage.setItem(SESSION_STORAGE_KEY, JSON.stringify(parsed));
        } else {
          window.localStorage.removeItem(SESSION_STORAGE_KEY);
        }
      } catch {
        window.localStorage.removeItem(SESSION_STORAGE_KEY);
      }
    }

    setReady(true);
  }, []);

  useEffect(() => {
    const handleUnauthorized = () => {
      setSession(null);
      window.localStorage.removeItem(SESSION_STORAGE_KEY);
    };

    window.addEventListener(AUTH_INVALID_EVENT, handleUnauthorized);
    return () => {
      window.removeEventListener(AUTH_INVALID_EVENT, handleUnauthorized);
    };
  }, []);

  const persistSession = useCallback((nextSession: Session | null) => {
    setSession(nextSession);

    if (nextSession) {
      window.localStorage.setItem(SESSION_STORAGE_KEY, JSON.stringify(nextSession));
    } else {
      window.localStorage.removeItem(SESSION_STORAGE_KEY);
    }
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      session,
      tenantMemberships: session?.tenantMemberships || [],
      activeTenant: session?.tenantMemberships.find((membership) => membership.tenantId === session.activeTenantId) || null,
      ready,
      async login(account: string, password: string) {
        const result = await loginRequest(account, password);
        const nextSession = normalizeSession({
          account: result.account,
          accessToken: result.accessToken,
          refreshToken: result.refreshToken,
          expiresAt: result.expiresAt,
          mustRotatePassword: result.mustRotatePassword,
          tenantMemberships: result.tenantMemberships,
          activeTenantId: result.activeTenantId
        });

        persistSession(nextSession);
        return nextSession;
      },
      async rotatePassword(password: string) {
        if (!session?.accessToken || !session.account.id) {
          throw new Error('invalid_access_token');
        }
        const account = await updateAccount(session.accessToken, session.account.id, {password});
        const nextSession: Session = {
          ...session,
          account,
          mustRotatePassword: account.mustRotatePassword
        };
        persistSession(nextSession);
        return nextSession;
      },
      switchTenant(tenantId: string | null) {
        if (!session) {
          return null;
        }
        if (tenantId && !session.tenantMemberships.some((membership) => membership.tenantId === tenantId)) {
          return session;
        }

        const nextSession: Session = {
          ...session,
          activeTenantId: tenantId
        };
        persistSession(nextSession);
        return nextSession;
      },
      async logout() {
        if (session?.accessToken) {
          try {
            await logoutRequest(session.accessToken);
          } catch {}
        }

        persistSession(null);
      }
    }),
    [persistSession, ready, session]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);

  if (!context) {
    throw new Error('useAuth must be used inside AuthProvider');
  }

  return context;
}
