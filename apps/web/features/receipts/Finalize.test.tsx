// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ReceivablesPanel } from './ReceivablesPanel';
import { ReceiptWorkspace } from './ReceiptWorkspace';
import { receiptRequest, receivablesRequest } from './api';
import type { Draft, Receipt } from './types';

vi.mock('./api', () => ({ receiptRequest: vi.fn(), receivablesRequest: vi.fn(), postJSON: (body: unknown) => ({ method: 'POST', headers: {}, body: JSON.stringify(body) }) }));
vi.mock('./SplitSessionPanel', () => ({ SplitSessionPanel: () => null }));

const draft: Draft = {
  merchant: 'Cafe', date: '2026-10-29', currency: 'USD', subtotal_minor: 800, tax_minor: 0, tip_minor: 0, discount_minor: 0, total_minor: 800, tax_method: 'equal', tip_method: 'equal',
  items: [{ id: 'i1', name: 'Soup', quantity: 1, amount_minor: 500, participant_id: 'bob' }, { id: 'i2', name: 'Tea', quantity: 1, amount_minor: 300, participant_id: 'me' }],
  participants: [{ id: 'me', name: 'Ada', included: true, is_owner: true }, { id: 'bob', name: 'Bob', included: true }],
};
const preview = { draft: true as const, total_minor: 800, shares: [{ participant_id: 'me', items_minor: 300, tax_minor: 0, tip_minor: 0, total_minor: 300 }, { participant_id: 'bob', items_minor: 500, tax_minor: 0, tip_minor: 0, total_minor: 500 }] };
const base: Receipt = { id: 'r1', status: 'NEEDS_REVIEW', version: 7, created_at: '2026-10-29', error_code: null, retries: 0, proposal: null, draft, preview };

beforeEach(() => { vi.mocked(receiptRequest).mockReset(); vi.mocked(receivablesRequest).mockReset(); localStorage.clear(); });
afterEach(() => { cleanup(); vi.restoreAllMocks(); });

describe('finalize and receivables', () => {
  it('finalizes only with a current preview, sending the version, then locks the view', async () => {
    const done = { ...base, status: 'FINALIZED' as const, version: 8 };
    vi.mocked(receiptRequest).mockImplementation(async (path, init) => path === '' ? { receipts: [base] } as never : (init?.method === 'POST' ? done : base) as never);
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    render(<ReceiptWorkspace />);
    const button = await screen.findByRole('button', { name: 'Finalize receipt' }) as HTMLButtonElement;
    expect(button.disabled).toBe(false);
    fireEvent.click(button);
    await screen.findByText(/This split is locked/);
    const call = vi.mocked(receiptRequest).mock.calls.find(([path]) => path === '/r1/finalize');
    expect(JSON.parse(call![1]!.body as string)).toEqual({ version: 7 });
    expect((screen.getByRole('button', { name: 'Delete receipt' }) as HTMLButtonElement).disabled).toBe(true);
    expect(screen.queryByRole('button', { name: 'Finalize receipt' })).toBeNull();
  });

  it('cannot finalize without a calculated preview', async () => {
    const noPreview = { ...base, preview: null };
    vi.mocked(receiptRequest).mockImplementation(async (path) => path === '' ? { receipts: [noPreview] } as never : noPreview as never);
    render(<ReceiptWorkspace />);
    expect(((await screen.findByRole('button', { name: 'Finalize receipt' })) as HTMLButtonElement).disabled).toBe(true);
  });

  it('shows what is owed, and marks a receivable paid', async () => {
    const row = { id: 'c1', receipt_id: 'r1', merchant: 'Cafe', receipt_date: '2026-10-29', participant_name: 'Bob', amount_owed_minor: 500, status: 'OWED' as const, paid_at: null };
    vi.mocked(receivablesRequest).mockImplementation(async (path) => path === '/search' ? { receivables: [row], outstanding_minor: 500, next_cursor: '' } as never : {} as never);
    render(<ReceivablesPanel />);
    await screen.findByText('Bob');
    expect(screen.getByText(/Still owed:/).textContent).toContain('$5.00');
    fireEvent.click(screen.getByRole('button', { name: 'Mark paid' }));
    await waitFor(() => expect(vi.mocked(receivablesRequest)).toHaveBeenCalledWith('/c1/paid', expect.objectContaining({ method: 'POST', body: JSON.stringify({ paid: true }) })));
  });
});
