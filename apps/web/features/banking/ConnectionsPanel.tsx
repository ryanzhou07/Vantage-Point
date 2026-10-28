'use client';

import { useCallback, useEffect, useState } from 'react';
import { Button } from '../../components/ui/Button';
import { Feedback } from '../../components/ui/Feedback';
import { disconnect, enroll, listConnections, type Connection } from './api';

type TellerEnrollment = { accessToken: string; enrollment: { id: string; institution: { name: string } } };
type TellerGlobal = { setup(o: {
  applicationId: string; environment: string; products: string[];
  onSuccess(e: TellerEnrollment): void; onExit(): void; onFailure(f: unknown): void;
}): { open(): void } };
declare global { interface Window { TellerConnect?: TellerGlobal } }

const SRC = 'https://cdn.teller.io/connect/connect.js';

function loadTeller(): Promise<TellerGlobal> {
  if (window.TellerConnect) return Promise.resolve(window.TellerConnect);
  return new Promise((resolve, reject) => {
    const s = document.createElement('script');
    s.src = SRC; s.async = false; // Teller requires a non-async load
    s.onload = () => (window.TellerConnect ? resolve(window.TellerConnect) : reject(new Error('missing')));
    s.onerror = () => reject(new Error('load'));
    document.head.appendChild(s);
  });
}

export function ConnectionsPanel() {
  const appId = process.env.NEXT_PUBLIC_TELLER_APPLICATION_ID;
  const env = process.env.NEXT_PUBLIC_TELLER_ENV || 'sandbox';
  const [rows, setRows] = useState<Connection[] | null>(null);
  const [loadError, setLoadError] = useState('');
  const [note, setNote] = useState<{ tone: 'status' | 'error'; text: string } | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    listConnections().then((r) => { setLoadError(''); setRows(r); }, (e: Error) => setLoadError(e.message));
  }, []);
  useEffect(load, [load]);

  async function connect() {
    if (!appId || busy) return;
    setBusy(true); setNote(null);
    try {
      const teller = await loadTeller();
      teller.setup({
        applicationId: appId, environment: env, products: ['verify'],
        onSuccess: (e) => {
          enroll({ enrollment_id: e.enrollment.id, access_token: e.accessToken, institution_name: e.enrollment.institution.name })
            .then(({ connection, created }) => {
              setRows((cur) => [connection, ...(cur ?? []).filter((r) => r.id !== connection.id)]);
              setNote({ tone: 'status', text: created ? `${connection.institution_name} connected.` : `${connection.institution_name} was already connected.` });
            })
            .catch((err: Error) => setNote({ tone: 'error', text: err.message }))
            .finally(() => setBusy(false));
        },
        onExit: () => { setBusy(false); setNote({ tone: 'status', text: 'Connection cancelled. Nothing was saved.' }); },
        onFailure: () => { setBusy(false); setNote({ tone: 'error', text: 'The bank connection did not complete. Try again.' }); },
      }).open();
    } catch {
      setBusy(false); setNote({ tone: 'error', text: 'Bank connection could not load. Check your network and try again.' });
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
      {!appId && <Feedback tone="error">Bank connection is not configured. Set NEXT_PUBLIC_TELLER_APPLICATION_ID.</Feedback>}
      {env === 'sandbox' && appId && <p className="settings-description">Sandbox mode: use test credentials only. Real bank logins do not work here.</p>}
      <Button onClick={connect} disabled={!appId || busy}>{busy ? 'Connecting…' : 'Connect a bank'}</Button>
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
