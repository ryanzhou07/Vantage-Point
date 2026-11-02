// @vitest-environment jsdom
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { JoinSplit } from './JoinSplit';
import { SplitSessionPanel } from './SplitSessionPanel';
import { receiptRequest } from './api';

vi.mock('./api', () => ({ receiptRequest: vi.fn() }));
let onStatus: ((s: string) => void) | undefined;
vi.mock('@supabase/supabase-js', () => ({
  createClient: () => ({
    channel: () => { const ch = { on: () => ch, subscribe: (cb: (s: string) => void) => { onStatus = cb; return ch; }, unsubscribe: vi.fn() }; return ch; },
    removeAllChannels: vi.fn(),
  }),
}));

const T0 = Date.parse('2026-10-28T12:00:00Z');
const people = [{ id: 'p1', name: 'Sam' }];
const joined = (extra = {}) => ({ state: 'ACTIVE', server_time: new Date(Date.now()).toISOString(), session: { id: 's1', expires_at: new Date(T0 + 300000).toISOString() }, receipt: { merchant: 'Cafe', currency: 'USD' }, participants: people, ...extra });
const reply = (status: number, body: unknown) => Promise.resolve(new Response(JSON.stringify(body), { status }));

beforeEach(() => {
  vi.useFakeTimers({ now: T0 });
  vi.stubEnv('NEXT_PUBLIC_SUPABASE_URL', 'http://127.0.0.1:1');
  vi.stubEnv('NEXT_PUBLIC_SUPABASE_ANON_KEY', 'anon');
  onStatus = undefined;
  sessionStorage.clear();
  vi.mocked(receiptRequest).mockReset();
});
afterEach(() => { cleanup(); vi.useRealTimers(); vi.unstubAllGlobals(); vi.unstubAllEnvs(); });

describe('open join and claims', () => {
  const items = (by: string) => [{ id: 'i1', name: 'Soup', amount_minor: 500, claimed_by: by }, { id: 'i2', name: 'Tea', amount_minor: 250, claimed_by: 'p1' }];
  const me = { id: 'p2', name: 'Jo' };

  it('joins with a typed name, claims an item, and shows a conflict when someone else holds it', async () => {
    window.location.hash = '#tok';
    const fetchMock = vi.fn().mockImplementation(() => reply(200, joined({ items: items('') })));
    vi.stubGlobal('fetch', fetchMock);
    render(<JoinSplit />);
    await act(async () => { await vi.advanceTimersByTimeAsync(0); });
    fireEvent.change(screen.getByLabelText('Your name'), { target: { value: 'Jo' } });
    fetchMock.mockImplementation(() => reply(200, joined({ participant: me, participants: [...people, me], items: items('') })));
    fireEvent.click(screen.getByRole('button', { name: 'Join' }));
    await act(async () => { await vi.advanceTimersByTimeAsync(0); });
    expect(JSON.parse(fetchMock.mock.calls.at(-1)![1].body).name).toBe('Jo');
    expect(screen.getByRole('button', { name: /Tea.*Taken by/ }).hasAttribute('disabled')).toBe(true);
    fetchMock.mockImplementation((url: string) => url.endsWith('/claim') ? reply(409, { error: { code: 'SPLIT_CLAIM_CONFLICT', message: 'x' } }) : reply(200, joined({ participant: me, items: items('p1') })));
    fireEvent.click(screen.getByRole('button', { name: /Soup.*Free/ }));
    await act(async () => { await vi.advanceTimersByTimeAsync(0); });
    expect(screen.getByRole('alert').textContent).toMatch(/Someone else already claimed/);
    expect(screen.getByRole('button', { name: /Soup.*Taken by/ })).toBeTruthy();
  });

  it('shows my total and lets me release a claim', async () => {
    window.location.hash = '#tok';
    const fetchMock = vi.fn().mockImplementation(() => reply(200, joined({ participant: me, participants: [...people, me], items: items('p2') })));
    vi.stubGlobal('fetch', fetchMock);
    render(<JoinSplit />);
    await act(async () => { await vi.advanceTimersByTimeAsync(0); });
    expect(screen.getByText(/Your items: \$5\.00/)).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: /Soup.*Yours/ }));
    await act(async () => { await vi.advanceTimersByTimeAsync(0); });
    const claimCall = fetchMock.mock.calls.find((c) => String(c[0]).endsWith('/claim'));
    expect(JSON.parse(claimCall![1].body)).toMatchObject({ item_id: 'i1', participant_id: 'p2', release: true });
  });
});

