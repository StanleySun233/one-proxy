import './globals.css';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'One Proxy Control Plane',
  description: 'Node orchestration, route rules, and certificate status for One Proxy.'
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
