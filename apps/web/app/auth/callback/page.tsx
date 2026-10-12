'use client';
import { useEffect, useState } from 'react';
import Link from 'next/link';
import { authClient } from '../../../lib/auth/client';
import { completeCallback } from '../../../lib/auth/flows';
// Share the one exchange across StrictMode's development effect replay.
let exchange: Promise<void> | undefined;
export default function Callback() {
  const [error, setError] = useState('');
  useEffect(() => {
    let active = true;
    exchange ??= Promise.resolve().then(() => completeCallback(authClient(), window.location.href));
    void exchange.then(() => { if (active) { window.history.replaceState(null, '', '/auth/callback'); window.location.replace('/'); } }, () => { if (active) { window.history.replaceState(null, '', '/auth/callback'); setError('Google sign-in could not complete. Please return and try again.'); } });
    return () => { active = false; };
  }, []);
  return <main className="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-6 p-6"><h1 className="text-2xl font-semibold">Completing sign-in</h1>{error ? <><p role="alert">{error}</p><Link className="min-h-12 rounded border p-3" href="/">Return to sign-in</Link></> : <p role="status">Verifying your Google session…</p>}</main>;
}
