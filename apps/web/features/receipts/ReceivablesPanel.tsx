'use client';

import { useEffect, useRef, useState } from 'react';
import { Button } from '../../components/ui/Button';
import { Feedback } from '../../components/ui/Feedback';
import { postJSON, receivablesRequest } from './api';
import { money } from './types';
import styles from './list.module.css';

type Row = {
  id: string; receipt_id: string; merchant: string; receipt_date: string; participant_name: string;
  amount_owed_minor: number; status: 'OWED' | 'PAID'; paid_at: string | null; payment_method?: 'VENMO' | 'ZELLE' | null;
};
type Page = { receivables: Row[]; outstanding_minor: number; next_cursor: string };
type Filters = { q: string; status: '' | 'OWED' | 'PAID'; from: string; to: string };

const NONE: Filters = { q: '', status: '', from: '', to: '' };
const METHOD = { VENMO: 'Venmo', ZELLE: 'Zelle' } as const;
const STATUS_LABEL = { '': 'All', OWED: 'Owes you', PAID: 'Paid' } as const;

function fromURL(): Filters {
  const p = new URLSearchParams(typeof window === 'undefined' ? '' : window.location.search);
  const status = p.get('status');
  return { q: p.get('q') ?? '', status: status === 'OWED' || status === 'PAID' ? status : '', from: p.get('from') ?? '', to: p.get('to') ?? '' };
}
const same = (a: Filters, b: Filters) => (a.q.trim().length >= 2 ? a.q.trim() : '') === (b.q.trim().length >= 2 ? b.q.trim() : '') && a.status === b.status && a.from === b.from && a.to === b.to;
// Only short filter text goes in the URL; cursors never do.
function query(f: Filters, cursor = ''): string {
  const p = new URLSearchParams();
  if (f.q.trim().length >= 2) p.set('q', f.q.trim());
  if (f.status) p.set('status', f.status);
  if (f.from) p.set('from', f.from);
  if (f.to) p.set('to', f.to);
  p.set('limit', '25');
  if (cursor) p.set('cursor', cursor);
  return `?${p.toString()}`;
}

