'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { Button } from '../../components/ui/Button';
import { Feedback } from '../../components/ui/Feedback';
import { JoinError, formatRemaining, joinRequest, useSessionLive, type JoinView } from './splitSession';
import styles from './split.module.css';

type Phase = 'loading' | 'ready' | 'expired' | 'unavailable';

export function JoinSplit() {
  const token = useRef('');
  const mounted = useRef(true);
  const chosen = useRef<string | undefined>(undefined);
  const [phase, setPhase] = useState<Phase>('loading');
  const [view, setView] = useState<JoinView | null>(null);

  const load = useCallback(async () => {
    if (!token.current) { setPhase('expired'); return; }
    try {
      const next = await joinRequest(token.current, chosen.current);
      if (!mounted.current) return;
      setView(next); setPhase('ready');
    } catch (e) {
      if (!mounted.current) return;
      // Backend decides: 401 means expired, closed or invalid. Other failures keep the last view and retry.
      if (e instanceof JoinError && e.message === 'expired') setPhase('expired');
      else setPhase((p) => (p === 'ready' ? p : 'unavailable'));
    }
  }, []);
  useEffect(() => {
    mounted.current = true;
    token.current = window.location.hash.slice(1);
    void load();
    return () => { mounted.current = false; };
  }, [load]);

  const live = phase === 'ready';
  const remaining = useSessionLive(view?.session ?? null, view?.server_time ?? null, live, () => void load());

  return (
    <main className={styles.join}>
      <h1>Join the split</h1>
      {phase === 'loading' && <Feedback>Opening the split...</Feedback>}
      {phase === 'expired' && <Feedback tone="error">This link has expired or is not valid. Ask the receipt owner to show a new QR code.</Feedback>}
      {phase === 'unavailable' && <>
        <Feedback tone="error">Could not reach the split right now.</Feedback>
        <Button onClick={() => { setPhase('loading'); void load(); }}>Try again</Button>
      </>}
      {phase === 'ready' && view && <>
        <p className={styles.timer} role="timer" aria-live="off">{remaining > 0 ? `Open for ${formatRemaining(remaining)}` : 'Confirming status...'}</p>
        <p>{view.receipt.merchant || 'Shared receipt'}</p>
        {view.participant
          ? <Feedback>You are joined as {view.participant.name}.</Feedback>
          : <fieldset className={styles.names}>
            <legend>Pick your name</legend>
            {view.participants.length === 0 && <p>The owner has not added names yet. Ask them, then reopen this page.</p>}
            {view.participants.map((p) => (
              <Button key={p.id} variant="secondary" onClick={() => { chosen.current = p.id; void load(); }}>{p.name}</Button>
            ))}
          </fieldset>}
      </>}
    </main>
  );
}
