export type TripRow = { id: string; name: string; is_owner: boolean; members: number; receipts: number; total_minor: number };
export type TripDetail = {
  id: string; name: string; is_owner: boolean;
  members: { user_id: string; name: string; is_owner: boolean }[];
  receipts: { id: string; merchant: string; date: string; status: string; total_minor: number; mine: boolean }[];
};
export type MemberView = {
  status: 'NEEDS_REVIEW' | 'FINALIZED' | string; receipt: { merchant: string; currency: string };
  participant?: { id: string; name: string }; participants: { id: string; name: string }[];
  items: { id: string; name: string; amount_minor: number; claimed_by: string[] }[];
};
