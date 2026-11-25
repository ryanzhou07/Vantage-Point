// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { PendingInvite } from './PendingInvite';
import { TripsPanel } from './TripsPanel';

vi.mock('../receipts/api', () => ({ tripRequest: vi.fn(), receiptRequest: vi.fn(), postJSON: (v: unknown) => ({ method: 'POST', body: JSON.stringify(v) }) }));
vi.mock('../../components/dashboard/profile-context', () => ({ useDashboardProfile: () => ({ id: 'u1' }) }));
vi.mock('next/link', () => ({ default: ({ children, href }: { children: React.ReactNode; href: string }) => <a href={href}>{children}</a> }));
import { tripRequest } from '../receipts/api';
const trip = vi.mocked(tripRequest);

beforeEach(() => { trip.mockReset(); sessionStorage.clear(); window.location.hash = ''; });
afterEach(cleanup);

describe('collections and trips', () => {
  it('lists collections, marks shared ones as trips, and creates a new one', async () => {
    trip.mockResolvedValueOnce({ trips: [{ id: 't1', name: 'Trip A', is_owner: true, members: 3, receipts: 2, total_minor: 4500 }, { id: 't2', name: 'Groceries', is_owner: true, members: 1, receipts: 0, total_minor: 0 }] });
    render(<TripsPanel />);
    await screen.findByText('Trip A');
    expect(screen.getByText('Trip · 3 members')).toBeTruthy();
    expect(screen.getByText('Private')).toBeTruthy();
    trip.mockResolvedValueOnce({ id: 't3' }).mockResolvedValueOnce({ trips: [] });
    fireEvent.change(screen.getByLabelText('New collection'), { target: { value: ' Japan ' } });
    fireEvent.click(screen.getByRole('button', { name: 'Create' }));
    await waitFor(() => expect(trip).toHaveBeenCalledWith('', expect.objectContaining({ body: JSON.stringify({ name: 'Japan' }) })));
  });

  it('joins from an invite link fragment, clears the fragment and never keeps the token', async () => {
    window.location.hash = '#invite-token-0123456789abc';
    trip.mockResolvedValueOnce({ name: 'Trip A' });
    render(<PendingInvite fromLink />);
    await screen.findByText(/You joined/);
    expect(trip).toHaveBeenCalledWith('/join', expect.objectContaining({ body: JSON.stringify({ token: 'invite-token-0123456789abc' }) }));
    expect(window.location.hash).toBe('');
    expect(sessionStorage.getItem('vp-trip-invite')).toBeNull();
  });

  it('shows a clear error for an expired invite', async () => {
    window.location.hash = '#expired-token-0123456789abc';
    trip.mockRejectedValueOnce(new Error('This invite link is invalid or has expired.'));
    render(<PendingInvite fromLink />);
    await screen.findByText(/invalid or has expired/);
  });
});
