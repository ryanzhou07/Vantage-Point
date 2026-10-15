import type { Profile } from '../../lib/auth/flows';

export type DashboardSession =
  | { status: 'loading' }
  | { status: 'signed-out' }
  | { status: 'error'; message: string }
  | { status: 'ready'; profile: Profile };

// A controller belongs to one mounted shell. It never caches a profile or token.
export function createDashboardSession(
  load: (signal: AbortSignal) => Promise<Profile | null>,
  publish: (state: DashboardSession) => void,
) {
  let request: AbortController | undefined;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let disposed = false;

  function refresh() {
    if (disposed) return;

    request?.abort();
    clearTimeout(timer);
    const current = new AbortController();
    request = current;
    publish({ status: 'loading' });

    // Leave the Supabase auth event callback before invoking SDK session methods.
    timer = setTimeout(async () => {
      try {
        const profile = await load(current.signal);
        if (disposed || current.signal.aborted) return;

        publish(profile ? { status: 'ready', profile } : { status: 'signed-out' });
      } catch {
        if (!disposed && !current.signal.aborted) {
          publish({ status: 'error', message: 'Your profile could not be loaded. Try again, or return to sign in.' });
        }
      }
    }, 0);
  }

  return {
    refresh,
    dispose() {
      disposed = true;
      request?.abort();
      clearTimeout(timer);
    },
  };
}
