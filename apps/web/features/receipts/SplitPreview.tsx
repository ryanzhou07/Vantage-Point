import { money, type Draft, type Preview } from './types';
import styles from './receipts.module.css';
export function SplitPreview({ preview, draft }: { preview: Preview; draft: Draft }) {
  return (
    <section aria-labelledby="split-preview" className={styles.preview}>
      <h2 id="split-preview">Saved draft split preview</h2>
      <p>This is an owner-reviewed draft. Nobody has confirmed a share, and no payment or receivable has been created.</p>
      <p>Tax: {draft.tax_method}. Tip: {draft.tip_method}. Currency: USD.</p>
      <div className={styles.table}>
        <table><caption>Per-person draft amounts</caption><thead><tr><th scope="col">Person</th><th scope="col">Items</th><th scope="col">Tax</th><th scope="col">Tip</th><th scope="col">Total</th></tr></thead>
          <tbody>{preview.shares.map((share) => <tr key={share.participant_id}><th scope="row">{draft.participants.find((person) => person.id === share.participant_id)?.name}</th><td>{money(share.items_minor)}</td><td>{money(share.tax_minor)}</td><td>{money(share.tip_minor)}</td><td>{money(share.total_minor)}</td></tr>)}</tbody>
          <tfoot><tr><th scope="row" colSpan={4}>Receipt total</th><td>{money(preview.total_minor)}</td></tr></tfoot>
        </table>
      </div>
    </section>
  );
}
