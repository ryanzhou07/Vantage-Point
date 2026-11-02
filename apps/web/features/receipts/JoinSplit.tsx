'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { Button } from '../../components/ui/Button';
import { Feedback } from '../../components/ui/Feedback';
import { JoinError, claimRequest, formatMoney, formatRemaining, joinRequest, useSessionLive, type JoinView } from './splitSession';
import styles from './split.module.css';

type Phase = 'loading' | 'ready' | 'expired' | 'unavailable';

export function JoinSplit() {
  const token = useRef('');
  const mounted = useRef(true);
  const chosen = useRef<string | undefined>(undefined);
  const [phase, setPhase] = useState<Phase>('loading');
  const [view, setView] = useState<JoinView | null>(null);
  const [name, setName] = useState('');
  const [problem, setProblem] = useState('');

  const load = useCallback(async (typedName?: string) => {
    if (!token.current) { setPhase('expired'); return; }
    try {
      let next = await joinRequest(token.current, chosen.current, typedName);
      if (!mounted.current) return;
      if (!next.participant && !chosen.current) {
        // Reload in the same tab: reuse the earlier identity for this session instead of joining twice.
        let saved: string | null = null;
        try { saved = sessionStorage.getItem(`vp-split-${next.session.id}`); } catch { /* ignore */ }
        if (saved) next = await joinRequest(token.current, saved).catch(() => next);
      }
      if (next.participant) {
        chosen.current = next.participant.id;
        try { sessionStorage.setItem(`vp-split-${next.session.id}`, next.participant.id); } catch { /* per-tab convenience only */ }
      }
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

  async function claim(itemId: string, release: boolean) {
    if (!view?.participant) return;
    setProblem('');
    try { await claimRequest(token.current, view.participant.id, itemId, release); }
    catch (e) {
      if (e instanceof JoinError && e.message === 'conflict') setProblem('Someone else already claimed that item.');
      else if (e instanceof JoinError && e.message === 'expired') { setPhase('expired'); return; }
      else setProblem(e instanceof Error && e.message !== 'unavailable' ? e.message : 'Could not update your claim. Try again.');
    }
    await load();
  }

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
          ? <>
            <Feedback>You are joined as {view.participant.name}.</Feedback>
            {problem && <Feedback tone="error">{problem}</Feedback>}
            <fieldset className={styles.names}>
              <legend>Tap the items you had</legend>
              {(view.items ?? []).length === 0 && <p>No items yet. Ask the owner to finish reviewing the receipt.</p>}
              {(view.items ?? []).map((i) => {
                const mine = i.claimed_by === view.participant!.id;
                const holder = view.participants.find((p) => p.id === i.claimed_by)?.name;
                return (
                  <Button key={i.id} variant={mine ? 'primary' : 'secondary'} aria-pressed={mine} disabled={!!i.claimed_by && !mine} onClick={() => void claim(i.id, mine)}>
                    {i.name} · {formatMoney(i.amount_minor)} · {mine ? 'Yours (tap to release)' : holder ? `Taken by ${holder}` : 'Free'}
                  </Button>
                );
              })}
            </fieldset>
            <p>Your items: {formatMoney((view.items ?? []).filter((i) => i.claimed_by === view.participant!.id).reduce((n, i) => n + i.amount_minor, 0))} before tax and tip.</p>
          </>
          : <>
            <form className={styles.names} onSubmit={(e) => { e.preventDefault(); if (name.trim()) void load(name.trim()); }}>
              <label className={styles.linkLabel}>Your name<input value={name} maxLength={100} onChange={(e) => setName(e.target.value)} /></label>
              <Button type="submit" disabled={!name.trim()}>Join</Button>
            </form>
            {view.participants.length > 0 && <fieldset className={styles.names}>
              <legend>Or pick a name the owner added</legend>
              {view.participants.map((p) => (
                <Button key={p.id} variant="secondary" onClick={() => { chosen.current = p.id; void load(); }}>{p.name}</Button>
              ))}
            </fieldset>}
          </>}
      </>}
    </main>
  );
}
