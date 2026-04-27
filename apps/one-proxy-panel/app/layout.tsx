import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'Control Plane',
  description: 'Node orchestration, route rules, and certificate status.'
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html suppressHydrationWarning>
      <body>{children}</body>
    </html>
  );
}
