'use client';

import { useEffect, useRef, useState } from 'react';
import { Button } from '../../components/ui/Button';
import { Feedback } from '../../components/ui/Feedback';
import { receiptRequest, receiptImage, postJSON } from './api';
import { ReceiptEditor } from './ReceiptEditor';
import { SplitPreview } from './SplitPreview';
import { SplitSessionPanel } from './SplitSessionPanel';
import { initialDraft, type Draft, type Receipt } from './types';
import styles from './receipts.module.css';

export function ReceiptWorkspace() {
  const [receipts, setReceipts] = useState<Receipt[]>([]);
  const [receipt, setReceipt] = useState<Receipt | null>(null);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [dirty, setDirty] = useState(false);
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
      .then((body) => { if (mounted.current) setReceipts(body.receipts ?? []); })
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
    setDraft(initialDraft(value));
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

  async function open(id: string) {
    if (dirty && !window.confirm('Discard unsaved changes and load this receipt?')) return;
    accept(await receiptRequest<Receipt>(`/${id}`));
    setImageURL('');
  }

  return (
    <div className={styles.workspace}>
      <header><p className={styles.eyebrow}>RECEIPTS / OWNER DRAFT</p><h1>From receipt to shared total</h1><p>Upload, check the details, assign whole items, and preview everyone’s share.</p></header>
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
        <div className={styles.actions}><h2 id="current-receipt">Current receipt</h2><Button variant="secondary" disabled={busy} onClick={() => void action(() => open(receipt.id))}>Reload receipt</Button></div>
        {receipt.status === 'PROCESSING' && <Feedback>Processing image… You can return later; the job is saved.</Feedback>}
        {receipt.error_code === 'LEGACY_REVIEW_REQUIRED' && !receipt.preview && <Feedback>Review the date, split methods and item assignments before calculating this recovered draft.</Feedback>}
        {receipt.status === 'DRAFT' && <Feedback>The upload did not finish. Upload the same image again to resume safely.</Feedback>}
        {receipt.status === 'PROCESSING_FAILED' && <><Feedback tone="error">{receipt.error_code === 'VISION_ACCESS_REQUIRED' ? 'Gemini 3.5 Flash-Lite access was denied. Ask the owner to verify the API key and model access; no other model was used.' : receipt.error_code === 'VISION_MODEL_UNAVAILABLE' ? 'Gemini 3.5 Flash-Lite is unavailable for this project. Ask the owner to verify model eligibility/access; no other model was used.' : receipt.error_code === 'VISION_RATE_LIMITED' ? 'The vision provider quota or rate limit was reached. Wait and check project quota before retrying.' : 'Extraction could not finish. Check the image and worker configuration.'}</Feedback><Button disabled={busy || receipt.retries >= 2} onClick={() => void action(async () => accept(await receiptRequest<Receipt>(`/${receipt.id}/retry`, postJSON({ version: receipt.version }))))}>Retry extraction ({2 - receipt.retries} remaining)</Button></>}
        <p>The original image and extracted proposal remain unchanged when you edit a draft.</p>
        {receipt.proposal && <details><summary>Original extracted proposal</summary><pre className={styles.proposal}>{JSON.stringify(receipt.proposal, null, 2)}</pre></details>}
        {!imageURL && <Button variant="secondary" disabled={busy} onClick={() => void action(async () => { const image = await receiptImage(receipt.id); if (mounted.current) setImageURL(URL.createObjectURL(image)); })}>Load private receipt image</Button>}
        {imageURL && <a href={imageURL} target="_blank" rel="noreferrer">Open your selected image for comparison</a>}
        {draft && <>
          <ReceiptEditor draft={draft} disabled={busy} onChange={(value) => { setDraft(value); setDirty(true); }} />
          <div className={styles.actions}><Button variant="secondary" disabled={busy} onClick={() => void action(() => save(false))}>Save draft</Button><Button disabled={busy} onClick={() => void action(() => save(true))}>Verify and calculate draft split</Button></div>
          {dirty && <Feedback>Unsaved changes. Save and calculate again to update the draft preview.</Feedback>}
          {!dirty && receipt.preview && <SplitPreview preview={receipt.preview} draft={draft} />}
        </>}
        {receipt.status === 'NEEDS_REVIEW' && <SplitSessionPanel receiptId={receipt.id} onUpdate={() => void syncClaims()} />}
      </section>}
    </div>
  );
}
