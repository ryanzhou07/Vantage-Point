'use client';
import { useCallback, useEffect, useRef, useState } from 'react';
import { authClient } from '../lib/auth/client';
import { loadProfile, signIn, signOut, type Profile } from '../lib/auth/flows';
export default function Home() {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [busy, setBusy] = useState(true);
  const [error, setError] = useState('');
  const pending = useRef(false);
  const reload = useCallback(async (signal: AbortSignal) => {
    try { const p = await loadProfile(authClient(), process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080', signal); if (!signal.aborted) { setProfile(p); setError(''); } }
    catch (e) { if (!signal.aborted) { setProfile(null); setError(e instanceof Error ? e.message : 'Sign-in is temporarily unavailable.'); } }
    finally { if (!signal.aborted) setBusy(false); }
  }, []);
  useEffect(() => {
    let controller = new AbortController();
    const initial = setTimeout(() => { void reload(controller.signal); }, 0);
    let unsubscribe = () => {};
    try { const { data } = authClient().auth.onAuthStateChange(() => { controller.abort(); controller = new AbortController(); setProfile(null); setBusy(true); setTimeout(() => { void reload(controller.signal); }, 0); }); unsubscribe = () => data.subscription.unsubscribe(); } catch { /* reload displays configuration error */ }
    return () => { clearTimeout(initial); controller.abort(); unsubscribe(); };
  }, [reload]);
  async function action(kind: 'in' | 'out') {
    if (pending.current) return;
    pending.current = true; setBusy(true); setError('');
    try { if (kind === 'in') await signIn(authClient(), window.location.origin); else { await signOut(authClient()); setProfile(null); } }
    catch (e) { setError(e instanceof Error ? e.message : 'Sign-in is temporarily unavailable.'); }
    finally { pending.current = false; setBusy(false); }
  }
  return <main className="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-6 p-6">
    <h1 className="text-3xl font-semibold">Vantage-Point</h1>
    {busy && <p role="status">Loading your session…</p>}
    {error && <p role="alert">{error}</p>}
    {profile ? <><p>Signed in as {profile.display_name || profile.email || 'Vantage-Point user'}.</p><button className="min-h-12 rounded bg-slate-900 p-3 text-white disabled:opacity-50" disabled={busy} onClick={() => void action('out')}>Sign out</button></> : <button className="min-h-12 rounded bg-slate-900 p-3 text-white disabled:opacity-50" disabled={busy} onClick={() => void action('in')}>Continue with Google</button>}
    {error && <button className="min-h-12 rounded border p-3" disabled={busy} onClick={() => { setBusy(true); void reload(AbortSignal.timeout(5000)); }}>Try again</button>}
  </main>;
}
