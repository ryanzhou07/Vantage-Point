'use client';

import { useEffect, useRef, useState } from 'react';
import { Button } from '../../components/ui/Button';
import { Feedback } from '../../components/ui/Feedback';
import { receiptRequest } from './api';
import { money } from './types';
import styles from './list.module.css';

export type SummaryRow = {
  id: string; merchant: string; date: string; status: string; currency: string; total_minor: number;
  amount_kind: 'DRAFT_PREVIEW' | 'FINALIZED' | 'NONE';
  shares: { participant_id: string; name: string; total_minor: number }[]; shares_total_minor: number;
};
type Page = { receipts: SummaryRow[]; next_cursor: string; outstanding_minor: number };

const STATUS: Record<string, string> = { DRAFT: 'Draft', PROCESSING: 'Processing', NEEDS_REVIEW: 'Needs review', PROCESSING_FAILED: 'Processing failed', FINALIZED: 'Finalized' };

export function ReceiptList() {
  const [rows, setRows] = useState<SummaryRow[]>([]);
  const [next, setNext] = useState('');
  const [outstanding, setOutstanding] = useState(0);
  const [phase, setPhase] = useState<'loading' | 'ready' | 'error'>('loading');
  const [more, setMore] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const mounted = useRef(true);

  useEffect(() => {
    const refresh = () => setAttempt((n) => n + 1);
    window.addEventListener('receipts:changed', refresh);
    return () => window.removeEventListener('receipts:changed', refresh);
  }, []);

  useEffect(() => {
    mounted.current = true;
    const controller = new AbortController();
    receiptRequest<Page>('/summary/10', { signal: controller.signal })
      .then((page) => { if (!mounted.current) return; setRows(page.receipts); setNext(page.next_cursor); setOutstanding(page.outstanding_minor); setPhase('ready'); })
      .catch(() => { if (mounted.current && !controller.signal.aborted) setPhase('error'); });
    return () => { mounted.current = false; controller.abort(); };
  }, [attempt]);

  async function loadMore() {
    setMore(true);
    try {
      const page = await receiptRequest<Page>(`/summary/10/${encodeURIComponent(next)}`);
      if (!mounted.current) return;
      setRows((current) => [...current, ...page.receipts]); setNext(page.next_cursor);
    } catch { if (mounted.current) setPhase('error'); }
    finally { if (mounted.current) setMore(false); }
  }

  return (
    <section className={styles.list} aria-labelledby="receipt-list-title">
      <h2 id="receipt-list-title">Receipt history</h2>
      {phase === 'loading' && <Feedback>Loading your receipts...</Feedback>}
      {phase === 'error' && <>
        <Feedback tone="error">Your receipts could not be loaded.</Feedback>
        <Button variant="secondary" onClick={() => { setPhase('loading'); setAttempt((n) => n + 1); }}>Try again</Button>
      </>}
      {phase === 'ready' && <>
        <p className={styles.outstanding}>Outstanding from finalized receipts: <strong>{money(outstanding)}</strong></p>
        <p className={styles.note}>Draft previews are only estimates and are not counted as owed.</p>
        {rows.length === 0 && <p>No receipts yet. Upload one on the Receipt splitting tab to start a split.</p>}
        <ul className={styles.rows}>
          {rows.map((row) => (
            <li key={row.id} className={styles.row}>
              <div className={styles.head}>
                <strong>{row.merchant || 'Unnamed receipt'}</strong>
                <span className={styles.status}>{STATUS[row.status] ?? row.status}</span>
              </div>
              <p className={styles.meta}>{row.date || 'No date'} · Receipt total {money(row.total_minor)}</p>
              {row.amount_kind === 'DRAFT_PREVIEW' && <span className={`${styles.badge} ${styles.draft}`}>Draft preview, not owed</span>}
              {row.amount_kind === 'FINALIZED' && <span className={`${styles.badge} ${styles.final}`}>Finalized</span>}
              {row.shares.length > 0 && <details>
                <summary>Participant amounts ({row.shares.length})</summary>
                <ul className={styles.shares}>
                  {row.shares.map((s) => <li key={s.participant_id}><span>{s.name || 'Unnamed'}</span><span>{money(s.total_minor)}</span></li>)}
                  <li className={styles.sum}><span>Total</span><span>{money(row.shares_total_minor)}</span></li>
                </ul>
              </details>}
            </li>
          ))}
        </ul>
        {next && <Button variant="secondary" disabled={more} onClick={() => void loadMore()}>{more ? 'Loading...' : 'Load more'}</Button>}
      </>}
    </section>
  );
}
