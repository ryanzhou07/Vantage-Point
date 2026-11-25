export type DashboardDestination = {
  slug: string;
  label: string;
  group: 'Banking' | 'Receipt splitting';
  description: string;
};

export const destinations: readonly DashboardDestination[] = [
  { slug: 'institutions', label: 'Institutions', group: 'Banking', description: 'Connect and manage your financial institutions.' },
  { slug: 'accounts', label: 'Accounts & balances', group: 'Banking', description: 'View accounts and current balances from connected institutions.' },
  { slug: 'transactions', label: 'Transactions', group: 'Banking', description: 'Browse transaction history from your accounts.' },
  { slug: 'recurring', label: 'Recurring activity', group: 'Banking', description: 'Review detected recurring transactions and payments.' },
  { slug: 'receipts', label: 'Drafts', group: 'Receipt splitting', description: 'Upload, review and collaboratively split itemized receipts.' },
  { slug: 'collections', label: 'Collections', group: 'Receipt splitting', description: 'Group receipts into collections and share trips with members.' },
  { slug: 'receipt-history', label: 'Browse', group: 'Receipt splitting', description: 'Search drafts and finalized receipts and see what each person owes.' },
  { slug: 'receivables', label: 'Owed to me', group: 'Receipt splitting', description: 'Track amounts owed and manually record payment status.' },
];

export function destinationFor(slug: string) {
  return destinations.find((destination) => destination.slug === slug);
}
