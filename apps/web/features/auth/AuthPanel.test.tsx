// @vitest-environment jsdom
import React from 'react';
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthPanel } from './AuthPanel';
import { useAuthSession } from './useAuthSession';

const fake = vi.hoisted(() => ({
  loadProfile: vi.fn(), signIn: vi.fn(), signOut: vi.fn(), unsubscribe: vi.fn(),
  listener: undefined as undefined | (() => void),
}));

vi.mock('../../lib/auth/client', () => ({
  authClient: () => ({ auth: { onAuthStateChange: (listener: () => void) => {
    fake.listener = listener;
    return { data: { subscription: { unsubscribe: fake.unsubscribe } } };
  } } }),
}));
vi.mock('../../lib/auth/flows', () => ({
  loadProfile: fake.loadProfile, signIn: fake.signIn, signOut: fake.signOut,
}));

const profile = { display_name: 'Sample User', email: 'sample@example.test' };

beforeEach(() => {
  vi.clearAllMocks();
  fake.listener = undefined;
  fake.loadProfile.mockResolvedValue(null);
  fake.signIn.mockResolvedValue(undefined);
  fake.signOut.mockResolvedValue(undefined);
});
afterEach(cleanup);

async function ready() {
  await waitFor(() => expect((screen.getByRole('button', { name: /Google/ }) as HTMLButtonElement).disabled).toBe(false));
}

describe('auth presentation and session control', () => {
  it('announces initial loading and provides a keyboard-operable named Google action', async () => {
    render(<AuthPanel />);
    expect(screen.getByRole('status').textContent).toContain('Loading');
    expect((screen.getByRole('button', { name: /Google/ }) as HTMLButtonElement).disabled).toBe(true);
    await ready();
    screen.getByRole('button', { name: /Google/ }).focus();
    expect(document.activeElement).toBe(screen.getByRole('button', { name: /Google/ }));
  });

  it('disables duplicate actions, presents failures and retries the authoritative profile load', async () => {
    let reject!: (reason: Error) => void;
    fake.signIn.mockReturnValue(new Promise((_, fail) => { reject = fail; }));
    render(<AuthPanel />);
    await ready();
    const button = screen.getByRole('button', { name: /Google/ });
    fireEvent.click(button);
    fireEvent.click(button);
    expect(fake.signIn).toHaveBeenCalledTimes(1);
    expect((button as HTMLButtonElement).disabled).toBe(true);
    await act(async () => reject(new Error('Please try again.')));
    expect(screen.getByRole('alert').textContent).toBe('Please try again.');
    fake.loadProfile.mockResolvedValue(profile);
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));
    await screen.findByRole('button', { name: 'Sign out' });
    expect(screen.queryByRole('alert')).toBeNull();
    expect(screen.getByText(/Signed in as Sample User/)).toBeTruthy();
  });

  it('preserves a signed-in profile on failed sign-out and clears it after success', async () => {
    fake.loadProfile.mockResolvedValue(profile);
    fake.signOut.mockRejectedValueOnce(new Error('Sign-out could not complete.'));
    render(<AuthPanel />);
    const button = await screen.findByRole('button', { name: 'Sign out' });
    fireEvent.click(button);
    await screen.findByRole('alert');
    expect(screen.getByText(/Signed in as Sample User/)).toBeTruthy();
    fireEvent.click(button);
    await screen.findByRole('button', { name: /Google/ });
    expect(screen.queryByText(/Signed in as Sample User/)).toBeNull();
  });

  it('cancels stale reads on auth events and unmount, and unsubscribes', async () => {
    const signals: AbortSignal[] = [];
    const finishes: ((value: unknown) => void)[] = [];
    fake.loadProfile.mockImplementation((_: unknown, __: unknown, signal: AbortSignal) => {
      signals.push(signal);
      return new Promise(resolve => finishes.push(resolve));
    });
    const view = render(<AuthPanel />);
    await waitFor(() => expect(signals).toHaveLength(1));
    act(() => fake.listener?.());
    await waitFor(() => expect(signals).toHaveLength(2));
    expect(signals[0].aborted).toBe(true);
    await act(async () => finishes[0](profile));
    expect(screen.queryByText(/Signed in as/)).toBeNull();
    view.unmount();
    expect(signals[1].aborted).toBe(true);
    expect(fake.unsubscribe).toHaveBeenCalledOnce();
    await act(async () => finishes[1](profile));
  });

  it('clears scheduled refreshes when unmounted before the deferred read', async () => {
    const view = render(<AuthPanel />);
    view.unmount();
    await new Promise(resolve => setTimeout(resolve, 10));
    expect(fake.loadProfile).not.toHaveBeenCalled();
  });

  it('deduplicates programmatic actions before a rerender can disable the UI', async () => {
    let state!: ReturnType<typeof useAuthSession>;
    function Probe() { state = useAuthSession(); return null; }
    fake.signIn.mockReturnValue(new Promise(() => {}));
    render(<Probe />);
    act(() => { void state.signIn(); void state.signIn(); });
    expect(fake.signIn).toHaveBeenCalledTimes(1);
  });
});
