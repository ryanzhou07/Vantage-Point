'use client';

import { useEffect, useRef, useState } from 'react';
import { Button } from '../../components/ui/Button';
import { Feedback } from '../../components/ui/Feedback';
import { postJSON, receivablesRequest } from './api';
import { money } from './types';
import styles from './list.module.css';

type Row = {
  id: string; receipt_id: string; merchant: string; receipt_date: string; participant_name: string;
  amount_owed_minor: number; status: 'OWED' | 'PAID'; paid_at: string | null;
};

// Manual receivable tracking: the backend decides amounts and totals, this only shows them and flips PAID/OWED.
export function ReceivablesPanel() {
  const [rows, setRows] = useState<Row[]>([]);
  const [outstanding, setOutstanding] = useState(0);
  const [phase, setPhase] = useState<'loading' | 'ready' | 'error'>('loading');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState('');
  const mounted = useRef(true);

  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    mounted.current = true;
    const refresh = () => setAttempt((n) => n + 1);
    window.addEventListener('receipts:changed', refresh);
    return () => { mounted.current = false; window.removeEventListener('receipts:changed', refresh); };
  }, []);
  useEffect(() => {
    const controller = new AbortController();
    receivablesRequest<{ receivables: Row[]; outstanding_minor: number }>('', { signal: controller.signal })
      .then((body) => { if (!mounted.current) return; setRows(body.receivables); setOutstanding(body.outstanding_minor); setPhase('ready'); })
      .catch(() => { if (mounted.current && !controller.signal.aborted) setPhase('error'); });
    return () => controller.abort();
  }, [attempt]);

  async function toggle(row: Row) {
    setBusy(row.id); setError('');
    try {
      await receivablesRequest(`/${row.id}/paid`, postJSON({ paid: row.status === 'OWED' }));
      setAttempt((n) => n + 1);
    } catch (e) { if (mounted.current) setError(e instanceof Error ? e.message : 'Could not update that payment. Try again.'); }
    finally { if (mounted.current) setBusy(''); }
  }

  return (
    <section className={styles.list} aria-labelledby="receivables-title">
      <h2 id="receivables-title">Money owed to you</h2>
      {phase === 'loading' && <Feedback>Loading what people owe you...</Feedback>}
      {phase === 'error' && <><Feedback tone="error">Could not load receivables.</Feedback><Button variant="secondary" onClick={() => { setPhase('loading'); setAttempt((n) => n + 1); }}>Try again</Button></>}
      {error && <Feedback tone="error">{error}</Feedback>}
      {phase === 'ready' && <>
        <p className={styles.outstanding}>Still owed: <strong>{money(outstanding)}</strong></p>
        {rows.length === 0 && <p>Nothing yet. Finalize a split to create what people owe you.</p>}
        <ul className={styles.rows}>
          {rows.map((row) => (
            <li key={row.id} className={styles.row}>
              <div className={styles.head}><strong>{row.participant_name}</strong><span className={styles.status}>{row.status === 'PAID' ? 'Paid' : 'Owes you'}</span></div>
              <p className={styles.meta}>{money(row.amount_owed_minor)} · {row.merchant || 'Receipt'} · {row.receipt_date || 'No date'}{row.paid_at ? ` · paid ${row.paid_at.slice(0, 10)}` : ''}</p>
              <Button variant="secondary" disabled={busy === row.id} onClick={() => void toggle(row)}>{row.status === 'OWED' ? 'Mark paid' : 'Mark unpaid'}</Button>
            </li>
          ))}
        </ul>
      </>}
    </section>
  );
}
