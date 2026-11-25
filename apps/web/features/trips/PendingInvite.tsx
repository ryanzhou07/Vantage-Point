'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { Feedback } from '../../components/ui/Feedback';
import { postJSON, tripRequest } from '../receipts/api';

const KEY = 'vp-trip-invite';

// Reads an invite from the join link's fragment, keeps it for this tab only while the user signs in,
// then joins. The token is removed as soon as it is used.
export function PendingInvite({ fromLink = false }: { fromLink?: boolean }) {
  const [state, setState] = useState<'idle' | 'joining' | 'joined' | 'failed'>(fromLink ? 'joining' : 'idle');
  const [message, setMessage] = useState('');
  useEffect(() => {
    let token = '';
    try {
      if (fromLink && window.location.hash.length > 1) { token = window.location.hash.slice(1); sessionStorage.setItem(KEY, token); history.replaceState(null, '', window.location.pathname); }
      else token = sessionStorage.getItem(KEY) ?? '';
    } catch { /* storage unavailable: the link still works in this page load */ if (fromLink) token = window.location.hash.slice(1); }
    if (!token && !fromLink) return;
    (token ? tripRequest<{ name: string }>('/join', postJSON({ token })) : Promise.reject(new Error('This invite link is missing its code. Ask the owner for a new one.')))
      .then((t) => { setMessage(t.name); setState('joined'); })
      .catch((e) => { setMessage(e instanceof Error ? e.message : 'This invite could not be used.'); setState('failed'); })
      .finally(() => { try { sessionStorage.removeItem(KEY); } catch { /* ignore */ } });
  }, [fromLink]);
  if (state === 'idle') return null;
  if (state === 'joining') return <Feedback>Joining the trip…</Feedback>;
  if (state === 'failed') return <Feedback tone="error">{message}</Feedback>;
  return <Feedback>You joined “{message}”. <Link href="/dashboard/collections">Open collections</Link></Feedback>;
}
