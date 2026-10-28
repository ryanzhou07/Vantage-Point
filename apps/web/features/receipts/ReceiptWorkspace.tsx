'use client';

import { useEffect, useRef, useState } from 'react';
import { Button } from '../../components/ui/Button';
import { Feedback } from '../../components/ui/Feedback';
import { receiptRequest, receiptImage, postJSON } from './api';
import { ReceiptEditor } from './ReceiptEditor';
import { SplitPreview } from './SplitPreview';
import { SplitSessionPanel } from './SplitSessionPanel';
import { useDashboardProfile } from '../../components/dashboard/profile-context';
import { initialDraft, money, withOwner, type Draft, type Receipt } from './types';
import { missingFields } from './validate';
import styles from './receipts.module.css';

const ACTIVE_KEY = 'vp-active-receipt';

export function ReceiptWorkspace() {
  const profile = useDashboardProfile();
  const ownerName = useRef<string | null>(null);
  useEffect(() => { ownerName.current = profile ? (profile.display_name || profile.email || 'Me') : null; });
  const [receipts, setReceipts] = useState<Receipt[]>([]);
  const [receipt, setReceipt] = useState<Receipt | null>(null);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [dirty, setDirty] = useState(false);
  const [editing, setEditing] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [imageURL, setImageURL] = useState('');
  const pending = useRef(false);
  const mounted = useRef(false);
  const fileInput = useRef<HTMLInputElement>(null);

  useEffect(() => {
    mounted.current = true;
    const controller = new AbortController();
    receiptRequest<{ receipts: Receipt[] }>('', { signal: controller.signal })
      .then(async (body) => {
        if (!mounted.current) return;
        const list = body.receipts ?? [];
        setReceipts(list);
        // Resume the receipt you were last working on (else the newest) so a refresh does not need a re-click.
        let last: string | null = null;
        try { last = localStorage.getItem(ACTIVE_KEY); } catch { /* per-viewer convenience only */ }
        const target = list.find((entry) => entry.id === last) ?? list[0];
        if (!target) return;
        const value = await receiptRequest<Receipt>(`/${target.id}`, { signal: controller.signal });
        if (mounted.current && !controller.signal.aborted) setReceipt((current) => { if (current) return current; setDraft(withOwner(initialDraft(value), ownerName.current)); return value; });
      })
      .catch(() => { if (mounted.current && !controller.signal.aborted) setError('Your receipts could not be loaded. Refresh when the Receipt service is ready.'); });
    return () => { mounted.current = false; controller.abort(); };
  }, []);

  useEffect(() => {
    if (!imageURL) return;
    return () => URL.revokeObjectURL(imageURL);
  }, [imageURL]);

  function accept(value: Receipt) {
    if (!mounted.current) return;
    setReceipt(value);
    try { localStorage.setItem(ACTIVE_KEY, value.id); } catch { /* ignore */ }
    setDraft(withOwner(initialDraft(value), ownerName.current));
    setDirty(false);
    setReceipts((current) => [value, ...current.filter((entry) => entry.id !== value.id)].slice(0, 20));
  }

  async function action(run: () => Promise<void>) {
    if (pending.current) return;
    pending.current = true;
    setBusy(true);
    setError('');
    setNotice('');
    try { await run(); }
    catch (failure) { if (mounted.current) setError(failure instanceof Error ? failure.message : 'Receipt operation failed. Try again.'); }
    finally { pending.current = false; if (mounted.current) setBusy(false); }
  }

  // Only reads existing work. Stops after two minutes; never enqueues paid work.
  useEffect(() => {
    if (receipt?.status !== 'PROCESSING') return;
    const controller = new AbortController();
    let requests = 0;
    const timer = setInterval(async () => {
      if (document.hidden || pending.current) return;
      if (++requests > 30) { clearInterval(timer); setNotice('Processing is taking longer. Use Reload receipt to check again.'); return; }
      try {
        const value = await receiptRequest<Receipt>(`/${receipt.id}`, { signal: controller.signal });
        if (!controller.signal.aborted) accept(value);
      } catch { if (!controller.signal.aborted) { clearInterval(timer); setNotice('Automatic refresh stopped. Use Reload receipt to try again.'); } }
    }, 4000);
    return () => { clearInterval(timer); controller.abort(); };
  }, [receipt?.id, receipt?.status]);

  async function upload() {
    // Guard before file inspection, reservation, storage upload or paid work.
    if (dirty && !window.confirm('Discard unsaved changes before uploading a receipt?')) return;
    const file = fileInput.current?.files?.[0];
    if (!file) throw new Error('Choose a receipt image first.');
    if (file.size > 5 * 1024 * 1024 || !['image/jpeg', 'image/png'].includes(file.type)) throw new Error('Choose a JPEG or PNG up to 5 MiB.');
    setNotice('Uploading image. Keep this page open…');
    const value = await receiptRequest<Receipt>('', { method: 'POST', headers: { 'Content-Type': file.type }, body: file });
    accept(value);
    setImageURL(URL.createObjectURL(file));
    setNotice('Image uploaded. Processing runs in the Receipt worker.');
  }

  async function save(calculate: boolean) {
    if (!receipt || !draft) return;
    const value = await receiptRequest<Receipt>(`/${receipt.id}/review`, postJSON({ version: receipt.version, draft, calculate }));
    accept(value);
    setNotice(calculate ? 'Owner-reviewed draft preview saved.' : 'Draft saved. Calculation has not been verified.');
  }

  // Pick up joins/claims made by participants, but never replace edits the owner has not saved.
  async function syncClaims() {
    if (!receipt || dirty || pending.current) return;
    try {
      const value = await receiptRequest<Receipt>(`/${receipt.id}`);
      if (value.version !== receipt.version && !pending.current) accept(value);
    } catch { /* next trigger retries */ }
  }

  const missing = draft ? missingFields(draft) : [];

  async function remove() {
    if (!receipt || !window.confirm('Delete this receipt, its image and all claims? This cannot be undone.')) return;
    const id = receipt.id;
    await receiptRequest<{ deleted: boolean }>(`/${id}`, { method: 'DELETE' });
    if (!mounted.current) return;
    try { if (localStorage.getItem(ACTIVE_KEY) === id) localStorage.removeItem(ACTIVE_KEY); } catch { /* ignore */ }
    setReceipt(null); setDraft(null); setDirty(false); setImageURL('');
    setReceipts((current) => current.filter((entry) => entry.id !== id));
    window.dispatchEvent(new Event('receipts:changed'));
    setNotice('Receipt deleted.');
  }

  async function confirm() {
    await save(false);
    setEditing(false);
    setNotice('Receipt confirmed. Share the QR code or link so people can claim their items.');
  }

  async function open(id: string) {
    if (dirty && !window.confirm('Discard unsaved changes and load this receipt?')) return;
    accept(await receiptRequest<Receipt>(`/${id}`));
    setEditing(false);
    setImageURL('');
  }

  return (
    <div className={styles.workspace}>
      <header><p className={styles.eyebrow}>RECEIPTS / OWNER DRAFT</p><h1>From receipt to shared total</h1><p>Upload, check the details, share the link so people claim their own items, and preview everyone’s share.</p></header>
      <section className={styles.card} aria-labelledby="receipt-upload">
        <h2 id="receipt-upload">Attach a receipt</h2>
        <p>JPEG or PNG, up to 5 MiB and 16 megapixels. Your image stays private. Processing sends it to the configured vision provider.</p>
        <label className={styles.file}>Receipt image<input ref={fileInput} type="file" accept="image/jpeg,image/png" disabled={busy} /></label>
        <Button disabled={busy} onClick={() => void action(upload)}>{busy ? 'Working…' : 'Upload and extract'}</Button>
      </section>
      {error && <Feedback tone="error">{error}</Feedback>}
      {notice && <Feedback>{notice}</Feedback>}
      <label className={styles.file}>Recent drafts (latest 20)<select value={receipt?.id ?? ''} disabled={busy} onChange={(event) => { if (event.target.value) void action(() => open(event.target.value)); }}><option value="">Choose a receipt</option>{receipts.map((value) => <option key={value.id} value={value.id}>{value.draft?.merchant || value.proposal?.merchant || 'Receipt'} — {value.status.toLowerCase().replaceAll('_', ' ')} — {value.created_at.slice(0, 10)}</option>)}</select></label>

      {receipt && <section className={styles.card} aria-labelledby="current-receipt">
        <div className={styles.actions}><h2 id="current-receipt">Current receipt</h2><Button variant="secondary" disabled={busy} onClick={() => void action(() => open(receipt.id))}>Reload receipt</Button><Button variant="secondary" disabled={busy || receipt.status === 'PROCESSING'} onClick={() => void action(remove)}>Delete receipt</Button></div>
        {receipt.status === 'PROCESSING' && <Feedback>Processing image… You can return later; the job is saved.</Feedback>}
        {receipt.error_code === 'LEGACY_REVIEW_REQUIRED' && !receipt.preview && <Feedback>Review the date, split methods and item assignments before calculating this recovered draft.</Feedback>}
        {receipt.status === 'DRAFT' && <Feedback>The upload did not finish. Upload the same image again to resume safely.</Feedback>}
        {receipt.status === 'PROCESSING_FAILED' && <><Feedback tone="error">{receipt.error_code === 'VISION_ACCESS_REQUIRED' ? 'Gemini 3.5 Flash-Lite access was denied. Ask the owner to verify the API key and model access; no other model was used.' : receipt.error_code === 'VISION_MODEL_UNAVAILABLE' ? 'Gemini 3.5 Flash-Lite is unavailable for this project. Ask the owner to verify model eligibility/access; no other model was used.' : receipt.error_code === 'VISION_RATE_LIMITED' ? 'The vision provider quota or rate limit was reached. Wait and check project quota before retrying.' : 'Extraction could not finish. Check the image and worker configuration.'}</Feedback><Button disabled={busy || receipt.retries >= 2} onClick={() => void action(async () => accept(await receiptRequest<Receipt>(`/${receipt.id}/retry`, postJSON({ version: receipt.version }))))}>Retry extraction ({2 - receipt.retries} remaining)</Button></>}
        <p>The original image and extracted proposal remain unchanged when you edit a draft.</p>
        {receipt.proposal && <details><summary>Original extracted proposal</summary><pre className={styles.proposal}>{JSON.stringify(receipt.proposal, null, 2)}</pre></details>}
        {!imageURL && <Button variant="secondary" disabled={busy} onClick={() => void action(async () => { const image = await receiptImage(receipt.id); if (mounted.current) setImageURL(URL.createObjectURL(image)); })}>Load private receipt image</Button>}
        {imageURL && <a href={imageURL} target="_blank" rel="noreferrer">Open your selected image for comparison</a>}
        {draft && (editing || !receipt.draft) && <>
          <ReceiptEditor draft={draft} disabled={busy} onChange={(value) => { setDraft(value); setDirty(true); }} />
          {missing.length > 0
            ? <div className="vp-feedback vp-feedback--error" role="alert"><strong>Fix these before you can continue ({missing.length}):</strong><ul>{missing.map((line) => <li key={line}>{line}</li>)}</ul></div>
            : <Feedback>Everything is filled in. Confirm to move on to sharing.</Feedback>}
          {dirty && <Feedback>Unsaved changes. Confirm or save the draft to keep them.</Feedback>}
          <div className={styles.actions}><Button disabled={busy || missing.length > 0} onClick={() => void action(confirm)}>Confirm receipt and continue</Button><Button variant="secondary" disabled={busy} onClick={() => void action(() => save(false))}>Save draft</Button></div>
        </>}
        {draft && receipt.draft && !editing && receipt.status === 'NEEDS_REVIEW' && <>
          <section className={styles.card} aria-labelledby="confirmed-receipt">
            <h2 id="confirmed-receipt">Step 2 of 2 · Share and claim</h2>
            <p>{receipt.draft.merchant} · {receipt.draft.date} · total {money(receipt.draft.total_minor)}. People scan the code or open the link, enter their name and claim their own items.</p>
            <ul>{draft.items.map((line) => <li key={line.id}>{line.name} · {money(line.amount_minor)} · {draft.participants.find((p) => p.id === line.participant_id)?.name ?? 'not claimed yet'}</li>)}</ul>
            <div className={styles.actions}>
              <Button variant="secondary" disabled={busy} onClick={() => setEditing(true)}>Edit receipt details</Button>
              <Button disabled={busy || draft.items.some((line) => !line.participant_id)} onClick={() => void action(() => save(true))}>Calculate split preview</Button>
            </div>
            {draft.items.some((line) => !line.participant_id) && <p>The split preview unlocks once every item is claimed.</p>}
          </section>
          {!dirty && receipt.preview && <SplitPreview preview={receipt.preview} draft={draft} />}
          <SplitSessionPanel receiptId={receipt.id} onUpdate={() => void syncClaims()} />
        </>}
      </section>}
    </div>
  );
}
