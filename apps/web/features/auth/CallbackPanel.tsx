'use client';

import Link from 'next/link';
import { Feedback } from '../../components/ui/Feedback';
import { Surface } from '../../components/ui/Surface';
import { useAuthCallback } from './useAuthCallback';

export function CallbackPanel() {
  const { error } = useAuthCallback();

  return (
    <Surface className="auth-panel" aria-labelledby="callback-title" aria-busy={!error}>
      <p className="auth-eyebrow">ONE MOMENT</p>
      <h1 id="callback-title">Completing sign-in</h1>
      <div className="auth-actions">
        {error ? (
          <>
            <Feedback tone="error">{error}</Feedback>
            <Link className="vp-button vp-button--secondary" href="/">Return to sign-in</Link>
          </>
        ) : (
          <Feedback>Verifying your Google session…</Feedback>
        )}
      </div>
    </Surface>
  );
}
