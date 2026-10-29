'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import qrcode from 'qrcode-generator';
import { Button } from '../../components/ui/Button';
import { Feedback } from '../../components/ui/Feedback';
import { receiptRequest } from './api';
import { formatRemaining, useSessionLive, type SessionView } from './splitSession';
import styles from './split.module.css';

// QR is rendered locally from the link; no third-party call. Output is our own generated SVG.
function Qr({ text }: { text: string }) {
  const qr = qrcode(0, 'M');
  qr.addData(text);
  qr.make();
  return <div className={styles.qr} role="img" aria-label="QR code for the split link" dangerouslySetInnerHTML={{ __html: qr.createSvgTag({ cellSize: 4, margin: 2, scalable: true }) }} />;
}

export function SplitSessionPanel({ receiptId, onUpdate }: { receiptId: string; onUpdate?: () => void }) {
  const [view, setView] = useState<SessionView | null>(null);
  const [link, setLink] = useState<{ id: string; url: string } | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [copied, setCopied] = useState(false);
  const mounted = useRef(true);
  const updateRef = useRef(onUpdate);
  useEffect(() => { updateRef.current = onUpdate; });
  useEffect(() => { mounted.current = true; return () => { mounted.current = false; }; }, []);

  const apply = useCallback((next: SessionView | undefined) => {
    if (!mounted.current || !next?.session) return;
    setView(next);
    // Kept in this tab only (cleared when it closes) so the QR and link survive a refresh during the short session.
    const key = `vp-split-link-${next.session.id}`;
    let token = next.token;
    try { if (token) sessionStorage.setItem(key, token); else token = sessionStorage.getItem(key) ?? undefined; } catch { /* ignore */ }
    if (token) setLink({ id: next.session.id, url: `${window.location.origin}/split#${token}` });
  }, []);
  const refetch = useCallback(async () => {
    updateRef.current?.(); // participants may have joined or claimed; let the owner view catch up
    try { apply(await receiptRequest<SessionView>(`/${receiptId}/session`)); } catch { /* keep last known state; next trigger retries */ }
  }, [receiptId, apply]);
  useEffect(() => { Promise.resolve(receiptRequest<SessionView>(`/${receiptId}/session`)).then(apply, () => undefined); }, [receiptId, apply]);

  async function run(path: string) {
    setBusy(true); setError(''); setCopied(false);
    try { apply(await receiptRequest<SessionView>(`/${receiptId}/session${path}`, { method: 'POST' })); }
    catch (e) { if (mounted.current) setError(e instanceof Error ? e.message : 'Split session request failed. Try again.'); }
    finally { if (mounted.current) setBusy(false); }
  }
  const active = view?.state === 'ACTIVE';
  const remaining = useSessionLive(view?.session ?? null, view?.server_time ?? null, active, () => void refetch());
  const url = link && view && link.id === view.session.id ? link.url : '';

  return (
    <section className={styles.card} aria-labelledby={`split-${receiptId}`}>
      <h2 id={`split-${receiptId}`}>Split session</h2>
      <p>Share a temporary link so people can join with their name and claim their own items. It closes automatically; the receipt stays saved.</p>
      {error && <Feedback tone="error">{error}</Feedback>}
      {active && <>
        <p className={styles.timer} role="timer" aria-live="off">{remaining > 0 ? `Closes in ${formatRemaining(remaining)}` : 'Confirming status...'}</p>
        {url ? <>
          <div><h3>QR code</h3><p>Scan with a phone camera.</p><Qr text={url} /></div>
          <div>
            <h3>Share link</h3>
            <label className={styles.linkLabel}>Link<input readOnly value={url} onFocus={(e) => e.currentTarget.select()} /></label>
            <div className={styles.actions}>
              <Button variant="secondary" onClick={() => { void navigator.clipboard?.writeText(url).then(() => setCopied(true), () => setError('Copy failed. Select the link and copy it manually.')); }}>Copy link</Button>
            </div>
            {copied && <Feedback>Link copied.</Feedback>}
          </div>
        </> : <Feedback>The QR code is only shown when a session starts. Restart to show a new one.</Feedback>}
        <div className={styles.actions}>
          <Button variant="secondary" disabled={busy} onClick={() => void run('/extend')}>Extend time</Button>
          <Button variant="secondary" disabled={busy} onClick={() => void run('/close')}>Close now</Button>
          <Button disabled={busy} onClick={() => void run('')}>Restart with new code</Button>
        </div>
      </>}
      {!active && <>
        {view?.state === 'EXPIRED' && <Feedback>This session expired. Participants can no longer open the link.</Feedback>}
        {view?.state === 'INVALIDATED' && <Feedback>This session was closed.</Feedback>}
        <div className={styles.actions}><Button disabled={busy} onClick={() => void run('')}>{view ? 'Start a new session' : 'Start split session'}</Button></div>
      </>}
    </section>
  );
}
