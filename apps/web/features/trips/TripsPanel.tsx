'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { Button } from '../../components/ui/Button';
import { Feedback } from '../../components/ui/Feedback';
import { Qr } from '../receipts/SplitSessionPanel';
import { postJSON, tripRequest } from '../receipts/api';
import { money } from '../receipts/types';
import { useDashboardProfile } from '../../components/dashboard/profile-context';
import { TripReceipt } from './TripReceipt';
import type { TripDetail, TripRow } from './types';
import styles from './trips.module.css';

export function TripsPanel() {
  const profile = useDashboardProfile();
  const [trips, setTrips] = useState<TripRow[] | null>(null);
  const [open, setOpen] = useState<string | null>(null);
  const [receipt, setReceipt] = useState<string | null>(null);
  const [detail, setDetail] = useState<TripDetail | null>(null);
  const [name, setName] = useState('');
  const [invite, setInvite] = useState('');
  const [error, setError] = useState('');
  const mounted = useRef(true);

  const run = useCallback((job: () => Promise<void>) => job().then(() => { if (mounted.current) setError(''); })
    .catch((e) => { if (mounted.current) setError(e instanceof Error ? e.message : 'Something went wrong. Try again.'); }), []);
  const loadList = useCallback(() => run(() => tripRequest<{ trips: TripRow[] }>('').then((b) => { if (mounted.current) setTrips(b.trips ?? []); })), [run]);
  const loadDetail = useCallback((id: string) => run(() => tripRequest<TripDetail>(`/${id}`).then((d) => { if (mounted.current) setDetail(d); })), [run]);
  const openTrip = (id: string | null) => { setDetail(null); setInvite(''); setOpen(id); };

  useEffect(() => { mounted.current = true; void loadList(); return () => { mounted.current = false; }; }, [loadList]);
  useEffect(() => { if (open) void loadDetail(open); }, [open, loadDetail]);

  if (open && receipt) return <TripReceipt tripId={open} receiptId={receipt} onBack={() => { setReceipt(null); void loadDetail(open); }} />;

  if (open) return (
    <section aria-label="Collection" className={styles.panel}>
      <Button variant="secondary" onClick={() => { openTrip(null); void loadList(); }}>← All collections</Button>
      {error && <Feedback tone="error">{error}</Feedback>}
      {!detail && !error && <Feedback>Loading…</Feedback>}
      {detail && <>
        <h2>{detail.name}</h2>
        <h3>Receipts ({detail.receipts.length})</h3>
        {detail.receipts.length === 0 && <p>No receipts here yet. Open a receipt on the Receipts tab and pick this collection. Drafts show up once confirmed.</p>}
        <ul className={styles.list}>{detail.receipts.map((r) => <li key={r.id}>
          <span><strong>{r.merchant || 'Receipt'}</strong> · {r.date || 'No date'} · {money(r.total_minor)} · {r.status === 'FINALIZED' ? 'Finalized' : 'Open for claims'}{r.mine && ' · yours'}</span>
          <Button variant="secondary" onClick={() => setReceipt(r.id)}>{r.status === 'FINALIZED' ? 'View' : 'Claim items'}</Button>
        </li>)}</ul>
        <h3>Members ({detail.members.length})</h3>
        <ul className={styles.list}>{detail.members.map((m) => <li key={m.user_id}>
          <span>{m.name}{m.is_owner && ' (owner)'}{m.user_id === profile?.id && ' (you)'}</span>
          {detail.is_owner && !m.is_owner && <Button variant="secondary" onClick={() => void run(async () => { await tripRequest(`/${detail.id}/members/${m.user_id}`, { method: 'DELETE' }); await loadDetail(detail.id); })}>Remove</Button>}
        </li>)}</ul>
        {detail.is_owner && <>
          <Button onClick={() => void run(async () => { const t = await tripRequest<{ token: string }>(`/${detail.id}/invite`, postJSON({})); setInvite(`${window.location.origin}/dashboard/collections/join#${t.token}`); })}>Create invite link (valid 7 days)</Button>
          {invite && <><p>Members must sign in to join. Making a new link cancels the old one.</p><Qr text={invite} /><p className={styles.link}>{invite}</p>
            <Button variant="secondary" onClick={() => void navigator.clipboard?.writeText(invite)}>Copy link</Button></>}
        </>}
        {detail.is_owner
          ? <Button variant="secondary" onClick={() => { if (window.confirm('Delete this collection? Receipts stay, but are no longer filed here and members lose access.')) void run(async () => { await tripRequest(`/${detail.id}`, { method: 'DELETE' }); openTrip(null); await loadList(); }); }}>Delete collection</Button>
          : <Button variant="secondary" onClick={() => { if (window.confirm('Leave this trip?')) void run(async () => { await tripRequest(`/${detail.id}/members/me`, { method: 'DELETE' }); openTrip(null); await loadList(); }); }}>Leave trip</Button>}
      </>}
    </section>
  );

  return (
    <section aria-label="Collections" className={styles.panel}>
      <h2>Collections & trips</h2>
      <p>Group receipts like “Trip A”. Invite others and it becomes a trip: only signed-in members can see and claim its receipts.</p>
      {error && <Feedback tone="error">{error}</Feedback>}
      <form className={styles.row} onSubmit={(e) => { e.preventDefault(); const n = name.trim(); if (n) void run(async () => { await tripRequest('', postJSON({ name: n })); setName(''); await loadList(); }); }}>
        <label>New collection<input value={name} maxLength={80} placeholder="e.g. Tokyo trip" onChange={(e) => setName(e.target.value)} /></label>
        <Button type="submit" disabled={!name.trim()}>Create</Button>
      </form>
      {trips === null && !error && <Feedback>Loading…</Feedback>}
      {trips?.length === 0 && <p>No collections yet. Create one above, then file receipts into it from the Receipts tab.</p>}
      <ul className={styles.grid}>{trips?.map((t) => <li key={t.id}><button type="button" className={styles.card} onClick={() => openTrip(t.id)}>
        <strong>{t.name}</strong><span>{t.receipts} receipts · {money(t.total_minor)}</span>
        <span className={styles.badge}>{t.members > 1 ? `Trip · ${t.members} members` : 'Private'}</span>
      </button></li>)}</ul>
    </section>
  );
}
