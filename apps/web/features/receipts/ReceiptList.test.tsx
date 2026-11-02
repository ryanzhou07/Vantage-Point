// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ReceiptList, type SummaryRow } from './ReceiptList';
import { receiptRequest } from './api';

vi.mock('./api', () => ({ receiptRequest: vi.fn() }));
const row = (over: Partial<SummaryRow> = {}): SummaryRow => ({
  id: 'r1', merchant: 'Cafe', date: '2026-10-01', status: 'NEEDS_REVIEW', currency: 'USD', total_minor: 1000, amount_kind: 'DRAFT_PREVIEW',
  shares: [{ participant_id: 'p1', name: 'Sam', total_minor: 667 }, { participant_id: 'p2', name: 'Lee', total_minor: 333 }], shares_total_minor: 1000, ...over,
});
beforeEach(() => { vi.mocked(receiptRequest).mockReset(); });
afterEach(cleanup);

describe('receipts list', () => {
  it('shows draft rows labeled as not owed, distinct from finalized, with exact participant amounts', async () => {
    vi.mocked(receiptRequest).mockResolvedValue({ receipts: [row(), row({ id: 'r2', merchant: 'Deli', amount_kind: 'FINALIZED' })], next_cursor: '', outstanding_minor: 0 } as never);
    render(<ReceiptList />);
    await waitFor(() => expect(screen.getByText('Cafe')).toBeTruthy());
    expect(screen.getByText('Draft preview, not owed')).toBeTruthy();
    expect(screen.getByText('Finalized')).toBeTruthy();
    expect(screen.getAllByText('$6.67').length).toBeGreaterThan(0);
    expect(screen.getAllByText('$10.00').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Outstanding from finalized receipts:')[0].parentElement?.textContent).toContain('$0.00');
  });

  it('pages with the backend cursor', async () => {
    vi.mocked(receiptRequest)
      .mockResolvedValueOnce({ receipts: [row()], next_cursor: 'c1', outstanding_minor: 0 } as never)
      .mockResolvedValueOnce({ receipts: [row({ id: 'r2', merchant: 'Deli' })], next_cursor: '', outstanding_minor: 0 } as never);
    render(<ReceiptList />);
    fireEvent.click(await screen.findByRole('button', { name: 'Load more' }));
    await screen.findByText('Deli');
    expect(vi.mocked(receiptRequest).mock.calls[1][0]).toBe('/summary/10/c1');
    expect(screen.queryByRole('button', { name: 'Load more' })).toBeNull();
  });

  it('renders loading, empty and error states', async () => {
    vi.mocked(receiptRequest).mockReturnValue(new Promise(() => undefined) as never);
    render(<ReceiptList />);
    expect(screen.getByRole('status').textContent).toMatch(/Loading/);
    cleanup();
    vi.mocked(receiptRequest).mockResolvedValue({ receipts: [], next_cursor: '', outstanding_minor: 0 } as never);
    render(<ReceiptList />);
    await screen.findByText(/No receipts yet/);
    cleanup();
    vi.mocked(receiptRequest).mockRejectedValueOnce(new Error('x')).mockResolvedValue({ receipts: [], next_cursor: '', outstanding_minor: 0 } as never);
    render(<ReceiptList />);
    expect((await screen.findByRole('alert')).textContent).toMatch(/could not be loaded/);
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));
    await screen.findByText(/No receipts yet/);
  });
});
