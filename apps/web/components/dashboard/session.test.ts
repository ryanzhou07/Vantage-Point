import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { Profile } from '../../lib/auth/flows';
import { createDashboardSession, type DashboardSession } from './session';

const owner = (id: string): Profile => ({ id, display_name: id, email: `${id}@example.test`, avatar_url: '', created_at: '', updated_at: '' });
const tick = () => vi.advanceTimersByTimeAsync(0);

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

describe('Dashboard private session lifecycle', () => {
  it('clears owner A immediately and rejects a late response after owner B signs in', async () => {
    const pending: ((profile: Profile | null) => void)[] = [];
    const states: DashboardSession[] = [];
    const load = vi.fn((_signal: AbortSignal) => new Promise<Profile | null>((resolve) => pending.push(resolve)));
    const session = createDashboardSession(load, (state) => states.push(state));

    session.refresh();
    await tick();
    pending[0](owner('A'));
    await tick();
    expect(states.at(-1)).toEqual({ status: 'ready', profile: owner('A') });

    session.refresh();
    expect(states.at(-1)).toEqual({ status: 'loading' });
    await tick();
    session.refresh();
    await tick();
    pending[2](owner('B'));
    await tick();
    pending[1](owner('A'));
    await tick();
    expect(states.at(-1)).toEqual({ status: 'ready', profile: owner('B') });
    expect(load.mock.calls[1][0].aborted).toBe(true);
    session.dispose();
  });

  it('shows signed out and never retains previous private state', async () => {
    const publish = vi.fn();
    const load = vi.fn().mockResolvedValueOnce(owner('A')).mockResolvedValueOnce(null);
    const session = createDashboardSession(load, publish);
    session.refresh();
    await tick();
    session.refresh();
    expect(publish).toHaveBeenLastCalledWith({ status: 'loading' });
    await tick();
    expect(publish).toHaveBeenLastCalledWith({ status: 'signed-out' });
    session.dispose();
  });

  it('uses safe errors and recovers with an explicit retry', async () => {
    const publish = vi.fn();
    const load = vi.fn().mockRejectedValueOnce(new Error('private token')).mockResolvedValueOnce(owner('B'));
    const session = createDashboardSession(load, publish);
    session.refresh();
    await tick();
    expect(publish).toHaveBeenLastCalledWith({ status: 'error', message: 'Your profile could not be loaded. Try again, or return to sign in.' });
    session.refresh();
    await tick();
    expect(publish).toHaveBeenLastCalledWith({ status: 'ready', profile: owner('B') });
    session.dispose();
  });

  it('deduplicates pending auth events and cancels loads on unmount', async () => {
    const publish = vi.fn();
    const load = vi.fn().mockResolvedValue(owner('A'));
    const session = createDashboardSession(load, publish);
    session.refresh();
    session.refresh();
    session.refresh();
    await tick();
    expect(load).toHaveBeenCalledOnce();
    session.refresh();
    session.dispose();
    await tick();
    expect(load).toHaveBeenCalledOnce();
    expect(publish).toHaveBeenLastCalledWith({ status: 'loading' });
    session.refresh();
    expect(load).toHaveBeenCalledOnce();
  });

  it('ignores late resolution after unmount', async () => {
    let resolve!: (profile: Profile) => void;
    const publish = vi.fn();
    const session = createDashboardSession(() => new Promise<Profile>((done) => { resolve = done; }), publish);
    session.refresh();
    await tick();
    session.dispose();
    resolve(owner('A'));
    await tick();
    expect(publish).toHaveBeenCalledOnce();
  });
});
