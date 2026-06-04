'use client';

import {QueryClient, QueryClientProvider} from '@tanstack/react-query';
import {ThemeProvider} from 'next-themes';
import {ReactNode, useState} from 'react';
import {Toaster} from 'sonner';

import {AuthProvider} from '@/components/auth-provider';
import {ControlPlaneAPIError} from '@/lib/api';

export function Providers({children}: {children: ReactNode}) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            retry(failureCount, error) {
              if (error instanceof ControlPlaneAPIError && error.status === 401) {
                return false;
              }

              return failureCount < 1;
            }
          }
        }
      })
  );

  return (
    <ThemeProvider attribute="data-theme" defaultTheme="light" enableSystem={false}>
      <AuthProvider>
        <QueryClientProvider client={queryClient}>
          {children}
          <Toaster position="top-right" richColors />
        </QueryClientProvider>
      </AuthProvider>
    </ThemeProvider>
  );
}
