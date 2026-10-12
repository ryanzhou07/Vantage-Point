'use client';

import { useEffect, useState } from 'react';
import { authClient } from '../../lib/auth/client';
import { completeCallback } from '../../lib/auth/flows';

// Share the one PKCE exchange across StrictMode's development effect replay.
let exchange: Promise<void> | undefined;

export function useAuthCallback() {
  const [error, setError] = useState('');

  useEffect(() => {
    let active = true;
    exchange ??= Promise.resolve().then(() => completeCallback(authClient(), window.location.href));

    void exchange.then(
      () => {
        if (!active) return;
        window.history.replaceState(null, '', '/auth/callback');
        window.location.replace('/');
      },
      () => {
        if (!active) return;
        window.history.replaceState(null, '', '/auth/callback');
        setError('Google sign-in could not complete. Please return and try again.');
      },
    );

    return () => { active = false; };
  }, []);

  return { error };
}
