import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'One Proxy Control Plane',
  description: 'Node orchestration, route rules, and certificate status for One Proxy.',
  icons: {
    icon: '/favicon.svg',
    shortcut: '/favicon.svg',
    apple: '/favicon.svg'
  }
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html suppressHydrationWarning>
      <body>{children}</body>
    </html>
  );
}
