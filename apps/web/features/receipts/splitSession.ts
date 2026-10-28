import { useEffect, useRef, useState } from 'react';
import { createClient } from '@supabase/supabase-js';

export type SessionState = 'ACTIVE' | 'EXPIRED' | 'INVALIDATED';
export type SessionView = { state: SessionState; server_time: string; session: { id: string; expires_at: string }; token?: string };
export type JoinView = {
  state: 'ACTIVE'; server_time: string; session: { id: string; expires_at: string };
  receipt: { merchant: string; currency: string }; participants: { id: string; name: string }[]; participant?: { id: string; name: string };
};

export class JoinError extends Error {
  constructor(public code: string, message: string) { super(message); }
}

// Public participant call. The token is only ever held in memory/URL fragment, never logged or stored.
export async function joinRequest(token: string, participantId?: string): Promise<JoinView> {
  const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'}/api/v1/split/session`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'omit', cache: 'no-store', referrerPolicy: 'no-referrer',
    body: JSON.stringify({ token, participant_id: participantId }), signal: AbortSignal.timeout(15000),
  });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new JoinError(String(body.error?.code ?? 'UNAVAILABLE'), response.status === 401 ? 'expired' : 'unavailable');
  return body as JoinView;
}

export function formatRemaining(ms: number): string {
  const total = Math.max(0, Math.ceil(ms / 1000));
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, '0')}`;
}

const POLL_MS = 15000;
const CONFIRM_MS = 3000;

/**
 * Display-only countdown plus refresh triggers. Realtime is a payload-free notification; on any
 * channel problem the hook falls back to bounded polling. Expiry is decided only by refetch results.
 */
export function useSessionLive(session: { id: string; expires_at: string } | null, serverTime: string | null, active: boolean, refetch: () => void): number {
  const [now, setNow] = useState(() => Date.now());
  const [offset, setOffset] = useState(0);
  const refetchRef = useRef(refetch);
  useEffect(() => { refetchRef.current = refetch; });
  const [seen, setSeen] = useState<string | null>(null);
  if (serverTime && seen !== serverTime) { setSeen(serverTime); setOffset(Date.parse(serverTime) - now); } // skew vs. the server clock; display only
  const id = session?.id;
  useEffect(() => {
    if (!id || !active) return;
    const tick = setInterval(() => setNow(Date.now()), 1000);
    let poll: ReturnType<typeof setInterval> | undefined;
    const startPolling = () => { poll ??= setInterval(() => refetchRef.current(), POLL_MS); };
    let channel: { unsubscribe: () => unknown } | undefined;
    let client: ReturnType<typeof createClient> | undefined;
    try {
      const url = process.env.NEXT_PUBLIC_SUPABASE_URL;
      const key = process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY;
      if (!url || !key) throw new Error('realtime not configured');
      client = createClient(url, key, { auth: { persistSession: false, autoRefreshToken: false, detectSessionInUrl: false } });
      const live = client.channel(`split:${id}`).on('broadcast', { event: 'session' }, () => refetchRef.current());
      live.subscribe((status: string) => { if (status === 'CHANNEL_ERROR' || status === 'TIMED_OUT' || status === 'CLOSED') startPolling(); });
      channel = live;
    } catch { startPolling(); }
    return () => { clearInterval(tick); if (poll) clearInterval(poll); void channel?.unsubscribe(); void client?.removeAllChannels(); };
  }, [id, active]);
  const remaining = session ? Date.parse(session.expires_at) - (now + offset) : 0;
  const past = active && remaining <= 0;
  useEffect(() => {
    if (!past) return;
    refetchRef.current();
    const again = setInterval(() => refetchRef.current(), CONFIRM_MS);
    return () => clearInterval(again);
  }, [past]);
  return remaining;
}
