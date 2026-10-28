export type Item = { id: string; name: string; quantity: number; amount_minor: number; participant_id: string };
export type Participant = { id: string; name: string; included: boolean; is_owner?: boolean };
export type Proposal = {
  merchant: string; date: string; currency: string; items: Item[];
  subtotal_minor: number; tax_minor: number; tip_minor: number; discount_minor: number; total_minor: number;
};
export type Draft = Proposal & { participants: Participant[]; tax_method: string; tip_method: string };
export type Preview = {
  draft: true; total_minor: number;
  shares: { participant_id: string; items_minor: number; tax_minor: number; tip_minor: number; total_minor: number }[];
};
export type Receipt = {
  id: string; status: 'DRAFT' | 'PROCESSING' | 'NEEDS_REVIEW' | 'PROCESSING_FAILED'; version: number;
  proposal: Proposal | null; draft: Draft | null; preview: Preview | null; error_code: string | null; retries: number; created_at: string;
};
export function initialDraft(receipt: Receipt): Draft | null {
  if (receipt.draft) return receipt.draft;
  if (!receipt.proposal) return null;
  return { ...receipt.proposal, participants: [], tax_method: '', tip_method: '' };
}
export const money = (cents: number) => new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(cents / 100);

// The signed-in owner is always a locked participant so later "who owes me" totals can separate their own share.
export function withOwner(draft: Draft | null, ownerName: string | null): Draft | null {
  if (!draft || !ownerName || draft.participants.some((p) => p.is_owner)) return draft;
  return { ...draft, participants: [{ id: crypto.randomUUID(), name: ownerName, included: true, is_owner: true }, ...draft.participants] };
}
