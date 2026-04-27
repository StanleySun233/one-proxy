'use client';

import {createContext, ReactNode, useContext, useEffect, useMemo, useState} from 'react';

import {AUTH_INVALID_EVENT, login as loginRequest, logout as logoutRequest, Session, SESSION_STORAGE_KEY, updateAccount} from '@/lib/control-plane-api';

type AuthContextValue = {
  session: Session | null;
  ready: boolean;
  login: (account: string, password: string) => Promise<Session>;
  rotatePassword: (password: string) => Promise<Session>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({children}: {children: ReactNode}) {
  const [session, setSession] = useState<Session | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    const stored = window.localStorage.getItem(SESSION_STORAGE_KEY);

    if (stored) {
      try {
        const parsed = JSON.parse(stored) as Session;
        if (Date.parse(parsed.expiresAt) > Date.now()) {
          setSession(parsed);
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

  const value = useMemo<AuthContextValue>(
    () => ({
      session,
      ready,
      async login(account: string, password: string) {
        const result = await loginRequest(account, password);
        const nextSession: Session = {
          account: result.account,
          accessToken: result.accessToken,
          refreshToken: result.refreshToken,
          expiresAt: result.expiresAt,
          mustRotatePassword: result.mustRotatePassword
        };

        setSession(nextSession);
        window.localStorage.setItem(SESSION_STORAGE_KEY, JSON.stringify(nextSession));
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
        setSession(nextSession);
        window.localStorage.setItem(SESSION_STORAGE_KEY, JSON.stringify(nextSession));
        return nextSession;
      },
      async logout() {
        if (session?.accessToken) {
          try {
            await logoutRequest(session.accessToken);
          } catch {}
        }

        setSession(null);
        window.localStorage.removeItem(SESSION_STORAGE_KEY);
      }
    }),
    [ready, session]
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
