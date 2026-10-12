import type { ReactNode } from 'react';
import { Brand } from '../../components/ui/Brand';

export function AuthFrame({ children }: { children: ReactNode }) {
  return (
    <main className="auth-frame">
      <div className="auth-content">
        <Brand />
        {children}
        <p className="auth-footnote">Your Google password stays with Google.</p>
      </div>
    </main>
  );
}
