import {ReactNode} from 'react';

import {AuthGate} from '@/components/auth-gate';
import {ConsoleShell} from '@/components/console-shell';
import {SetupGuard} from '@/components/setup-guard';

export default function ConsoleLayout({children}: {children: ReactNode}) {
  return (
    <SetupGuard>
      <AuthGate>
        <ConsoleShell>{children}</ConsoleShell>
      </AuthGate>
    </SetupGuard>
  );
}
