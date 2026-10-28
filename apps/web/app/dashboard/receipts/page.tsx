import { ReceiptList } from '../../../features/receipts/ReceiptList';
import { ReceiptWorkspace } from '../../../features/receipts/ReceiptWorkspace';

export default function ReceiptsPage() {
  return (
    <div style={{ display: 'grid', gap: '2rem', minWidth: 0 }}>
      <ReceiptList />
      <ReceiptWorkspace />
    </div>
  );
}
