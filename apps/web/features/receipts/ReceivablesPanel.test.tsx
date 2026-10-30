// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('./api', () => ({ receivablesRequest: vi.fn(), postJSON: (v: unknown) => ({ method: 'POST', body: JSON.stringify(v) }) }));
import { receivablesRequest } from './api';
import { ReceivablesPanel } from './ReceivablesPanel';

const row = (id: string, name: string, extra = {}) => ({ id, receipt_id: 'r1', merchant: 'Cafe', receipt_date: '2026-10-29', participant_name: name, amount_owed_minor: 500, status: 'OWED' as const, paid_at: null, payment_method: 'VENMO' as const, ...extra });
const page = (rows: unknown[], next = '', outstanding = 500) => ({ receivables: rows, outstanding_minor: outstanding, next_cursor: next });
// Searches are POST /search with a JSON body (the shared auth middleware rejects URL query strings).
const bodies = () => vi.mocked(receivablesRequest).mock.calls.filter(([path]) => path === '/search').map(([, init]) => JSON.parse(String(init?.body)) as Record<string, string | number>);

beforeEach(() => { window.history.replaceState(null, '', '/dashboard/receipts'); vi.mocked(receivablesRequest).mockReset(); });
afterEach(cleanup);

describe('ReceivablesPanel filters and paging', () => {
  it('debounces the search, shows a removable chip, and the URL mirrors the filter', async () => {
    vi.mocked(receivablesRequest).mockResolvedValue(page([row('a', 'Bob')]) as never);
    render(<ReceivablesPanel />);
    await screen.findByText('Bob');
    const box = screen.getByLabelText('Search name or merchant');
    fireEvent.change(box, { target: { value: 'b' } });
    fireEvent.change(box, { target: { value: 'bo' } });
    fireEvent.change(box, { target: { value: 'bob' } });
    await waitFor(() => expect(bodies().some((b) => b.q === 'bob')).toBe(true));
    expect(bodies().filter((b) => 'q' in b)).toEqual([{ q: 'bob', limit: 25 }]); // one request, not one per keystroke
    expect(window.location.search).toContain('q=bob');
    fireEvent.click(await screen.findByRole('button', { name: 'Remove filter: Search: bob' }));
    await waitFor(() => expect(window.location.search).toBe(''));
  });

  it('sends nothing for an invalid date range and explains why', async () => {
    vi.mocked(receivablesRequest).mockResolvedValue(page([row('a', 'Bob')]) as never);
    render(<ReceivablesPanel />);
    await screen.findByText('Bob');
    fireEvent.change(screen.getByLabelText('From'), { target: { value: '2026-10-10' } });
    fireEvent.change(screen.getByLabelText('To'), { target: { value: '2026-10-01' } });
    await screen.findByRole('alert');
    expect(bodies().some((b) => b.to === '2026-10-01')).toBe(false);
  });

  it('loads more without duplicates and keeps the backend total', async () => {
    vi.mocked(receivablesRequest).mockImplementation(async (_path, init) => JSON.parse(String(init?.body)).cursor === 'c1' ? page([row('a', 'Bob'), row('b', 'Sam')], '', 900) as never : page([row('a', 'Bob')], 'c1') as never);
    render(<ReceivablesPanel />);
    fireEvent.click(await screen.findByRole('button', { name: 'Load more' }));
    await screen.findByText('Sam');
    expect(screen.getAllByText('Bob')).toHaveLength(1);
    expect(screen.queryByRole('button', { name: 'Load more' })).toBeNull();
    expect(screen.getByText(/\$9\.00/)).toBeTruthy();
  });

  it('tells apart "nothing yet" and "no matches", and shows the payment method', async () => {
    vi.mocked(receivablesRequest).mockResolvedValue(page([]) as never);
    render(<ReceivablesPanel />);
    await screen.findByText(/Nothing yet/);
    cleanup();
    window.history.replaceState(null, '', '/dashboard/receipts?status=PAID');
    vi.mocked(receivablesRequest).mockResolvedValue(page([]) as never);
    render(<ReceivablesPanel />);
    await screen.findByText(/No receivables match these filters/);
    expect(screen.getByRole('button', { name: 'Clear filters' })).toBeTruthy();
    cleanup();
    window.history.replaceState(null, '', '/dashboard/receipts');
    vi.mocked(receivablesRequest).mockResolvedValue(page([row('z', 'Zed', { payment_method: 'ZELLE' }), row('n', 'Nia', { payment_method: null })]) as never);
    render(<ReceivablesPanel />);
    await screen.findByText(/Zelle/);
    expect(screen.getByText(/Automatic matching needs a connected bank/)).toBeTruthy();
    expect(screen.getByText(/Not chosen/)).toBeTruthy();
  });
});
