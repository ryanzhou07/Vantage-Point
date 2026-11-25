'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { Button } from '../../components/ui/Button';
import { Feedback } from '../../components/ui/Feedback';
import { postJSON, tripRequest } from '../receipts/api';
import { money } from '../receipts/types';
import type { MemberView } from './types';

// A signed-in trip member claims items as themselves; no link or timer, open until the owner finalizes.
export function TripReceipt({ tripId, receiptId, onBack }: { tripId: string; receiptId: string; onBack: () => void }) {
  const [view, setView] = useState<MemberView | null>(null);
  const [error, setError] = useState('');
  const mounted = useRef(true);
  const base = `/${tripId}/receipts/${receiptId}`;

  const load = useCallback(() => tripRequest<MemberView>(base)
    .then((next) => { if (mounted.current) { setView(next); setError(''); } })
    .catch((e) => { if (mounted.current) setError(e instanceof Error ? e.message : 'Could not load this receipt.'); }), [base]);

  useEffect(() => {
    mounted.current = true;
    void load();
    const poll = setInterval(() => { if (!document.hidden) void load(); }, 10000);
    return () => { mounted.current = false; clearInterval(poll); };
  }, [load]);

  async function claim(itemId: string, release: boolean) {
    setError('');
    const me = view?.participant?.id;
    if (me) setView((v) => v && { ...v, items: v.items.map((i) => i.id === itemId ? { ...i, claimed_by: release ? i.claimed_by.filter((x) => x !== me) : [...i.claimed_by, me] } : i) });
    try { await tripRequest(`${base}/claim`, postJSON({ item_id: itemId, release })); }
    catch (e) { setError(e instanceof Error ? e.message : 'Could not update your claim.'); }
    await load();
  }

  const open = view?.status === 'NEEDS_REVIEW';
  return (
    <section aria-label="Trip receipt">
      <Button variant="secondary" onClick={onBack}>← Back to trip</Button>
      <h2>{view?.receipt.merchant || 'Receipt'}</h2>
      {error && <Feedback tone="error">{error}</Feedback>}
      {!view && !error && <Feedback>Loading…</Feedback>}
      {view && !open && <Feedback>This receipt is finalized, so claims are locked.</Feedback>}
      {view && open && <p>Tap the items you had. Anyone can share an item; it splits evenly between everyone who taps it.</p>}
      {view && <ul style={{ listStyle: 'none', padding: 0, display: 'grid', gap: '.5rem' }}>
        {view.items.map((i) => {
          const mine = !!view.participant && i.claimed_by.includes(view.participant.id);
          const others = view.participants.filter((p) => p.id !== view.participant?.id && i.claimed_by.includes(p.id)).map((p) => p.name);
          return <li key={i.id}><Button variant={mine ? 'primary' : 'secondary'} aria-pressed={mine} disabled={!open} onClick={() => void claim(i.id, mine)}>
            {i.name} · {money(i.amount_minor)} · {mine ? 'Yours (tap to release)' : 'Tap to claim'}{others.length > 0 && ` · also ${others.join(', ')}`}
          </Button></li>;
        })}
      </ul>}
    </section>
  );
}