describe('participant join', () => {
  it('joins, shows the countdown, and shows expired once the backend rejects', async () => {
    window.location.hash = '#tok123';
    const fetchMock = vi.fn().mockImplementation(() => reply(200, joined()));
    vi.stubGlobal('fetch', fetchMock);
    render(<JoinSplit />);
    await act(async () => { await vi.advanceTimersByTimeAsync(0); });
    expect(JSON.parse(fetchMock.mock.calls[0][1].body).token).toBe('tok123');
    expect(screen.getByRole('timer').textContent).toBe('Open for 5:00');
    fetchMock.mockImplementation(() => reply(200, joined({ participant: people[0] })));
    fireEvent.click(screen.getByRole('button', { name: 'Sam' }));
    await act(async () => { await vi.advanceTimersByTimeAsync(0); });
    expect(screen.getByText('You are joined as Sam.')).toBeTruthy();
    fetchMock.mockImplementation(() => reply(401, { error: { code: 'SESSION_INVALID' } }));
    await act(async () => { await vi.advanceTimersByTimeAsync(300000); });
    expect(screen.getByRole('alert').textContent).toMatch(/expired.*Ask the receipt owner/);
    expect(screen.queryByRole('timer')).toBeNull();
    expect(window.localStorage.length).toBe(0);
  });

  it('shows an actionable error for an invalid token and when no token is present', async () => {
    window.location.hash = '#bad';
    vi.stubGlobal('fetch', vi.fn().mockImplementation(() => reply(401, { error: { code: 'SESSION_INVALID' } })));
    render(<JoinSplit />);
    await act(async () => { await vi.advanceTimersByTimeAsync(0); });
    expect(screen.getByRole('alert').textContent).toMatch(/Ask the receipt owner/);
  });

  it('keeps working and polls when realtime drops', async () => {
    window.location.hash = '#tok';
    const fetchMock = vi.fn().mockImplementation(() => reply(200, joined()));
    vi.stubGlobal('fetch', fetchMock);
    render(<JoinSplit />);
    await act(async () => { await vi.advanceTimersByTimeAsync(0); });
    expect(fetchMock).toHaveBeenCalledTimes(1);
    act(() => onStatus?.('CHANNEL_ERROR'));
    await act(async () => { await vi.advanceTimersByTimeAsync(15000); });
    expect(fetchMock.mock.calls.length).toBeGreaterThan(1);
    expect(screen.getByRole('button', { name: 'Sam' })).toBeTruthy();
  });
});

describe('owner session panel', () => {
  it('starts with a local QR and link, then reflects backend expiry and stops actions', async () => {
    const view = (state: string, extra = {}) => ({ state, server_time: new Date(Date.now()).toISOString(), session: { id: 's1', expires_at: new Date(T0 + 300000).toISOString() }, ...extra });
    vi.mocked(receiptRequest).mockImplementation(async (path: string, init?: RequestInit) => {
      if (init?.method === 'POST') return view('ACTIVE', { token: 'secret-token' }) as never;
      if (Date.now() >= T0 + 300000) return view('EXPIRED') as never;
      throw new Error('No active split session.');
    });
    render(<SplitSessionPanel receiptId="r1" />);
    await act(async () => { await vi.advanceTimersByTimeAsync(0); });
    fireEvent.click(screen.getByRole('button', { name: 'Start split session' }));
    await act(async () => { await vi.advanceTimersByTimeAsync(0); });
    expect((screen.getByLabelText('Link') as HTMLInputElement).value).toMatch(/\/split#secret-token$/);
    expect(screen.getByRole('img', { name: /QR code/ }).querySelector('svg')).toBeTruthy();
    expect(screen.getByRole('timer').textContent).toBe('Closes in 5:00');
    await act(async () => { await vi.advanceTimersByTimeAsync(300000); });
    expect(screen.getByText(/session expired/i)).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Close now' })).toBeNull();
    expect(screen.queryByLabelText('Link')).toBeNull();
    expect(window.localStorage.length).toBe(0);
  });
});
