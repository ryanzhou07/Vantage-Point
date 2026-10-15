'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { authClient } from '../../lib/auth/client';
import { loadProfile, signIn, signOut, type Profile } from '../../lib/auth/flows';

const unavailable = 'Sign-in is temporarily unavailable.';

export function useAuthSession() {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [busy, setBusy] = useState(true);
  const [error, setError] = useState('');
  const mounted = useRef(false);
  const pending = useRef(false);
  const loading = useRef(true);
  const controller = useRef<AbortController | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  const reload = useCallback(async (signal: AbortSignal) => {
    try {
      const nextProfile = await loadProfile(
        authClient(),
        process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080',
        signal,
      );

      if (mounted.current && !signal.aborted) {
        setProfile(nextProfile);
        setError('');
      }
    } catch (cause) {
      if (mounted.current && !signal.aborted) {
        setProfile(null);
        setError(cause instanceof Error ? cause.message : unavailable);
      }
    } finally {
      if (mounted.current && !signal.aborted) {
        loading.current = false;
        setBusy(pending.current);
      }
    }
  }, []);

  const refresh = useCallback(() => {
    clearTimeout(timer.current);
    controller.current?.abort();
    const next = new AbortController();
    controller.current = next;
    loading.current = true;
    setProfile(null);
    setBusy(true);
    setError('');

    // Defer provider reads outside Supabase's auth-event callback.
    timer.current = setTimeout(() => void reload(next.signal), 0);
  }, [reload]);

  useEffect(() => {
    mounted.current = true;
    const initial = new AbortController();
    controller.current = initial;
    timer.current = setTimeout(() => void reload(initial.signal), 0);
    let unsubscribe = () => {};

    try {
      const { data } = authClient().auth.onAuthStateChange(refresh);
      unsubscribe = () => data.subscription.unsubscribe();
    } catch {
      // The initial profile load presents missing configuration safely.
    }

    return () => {
      mounted.current = false;
      clearTimeout(timer.current);
      controller.current?.abort();
      unsubscribe();
    };
  }, [refresh, reload]);

  const action = useCallback(async (kind: 'in' | 'out') => {
    if (pending.current || !mounted.current) return;

    pending.current = true;
    setBusy(true);
    setError('');

    try {
      if (kind === 'in') {
        await signIn(authClient(), window.location.origin);
      } else {
        await signOut(authClient());
        if (mounted.current) setProfile(null);
      }
    } catch (cause) {
      if (mounted.current) {
        setError(cause instanceof Error ? cause.message : unavailable);
      }
    } finally {
      pending.current = false;
      if (mounted.current) setBusy(loading.current);
    }
  }, []);

  return {
    profile,
    busy,
    error,
    signIn: () => action('in'),
    signOut: () => action('out'),
    retry: () => { if (!pending.current) refresh(); },
  };
}
