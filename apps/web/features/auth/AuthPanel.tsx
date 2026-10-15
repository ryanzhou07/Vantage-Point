'use client';

import { Button } from '../../components/ui/Button';
import { Feedback } from '../../components/ui/Feedback';
import { Surface } from '../../components/ui/Surface';
import { GoogleMark } from './GoogleMark';
import { useAuthSession } from './useAuthSession';

export function AuthPanel() {
  const { profile, busy, error, signIn, signOut, retry } = useAuthSession();

  return (
    <Surface className="auth-panel" aria-labelledby="auth-title" aria-busy={busy}>
      <p className="auth-eyebrow">YOUR VANTAGE POINT</p>
      <h1 id="auth-title">{profile ? 'Welcome back.' : 'A clearer view starts here.'}</h1>
      <p className="auth-description">
        {profile
          ? `Signed in as ${profile.display_name || profile.email || 'Vantage-Point user'}.`
          : 'Sign in to your Vantage-Point account with Google.'}
      </p>

      <div className="auth-actions">
        {busy && <Feedback>Loading your session…</Feedback>}
        {error && <Feedback tone="error">{error}</Feedback>}

        {profile ? (
          <Button disabled={busy} onClick={() => void signOut()}>Sign out</Button>
        ) : (
          <Button variant="secondary" disabled={busy} onClick={() => void signIn()}>
            <GoogleMark />
            Continue with Google
          </Button>
        )}

        {error && (
          <Button variant="secondary" disabled={busy} onClick={retry}>Try again</Button>
        )}
      </div>
    </Surface>
  );
}
