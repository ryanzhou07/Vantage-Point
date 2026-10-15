'use client';

import Link from 'next/link';
import { useEffect, useRef, useState, type ReactNode } from 'react';
import { authClient } from '../../lib/auth/client';
import { loadProfile, signOut } from '../../lib/auth/flows';
import { createDashboardSession, type DashboardSession } from './session';
import { DashboardNavigation } from './dashboard-navigation';
import { DashboardProfileContext } from './profile-context';

const actionClass = 'inline-flex min-h-12 items-center justify-center rounded-xl border border-slate-300 px-5 py-3 text-sm font-semibold text-slate-800 hover:bg-slate-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-700 disabled:opacity-50';

export function DashboardShell({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<DashboardSession>({ status: 'loading' });
  const [signingOut, setSigningOut] = useState(false);
  const [actionError, setActionError] = useState('');
  const [navigationOpen, setNavigationOpen] = useState(false);
  const controller = useRef<ReturnType<typeof createDashboardSession> | null>(null);
  const pending = useRef(false);

  useEffect(() => {
    const sessionController = createDashboardSession(
      (signal) => loadProfile(authClient(), process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080', signal),
      (state) => {
        setSession(state);
        setActionError('');
        setNavigationOpen(false);
      },
    );
    controller.current = sessionController;
    let unsubscribe = () => {};

    try {
      const { data } = authClient().auth.onAuthStateChange(() => sessionController.refresh());
      unsubscribe = () => data.subscription.unsubscribe();
    } catch {
      // The bounded profile load reports missing local configuration safely.
    }
    sessionController.refresh();

    return () => {
      sessionController.dispose();
      unsubscribe();
      controller.current = null;
    };
  }, []);

  async function handleSignOut() {
    if (pending.current) return;

    pending.current = true;
    setSigningOut(true);
    setActionError('');

    try {
      await signOut(authClient());
      controller.current?.refresh();
    } catch {
      setActionError('Sign-out could not complete. Please try again.');
    } finally {
      pending.current = false;
      setSigningOut(false);
    }
  }

  if (session.status !== 'ready') {
    return (
      <main className="mx-auto flex min-h-screen max-w-lg flex-col justify-center gap-5 p-6 text-slate-900">
        <p className="text-sm font-semibold tracking-widest text-teal-800">VANTAGE-POINT</p>
        <h1 className="text-3xl font-semibold">Your dashboard</h1>
        {session.status === 'loading' && <p role="status">Loading your session…</p>}
        {session.status === 'signed-out' && <p>Sign in to access your dashboard.</p>}
        {session.status === 'error' && (
          <>
            <p role="alert">{session.message}</p>
            <button className={actionClass} onClick={() => controller.current?.refresh()}>Try again</button>
          </>
        )}
        {session.status !== 'loading' && <Link className={actionClass} href="/">Return to sign in</Link>}
      </main>
    );
  }

  return (
    <div className="min-h-screen bg-white text-slate-900">
      <a href="#dashboard-content" className="sr-only rounded-xl bg-white p-4 focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-50">Skip to content</a>
      <header className="border-b border-slate-200 bg-white">
        <div className="flex flex-wrap items-center justify-between gap-4 px-5 py-5 sm:px-8">
          <Link href="/dashboard" className="text-xl font-semibold tracking-tight focus-visible:outline-2 focus-visible:outline-teal-700">Vantage-Point<span className="ml-1 text-teal-700">.</span></Link>
          <div className="flex min-w-0 flex-wrap items-center gap-3">
            <p className="max-w-60 break-words text-sm text-slate-600">{session.profile.display_name || session.profile.email || 'Vantage-Point user'}</p>
            <button className={actionClass} disabled={signingOut} onClick={() => void handleSignOut()}>{signingOut ? 'Signing out…' : 'Sign out'}</button>
          </div>
        </div>
        {actionError && <p role="alert" className="mx-auto max-w-7xl px-5 pb-4 text-red-800">{actionError}</p>}
      </header>

      <div className="vp-dashboard-layout">
        <aside className="vp-dashboard-sidebar">
          <button className={`${actionClass} w-full lg:hidden`} aria-expanded={navigationOpen} aria-controls="dashboard-navigation" onClick={() => setNavigationOpen(!navigationOpen)}>Browse dashboard</button>
          <div id="dashboard-navigation" className={`vp-dashboard-nav-container ${navigationOpen ? 'block' : 'hidden'} lg:block`} onClick={(event) => {
            if ((event.target as HTMLElement).closest('a')) setNavigationOpen(false);
          }}>
            <DashboardNavigation />
          </div>
        </aside>
        <main id="dashboard-content" tabIndex={-1} className="vp-dashboard-content outline-none" key={session.profile.id}>
          <DashboardProfileContext.Provider value={session.profile}>
            {children}
          </DashboardProfileContext.Provider>
        </main>
      </div>
    </div>
  );
}
