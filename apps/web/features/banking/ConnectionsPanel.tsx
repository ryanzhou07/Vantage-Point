'use client';

import { useCallback, useEffect, useState } from 'react';
import { Button } from '../../components/ui/Button';
import { Feedback } from '../../components/ui/Feedback';
import { connect, createLinkToken, disconnect, listConnections, type Connection } from './api';

type PlaidHandler = { open(): void };
type PlaidGlobal = { create(o: { token: string; onSuccess(publicToken: string): void; onExit(err: unknown): void }): PlaidHandler };
declare global { interface Window { Plaid?: PlaidGlobal } }

const SRC = 'https://cdn.plaid.com/link/v2/stable/link-initialize.js';

// Loaded only on click, never at page load.
function loadPlaid(): Promise<PlaidGlobal> {
  if (window.Plaid) return Promise.resolve(window.Plaid);
  return new Promise((resolve, reject) => {
    const s = document.createElement('script');
    s.src = SRC; s.async = true;
    s.onload = () => (window.Plaid ? resolve(window.Plaid) : reject(new Error('missing')));
    s.onerror = () => reject(new Error('load'));
    document.head.appendChild(s);
  });
}

export function ConnectionsPanel() {
  const [rows, setRows] = useState<Connection[] | null>(null);
  const [loadError, setLoadError] = useState('');
  const [note, setNote] = useState<{ tone: 'status' | 'error'; text: string } | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    listConnections().then((r) => { setLoadError(''); setRows(r); }, (e: Error) => setLoadError(e.message));
  }, []);
  useEffect(load, [load]);

  async function start() {
    if (busy) return;
    setBusy(true); setNote(null);
    try {
      const [plaid, token] = await Promise.all([loadPlaid(), createLinkToken()]);
      plaid.create({
        token,
        onSuccess: (publicToken) => {
          connect(publicToken)
            .then(({ connection, created }) => {
              setRows((cur) => [connection, ...(cur ?? []).filter((r) => r.id !== connection.id)]);
              setNote({ tone: 'status', text: created ? `${connection.institution_name} connected.` : `${connection.institution_name} was already connected.` });
            })
            .catch((err: Error) => setNote({ tone: 'error', text: err.message }))
            .finally(() => setBusy(false));
        },
        onExit: (err) => {
          setBusy(false);
          setNote(err ? { tone: 'error', text: 'The bank connection did not complete. Try again.' } : { tone: 'status', text: 'Connection cancelled. Nothing was saved.' });
        },
      }).open();
    } catch (e) {
      setBusy(false);
      setNote({ tone: 'error', text: (e as Error).message === 'load' || (e as Error).message === 'missing' ? 'Bank connection could not load. Check your network and try again.' : (e as Error).message });
    }
  }

  async function remove(c: Connection) {
    setNote(null);
    try {
      await disconnect(c.id);
      setRows((cur) => (cur ?? []).filter((r) => r.id !== c.id));
      setNote({ tone: 'status', text: `${c.institution_name} disconnected.` });
    } catch (e) { setNote({ tone: 'error', text: (e as Error).message }); }
  }

  return (
    <div className="settings-connections">
      <Button onClick={start} disabled={busy}>{busy ? 'Connecting…' : 'Connect a bank'}</Button>
      {note && <Feedback tone={note.tone}>{note.text}</Feedback>}
      {loadError ? (
        <div><Feedback tone="error">{loadError}</Feedback><Button variant="secondary" onClick={load}>Retry</Button></div>
      ) : rows === null ? (
        <p role="status" className="settings-description">Loading connections…</p>
      ) : rows.length === 0 ? (
        <div className="settings-empty"><h4>No bank connections yet</h4><p className="settings-description">Connect a bank to see it listed here.</p></div>
      ) : (
        <ul className="settings-connection-list" aria-label="Connected institutions">
          {rows.map((c) => (
            <li key={c.id}>
              <div className="min-w-0"><p className="font-semibold break-words">{c.institution_name || 'Unnamed institution'}</p><p className="settings-description">{c.status} · connected {new Date(c.created_at).toLocaleDateString()}</p></div>
              <Button variant="secondary" aria-label={`Disconnect ${c.institution_name}`} onClick={() => remove(c)}>Disconnect</Button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
