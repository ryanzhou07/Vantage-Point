// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ReceiptWorkspace } from './ReceiptWorkspace';
import { ReceiptEditor } from './ReceiptEditor';
import { receiptRequest } from './api';
import { initialDraft, type Receipt } from './types';

vi.mock('./api', () => ({ receiptRequest: vi.fn(), postJSON: (body: unknown) => ({ method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }) }));
const fixture: Receipt = {
  id: 'receipt-one', status: 'NEEDS_REVIEW', version: 3, created_at: '2026-10-15', error_code: null, retries: 0, draft: null, preview: null,
  proposal: { merchant: 'Synthetic cafe', date: '', currency: 'USD', items: [{ id: 'meal', name: 'Meal', quantity: 1, amount_minor: 1000, participant_id: '' }], subtotal_minor: 1000, tax_minor: 80, tip_minor: 200, discount_minor: 0, total_minor: 1280 },
};
beforeEach(() => { vi.mocked(receiptRequest).mockReset(); });
afterEach(() => { cleanup(); vi.restoreAllMocks(); });

describe('owner receipt review', () => {
  it('keeps methods explicit and preserves whole-line amounts when quantity changes', () => {
    const draft = initialDraft(fixture)!;
    const onChange = vi.fn();
    render(<ReceiptEditor draft={draft} disabled={false} onChange={onChange} />);
    expect((screen.getByLabelText('Tax allocation') as HTMLSelectElement).value).toBe('');
    expect((screen.getByLabelText('Tip allocation') as HTMLSelectElement).value).toBe('');
    fireEvent.change(screen.getByLabelText('Quantity'), { target: { value: '3' } });
    expect(onChange.mock.lastCall?.[0].items[0]).toMatchObject({ quantity: 3, amount_minor: 1000 });
    fireEvent.click(screen.getByRole('button', { name: 'Add participant' }));
    expect(onChange.mock.lastCall?.[0].participants[0]).toMatchObject({ name: '', included: true });
  });

  it('loads a proposal, sends corrected draft/version and renders only the backend preview', async () => {
    const draft = { ...initialDraft(fixture)!, participants: [{ id: 'a', name: 'Student', included: true }], tax_method: 'equal', tip_method: 'proportional' };
    draft.items = draft.items.map((item) => ({ ...item, participant_id: 'a' }));
    const saved = { ...fixture, version: 4, draft, preview: { draft: true as const, total_minor: 1280, shares: [{ participant_id: 'a', items_minor: 1000, tax_minor: 80, tip_minor: 200, total_minor: 1280 }] } };
    vi.mocked(receiptRequest).mockImplementation(async (path, init) => {
      if (path === '') return { receipts: [fixture] } as never;
      if (!init?.method) return saved as never;
      return saved as never;
    });
    render(<ReceiptWorkspace />);
    await waitFor(() => expect(screen.getAllByRole('option').length).toBe(2));
    fireEvent.change(screen.getByLabelText('Recent drafts (latest 20)'), { target: { value: fixture.id } });
    await screen.findByRole('heading', { name: 'Saved draft split preview' });
    expect(screen.getByText(/Nobody has confirmed/)).toBeTruthy();
    fireEvent.change(screen.getByLabelText('Merchant'), { target: { value: 'Reviewed cafe' } });
    expect(screen.queryByRole('heading', { name: 'Saved draft split preview' })).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Verify and calculate draft split' }));
    await screen.findByText('Owner-reviewed draft preview saved.');
    const request = vi.mocked(receiptRequest).mock.calls.find(([path, init]) => path.endsWith('/review') && init?.method === 'POST');
    expect(JSON.parse(request![1]!.body as string)).toMatchObject({ version: 4, calculate: true, draft: { merchant: 'Reviewed cafe' } });
  });

  it('surfaces conflicts while preserving unsaved corrections', async () => {
    vi.mocked(receiptRequest).mockImplementation(async (path, init) => {
      if (path === '') return { receipts: [fixture] } as never;
      if (!init?.method) return fixture as never;
      throw new Error('This receipt changed. Reload before saving.');
    });
    render(<ReceiptWorkspace />);
    await waitFor(() => expect(screen.getAllByRole('option').length).toBe(2));
    fireEvent.change(screen.getByLabelText('Recent drafts (latest 20)'), { target: { value: fixture.id } });
    await screen.findByLabelText('Merchant');
    fireEvent.change(screen.getByLabelText('Merchant'), { target: { value: 'My correction' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save draft' }));
    await screen.findByRole('alert');
    expect((screen.getByLabelText('Merchant') as HTMLInputElement).value).toBe('My correction');
  });

  it('does not automatically retry paid work when extraction fails', async () => {
    const failed = { ...fixture, status: 'PROCESSING_FAILED' as const, proposal: null, error_code: 'VISION_ACCESS_REQUIRED' };
    vi.mocked(receiptRequest).mockImplementation(async (path) => path === '' ? { receipts: [failed] } as never : failed as never);
    render(<ReceiptWorkspace />);
    await waitFor(() => expect(screen.getAllByRole('option').length).toBe(2));
    fireEvent.change(screen.getByLabelText('Recent drafts (latest 20)'), { target: { value: fixture.id } });
    await screen.findByText(/Gemini 3.5 Flash-Lite access was denied/);
    expect(vi.mocked(receiptRequest).mock.calls.some(([path]) => path.endsWith('/retry'))).toBe(false);
    expect(screen.getByRole('button', { name: 'Retry extraction (2 remaining)' })).toBeTruthy();
  });
  it.each(['same-receipt.png', 'different-receipt.png'])('keeps unsaved receipt A when upload of %s is canceled before any request', async (filename) => {
    vi.mocked(receiptRequest).mockImplementation(async (path) => path === '' ? { receipts: [fixture] } as never : fixture as never);
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false);
    render(<ReceiptWorkspace />);
    await waitFor(() => expect(screen.getAllByRole('option').length).toBe(2));
    fireEvent.change(screen.getByLabelText('Recent drafts (latest 20)'), { target: { value: fixture.id } });
    await screen.findByLabelText('Merchant');
    fireEvent.change(screen.getByLabelText('Merchant'), { target: { value: 'Unsaved receipt A correction' } });
    const before = vi.mocked(receiptRequest).mock.calls.length;
    fireEvent.change(screen.getByLabelText('Receipt image'), { target: { files: [new File(['synthetic'], filename, { type: 'image/png' })] } });
    fireEvent.click(screen.getByRole('button', { name: 'Upload and extract' }));
    await waitFor(() => expect(confirm).toHaveBeenCalledOnce());
    await waitFor(() => expect((screen.getByRole('button', { name: 'Upload and extract' }) as HTMLButtonElement).disabled).toBe(false));
    expect(vi.mocked(receiptRequest).mock.calls.length).toBe(before);
    expect((screen.getByLabelText('Merchant') as HTMLInputElement).value).toBe('Unsaved receipt A correction');
    expect((screen.getByLabelText('Recent drafts (latest 20)') as HTMLSelectElement).value).toBe(fixture.id);
    expect(screen.getByText(/Unsaved changes/)).toBeTruthy();
  });

  it('shows recovered legacy fields as needing owner review without inventing a date or preview', async () => {
    const recovered = { ...fixture, error_code: 'LEGACY_REVIEW_REQUIRED' };
    vi.mocked(receiptRequest).mockImplementation(async (path) => path === '' ? { receipts: [recovered] } as never : recovered as never);
    render(<ReceiptWorkspace />);
    await waitFor(() => expect(screen.getAllByRole('option').length).toBe(2));
    fireEvent.change(screen.getByLabelText('Recent drafts (latest 20)'), { target: { value: fixture.id } });
    await screen.findByText('Review the date, split methods and item assignments before calculating this recovered draft.');
    expect((screen.getByLabelText('Receipt date (YYYY-MM-DD)') as HTMLInputElement).value).toBe('');
    expect(screen.queryByRole('heading', { name: 'Saved draft split preview' })).toBeNull();
    expect((screen.getByLabelText('Merchant') as HTMLInputElement).value).toBe('Synthetic cafe');
  });

});