// Receivables with filters and paging: the backend decides rows, totals and order; this only shows them and flips PAID/OWED.
export function ReceivablesPanel() {
  const [input, setInput] = useState<Filters>(fromURL);
  const [applied, setApplied] = useState<Filters>(fromURL);
  const [rows, setRows] = useState<Row[]>([]);
  const [outstanding, setOutstanding] = useState(0);
  const [cursor, setCursor] = useState('');
  const [phase, setPhase] = useState<'loading' | 'ready' | 'error'>('loading');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState('');
  const [more, setMore] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const mounted = useRef(true);
  const focusId = useRef('');
  const rowRefs = useRef<Record<string, HTMLLIElement | null>>({});

  const rangeError = input.from && input.to && input.to < input.from ? 'The end date must be on or after the start date.' : '';
  const shortSearch = input.q.trim().length === 1;
  const active = Object.entries(applied).some(([key, value]) => (key === 'q' ? value.trim().length >= 2 : !!value));

  useEffect(() => {
    mounted.current = true;
    const refresh = () => setAttempt((n) => n + 1);
    window.addEventListener('receipts:changed', refresh);
    return () => { mounted.current = false; window.removeEventListener('receipts:changed', refresh); };
  }, []);
  // Text is debounced; dates and status apply at once. An invalid range sends nothing.
  useEffect(() => {
    if (rangeError || same(input, applied)) return;
    const timer = setTimeout(() => setApplied(input), input.q === applied.q ? 0 : 300);
    return () => clearTimeout(timer);
  }, [input, rangeError, applied]);
  useEffect(() => {
    const controller = new AbortController();
    receivablesRequest<Page>(query(applied), { signal: controller.signal })
      .then((body) => {
        if (!mounted.current || controller.signal.aborted) return;
        setRows(body.receivables); setOutstanding(body.outstanding_minor); setCursor(body.next_cursor); setPhase('ready');
        const url = `${window.location.pathname}${query(applied).replace(/&?limit=25/, '').replace(/^\?$/, '')}`;
        window.history.replaceState(null, '', url);
      })
      .catch(() => { if (mounted.current && !controller.signal.aborted) setPhase('error'); });
    return () => controller.abort();
  }, [applied, attempt]);
  useEffect(() => {
    if (focusId.current) { rowRefs.current[focusId.current]?.focus(); focusId.current = ''; }
  }, [rows]);

  async function loadMore() {
    setMore(true); setError('');
    try {
      const body = await receivablesRequest<Page>(query(applied, cursor));
      if (!mounted.current) return;
      const seen = new Set(rows.map((r) => r.id));
      const fresh = body.receivables.filter((r) => !seen.has(r.id));
      focusId.current = fresh[0]?.id ?? '';
      setRows([...rows, ...fresh]); setOutstanding(body.outstanding_minor); setCursor(body.next_cursor);
    } catch (e) { if (mounted.current) setError(e instanceof Error ? e.message : 'Could not load more. Try again.'); }
    finally { if (mounted.current) setMore(false); }
  }
  async function toggle(row: Row) {
    setBusy(row.id); setError('');
    try {
      await receivablesRequest(`/${row.id}/paid`, postJSON({ paid: row.status === 'OWED' }));
      setAttempt((n) => n + 1);
    } catch (e) { if (mounted.current) setError(e instanceof Error ? e.message : 'Could not update that payment. Try again.'); }
    finally { if (mounted.current) setBusy(''); }
  }
  const clear = () => { setInput(NONE); setApplied(NONE); };
  const chips: [string, () => void][] = [];
  if (applied.q.trim().length >= 2) chips.push([`Search: ${applied.q.trim()}`, () => { setInput({ ...input, q: '' }); setApplied({ ...applied, q: '' }); }]);
  if (applied.status) chips.push([`Status: ${STATUS_LABEL[applied.status]}`, () => { setInput({ ...input, status: '' }); setApplied({ ...applied, status: '' }); }]);
  if (applied.from) chips.push([`From ${applied.from}`, () => { setInput({ ...input, from: '' }); setApplied({ ...applied, from: '' }); }]);
  if (applied.to) chips.push([`To ${applied.to}`, () => { setInput({ ...input, to: '' }); setApplied({ ...applied, to: '' }); }]);

  return (
    <section className={styles.list} aria-labelledby="receivables-title">
      <h2 id="receivables-title">Receivables · money owed to you</h2>
      <form role="search" className={styles.filters} onSubmit={(e) => e.preventDefault()}>
        <label className={styles.field}>Search name or merchant
          <input type="search" value={input.q} maxLength={64} onChange={(e) => setInput({ ...input, q: e.target.value })} />
        </label>
        {shortSearch && <p className={styles.note}>Type at least 2 characters to search.</p>}
        <label className={styles.field}>From
          <input type="date" value={input.from} onChange={(e) => setInput({ ...input, from: e.target.value })} />
        </label>
        <label className={styles.field}>To
          <input type="date" value={input.to} onChange={(e) => setInput({ ...input, to: e.target.value })} />
        </label>
        {rangeError && <p role="alert" className={styles.fieldError}>{rangeError}</p>}
        <fieldset className={styles.statusGroup}>
          <legend>Status</legend>
          {(['', 'OWED', 'PAID'] as const).map((value) => (
            <Button key={value || 'all'} variant={input.status === value ? 'primary' : 'secondary'} aria-pressed={input.status === value} onClick={() => setInput({ ...input, status: value })}>{STATUS_LABEL[value]}</Button>
          ))}
        </fieldset>
      </form>
      {chips.length > 0 && <ul className={styles.chips} aria-label="Active filters">
        {chips.map(([label, remove]) => <li key={label}><button type="button" className={styles.chip} aria-label={`Remove filter: ${label}`} onClick={remove}>{label} ×</button></li>)}
        <li><Button variant="secondary" onClick={clear}>Clear all</Button></li>
      </ul>}
      {phase === 'loading' && <Feedback>Loading what people owe you...</Feedback>}
      {phase === 'error' && <><Feedback tone="error">Could not load receivables.</Feedback><Button variant="secondary" onClick={() => { setPhase('loading'); setAttempt((n) => n + 1); }}>Try again</Button></>}
      {error && <Feedback tone="error">{error}</Feedback>}
      {phase === 'ready' && <>
        <p className={styles.outstanding} aria-live="polite">{rows.length}{cursor ? '+' : ''} shown · Still owed{active ? ' (matching filters)' : ''}: <strong>{money(outstanding)}</strong></p>
        {rows.length === 0 && !active && <p>Nothing yet. Finalize a split to create what people owe you.</p>}
        {rows.length === 0 && active && <><p>No receivables match these filters. Try a wider date range or a shorter search.</p><Button variant="secondary" onClick={clear}>Clear filters</Button></>}
        <ul className={styles.rows}>
          {rows.map((row) => (
            <li key={row.id} className={styles.row} tabIndex={-1} ref={(el) => { rowRefs.current[row.id] = el; }}>
              <div className={styles.head}><strong>{row.participant_name}</strong><span className={styles.status}>{row.status === 'PAID' ? 'Paid' : 'Owes you'}</span></div>
              <p className={styles.meta}>{money(row.amount_owed_minor)} · {row.merchant || 'Receipt'} · {row.receipt_date || 'No date'} · {row.payment_method ? METHOD[row.payment_method] : 'Not chosen'}{row.paid_at ? ` · paid ${row.paid_at.slice(0, 10)}` : ''}</p>
              {row.status === 'OWED' && row.payment_method === 'ZELLE' && <p className={styles.note}>Automatic matching needs a connected bank. Mark it paid yourself for now.</p>}
              <Button variant="secondary" disabled={busy === row.id} onClick={() => void toggle(row)}>{row.status === 'OWED' ? 'Mark paid' : 'Mark unpaid'}</Button>
            </li>
          ))}
        </ul>
        {cursor && <Button variant="secondary" disabled={more} onClick={() => void loadMore()}>{more ? 'Loading...' : 'Load more'}</Button>}
      </>}
    </section>
  );
}
