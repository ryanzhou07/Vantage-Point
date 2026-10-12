// @vitest-environment jsdom
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import type { Profile } from '../../lib/auth/flows';
import { DashboardShell } from './dashboard-shell';

const mocks = vi.hoisted(() => ({
  load: vi.fn(),
  signOut: vi.fn(),
  subscribe: vi.fn(),
  unsubscribe: vi.fn(),
}));
vi.mock('../../lib/auth/client', () => ({ authClient: () => ({ auth: { onAuthStateChange: mocks.subscribe } }) }));
vi.mock('../../lib/auth/flows', () => ({ loadProfile: mocks.load, signOut: mocks.signOut }));
vi.mock('next/navigation', () => ({ usePathname: () => '/dashboard' }));

let authChanged: () => void;
const profile = (id: string): Profile => ({ id, display_name: `Owner ${id}`, email: '', avatar_url: '', created_at: '', updated_at: '' });
const renderShell = () => render(<DashboardShell><h1>Private dashboard content</h1></DashboardShell>);

beforeEach(() => {
  vi.clearAllMocks();
  mocks.load.mockResolvedValue(profile('A'));
  mocks.signOut.mockResolvedValue(undefined);
  mocks.subscribe.mockImplementation((callback: () => void) => {
    authChanged = callback;
    return { data: { subscription: { unsubscribe: mocks.unsubscribe } } };
  });
});
afterEach(cleanup);

it('does not mount dashboard content while loading or signed out', async () => {
  mocks.load.mockResolvedValue(null);
  renderShell();
  expect(screen.getByRole('status').textContent).toContain('Loading');
  expect(screen.queryByText('Private dashboard content')).toBeNull();
  await screen.findByText('Sign in to access your dashboard.');
  expect(screen.queryByRole('navigation')).toBeNull();
  expect(screen.getByRole('link', { name: 'Return to sign in' }).getAttribute('href')).toBe('/');
});

it('shows safe error, then retries and restores authenticated content', async () => {
  mocks.load.mockRejectedValueOnce(new Error('private credentials'));
  renderShell();
  expect((await screen.findByRole('alert')).textContent).not.toContain('private credentials');
  expect(screen.queryByText('Private dashboard content')).toBeNull();
  fireEvent.click(screen.getByRole('button', { name: 'Try again' }));
  await screen.findByText('Owner A');
  expect(screen.getByText('Private dashboard content')).toBeTruthy();
});

it('clears previous user content on auth change and unsubscribes on unmount', async () => {
  const view = renderShell();
  await screen.findByText('Owner A');
  mocks.load.mockResolvedValue(profile('B'));
  act(authChanged);
  expect(screen.queryByText('Owner A')).toBeNull();
  expect(screen.queryByText('Private dashboard content')).toBeNull();
  await screen.findByText('Owner B');
  view.unmount();
  expect(mocks.unsubscribe).toHaveBeenCalledOnce();
});

it('supports mobile menu expansion, link closure and accessible navigation without domain calls', async () => {
  const fetcher = vi.spyOn(globalThis, 'fetch');
  renderShell();
  await screen.findByText('Owner A');
  const menu = screen.getByRole('button', { name: 'Browse dashboard' });
  expect(menu.getAttribute('aria-expanded')).toBe('false');
  menu.focus();
  expect(document.activeElement).toBe(menu);
  fireEvent.click(menu);
  expect(menu.getAttribute('aria-expanded')).toBe('true');
  const accounts = screen.getByRole('link', { name: 'Accounts & balances' });
  expect(accounts.getAttribute('href')).toBe('/dashboard/accounts');
  accounts.focus();
  expect(document.activeElement).toBe(accounts);
  accounts.addEventListener('click', (event) => event.preventDefault());
  fireEvent.click(accounts);
  expect(menu.getAttribute('aria-expanded')).toBe('false');
  expect(fetcher).not.toHaveBeenCalled();
  expect(mocks.load).toHaveBeenCalledOnce();
  fetcher.mockRestore();
});

it('deduplicates sign out and removes private content after success', async () => {
  let finish!: () => void;
  mocks.signOut.mockImplementation(() => new Promise<void>((resolve) => { finish = resolve; }));
  renderShell();
  await screen.findByText('Owner A');
  fireEvent.click(screen.getByRole('button', { name: 'Sign out' }));
  fireEvent.click(screen.getByRole('button', { name: 'Signing out…' }));
  expect(mocks.signOut).toHaveBeenCalledOnce();
  mocks.load.mockResolvedValue(null);
  await act(async () => finish());
  await screen.findByText('Sign in to access your dashboard.');
  expect(screen.queryByText('Private dashboard content')).toBeNull();
});

it('reports sign-out failure and allows a retry', async () => {
  mocks.signOut.mockRejectedValueOnce(new Error('secret'));
  renderShell();
  await screen.findByText('Owner A');
  fireEvent.click(screen.getByRole('button', { name: 'Sign out' }));
  expect((await screen.findByRole('alert')).textContent).toBe('Sign-out could not complete. Please try again.');
  await waitFor(() => expect((screen.getByRole('button', { name: 'Sign out' }) as HTMLButtonElement).disabled).toBe(false));
  fireEvent.click(screen.getByRole('button', { name: 'Sign out' }));
  await waitFor(() => expect(mocks.signOut).toHaveBeenCalledTimes(2));
});
