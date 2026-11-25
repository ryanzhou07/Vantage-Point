'use client';

import { useEffect, useState } from 'react';
import { Feedback } from '../../components/ui/Feedback';
import { postJSON, receiptRequest, tripRequest } from '../receipts/api';
import type { TripRow } from './types';

// Files one of your receipts into a collection (or none). Only the receipt owner can do this.
export function CollectionPicker({ receiptId, value, disabled }: { receiptId: string; value: string | null | undefined; disabled?: boolean }) {
  const [trips, setTrips] = useState<TripRow[]>([]);
  const [current, setCurrent] = useState(value ?? '');
  const [error, setError] = useState('');
  useEffect(() => { tripRequest<{ trips: TripRow[] }>('').then((b) => setTrips(b.trips ?? [])).catch(() => {}); }, []);
  async function pick(id: string) {
    setError('');
    try { await receiptRequest(`/${receiptId}/collection`, postJSON({ collection_id: id || null })); setCurrent(id); }
    catch (e) { setError(e instanceof Error ? e.message : 'Could not file this receipt.'); }
  }
  return <>
    <label>Collection<select value={current} disabled={disabled} onChange={(e) => void pick(e.target.value)}>
      <option value="">No collection</option>
      {trips.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
    </select></label>
    {error && <Feedback tone="error">{error}</Feedback>}
  </>;
}
