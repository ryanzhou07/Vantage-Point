import { Button } from '../../components/ui/Button';
import type { Draft, Item, Participant } from './types';
import styles from './receipts.module.css';

type Props = { draft: Draft; disabled: boolean; onChange: (draft: Draft) => void };
function Cents({ label, value, onChange }: { label: string; value: number; onChange: (value: number) => void }) {
  return <label>{label} (cents)<input type="number" min="0" max="100000000" step="1" required value={value} onChange={(event) => onChange(event.target.valueAsNumber)} /></label>;
}
export function ReceiptEditor({ draft, disabled, onChange }: Props) {
  const update = (change: Partial<Draft>) => onChange({ ...draft, ...change });
  const item = (id: string, change: Partial<Item>) => update({ items: draft.items.map((value) => value.id === id ? { ...value, ...change } : value) });
  const person = (id: string, change: Partial<Participant>) => update({ participants: draft.participants.map((value) => value.id === id ? { ...value, ...change } : value) });

  return (
    <fieldset disabled={disabled} className={styles.editor}>
      <legend>Review and assign your receipt</legend>
      <p>Extraction is a proposal. Check the image and correct every line. Enter complete line amounts in cents; quantity is descriptive.</p>
      <div className={styles.fields}>
        <label>Merchant<input maxLength={200} value={draft.merchant} onChange={(event) => update({ merchant: event.target.value })} /></label>
        <label>Receipt date (YYYY-MM-DD)<input maxLength={32} value={draft.date} onChange={(event) => update({ date: event.target.value })} /></label>
        <label>Currency<select value={draft.currency} onChange={(event) => update({ currency: event.target.value })}><option value="USD">USD</option>{draft.currency !== 'USD' && <option value={draft.currency}>{draft.currency || 'Unknown'}</option>}</select></label>
      </div>

      <h3>Participants</h3>
      <p>Included participants share tax/tip by your chosen method. Anyone assigned an item must be included.</p>
      {draft.participants.map((participant, index) => (
        <div className={styles.person} key={participant.id}>
          <label>Participant {index + 1}<input required maxLength={100} value={participant.name} onChange={(event) => person(participant.id, { name: event.target.value })} /></label>
          <label className={styles.check}><input type="checkbox" checked={participant.included} onChange={(event) => person(participant.id, { included: event.target.checked })} />Included</label>
          <Button variant="secondary" onClick={() => update({ participants: draft.participants.filter((value) => value.id !== participant.id), items: draft.items.map((value) => value.participant_id === participant.id ? { ...value, participant_id: '' } : value) })}>Remove participant {index + 1}</Button>
        </div>
      ))}
      <Button variant="secondary" disabled={draft.participants.length >= 40} onClick={() => update({ participants: [...draft.participants, { id: crypto.randomUUID(), name: '', included: true }] })}>Add participant</Button>

      <h3>Whole item lines</h3>
      {draft.items.map((line, index) => (
        <fieldset className={styles.line} key={line.id}>
          <legend>Item {index + 1}</legend>
          <label>Name<input required maxLength={200} value={line.name} onChange={(event) => item(line.id, { name: event.target.value })} /></label>
          <label>Quantity<input type="number" min="1" max="1000" step="1" required value={line.quantity} onChange={(event) => item(line.id, { quantity: event.target.valueAsNumber })} /></label>
          <Cents label="Line amount" value={line.amount_minor} onChange={(amount_minor) => item(line.id, { amount_minor })} />
          <label>Assigned person<select value={line.participant_id} onChange={(event) => item(line.id, { participant_id: event.target.value })}><option value="">Not assigned</option>{draft.participants.map((participant) => <option key={participant.id} value={participant.id}>{participant.name || 'Unnamed participant'}</option>)}</select></label>
          <Button variant="secondary" onClick={() => update({ items: draft.items.filter((value) => value.id !== line.id) })}>Remove item {index + 1}</Button>
        </fieldset>
      ))}
      <Button variant="secondary" disabled={draft.items.length >= 200} onClick={() => update({ items: [...draft.items, { id: crypto.randomUUID(), name: '', quantity: 1, amount_minor: 0, participant_id: '' }] })}>Add item</Button>

      <h3>Reconcile amounts</h3>
      <p>Nonzero receipt-level discounts block calculation in this first version. Resolve them explicitly into verified item line amounts and subtotal; the original extraction remains available.</p>
      <div className={styles.fields}>
        <Cents label="Subtotal" value={draft.subtotal_minor} onChange={(subtotal_minor) => update({ subtotal_minor })} />
        <Cents label="Tax" value={draft.tax_minor} onChange={(tax_minor) => update({ tax_minor })} />
        <Cents label="Tip" value={draft.tip_minor} onChange={(tip_minor) => update({ tip_minor })} />
        <Cents label="Receipt-level discount" value={draft.discount_minor} onChange={(discount_minor) => update({ discount_minor })} />
        <Cents label="Receipt total" value={draft.total_minor} onChange={(total_minor) => update({ total_minor })} />
      </div>
      <div className={styles.fields}>
        {(['tax', 'tip'] as const).map((kind) => <label key={kind}>{kind === 'tax' ? 'Tax allocation' : 'Tip allocation'}<select value={draft[`${kind}_method`]} onChange={(event) => update({ [`${kind}_method`]: event.target.value })}><option value="">Choose a method</option><option value="proportional">Proportional to item subtotal</option><option value="equal">Equal among included participants</option></select></label>)}
      </div>
    </fieldset>
  );
}
