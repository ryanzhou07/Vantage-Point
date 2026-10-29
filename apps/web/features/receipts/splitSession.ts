import { useEffect, useRef, useState } from 'react';
import { createClient } from '@supabase/supabase-js';

export type SessionState = 'ACTIVE' | 'EXPIRED' | 'INVALIDATED';
export type SessionView = { state: SessionState; server_time: string; session: { id: string; expires_at: string }; token?: string };
export type JoinView = {
  state: 'ACTIVE'; server_time: string; session: { id: string; expires_at: string };
  receipt: { merchant: string; currency: string }; participants: { id: string; name: string }[]; participant?: { id: string; name: string };
  items?: { id: string; name: string; amount_minor: number; claimed_by: string }[];
};

export class JoinError extends Error {
  constructor(public code: string, message: string) { super(message); }
}

const apiURL = () => process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';

async function publicPost(path: string, payload: object) {
  const response = await fetch(`${apiURL()}/api/v1/split/${path}`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'omit', cache: 'no-store', referrerPolicy: 'no-referrer',
    body: JSON.stringify(payload), signal: AbortSignal.timeout(15000),
  });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    const code = String(body.error?.code ?? 'UNAVAILABLE');
    throw new JoinError(code, response.status === 401 ? 'expired' : response.status === 409 && code === 'SPLIT_CLAIM_CONFLICT' ? 'conflict' : String(body.error?.message ?? 'unavailable'));
  }
  return body;
}

// Public participant calls. The token is only ever held in memory/URL fragment, never logged or stored.
// A typed name (with no participant id) joins as a new receipt-scoped participant.
export async function joinRequest(token: string, participantId?: string, name?: string): Promise<JoinView> {
  return publicPost('session', { token, participant_id: participantId, name }) as Promise<JoinView>;
}

export async function claimRequest(token: string, participantId: string, itemId: string, release: boolean): Promise<void> {
  await publicPost('claim', { token, participant_id: participantId, item_id: itemId, release });
}

export function formatMoney(minor: number): string {
  return `$${Math.floor(minor / 100)}.${String(minor % 100).padStart(2, '0')}`;
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
