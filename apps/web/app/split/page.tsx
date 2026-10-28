import type { Metadata } from 'next';
import { JoinSplit } from '../../features/receipts/JoinSplit';

export const metadata: Metadata = { title: 'Join split', referrer: 'no-referrer', robots: { index: false } };

export default function SplitPage() {
  return <JoinSplit />;
}
