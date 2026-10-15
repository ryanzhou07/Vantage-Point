// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { DashboardProfileContext } from '../dashboard/profile-context';
import { SettingsScreen } from './settings-screen';
import { DashboardShell } from '../dashboard/dashboard-shell';

vi.mock('../../lib/auth/client', () => ({ authClient: () => ({ auth: { onAuthStateChange: () => ({ data: { subscription: { unsubscribe() {} } } }) } }) }));
vi.mock('../../lib/auth/flows', () => ({ loadProfile: async () => null, signOut: vi.fn() }));
vi.mock('next/navigation', () => ({ usePathname: () => '/dashboard/settings' }));

afterEach(() => { cleanup(); vi.restoreAllMocks(); });

it('uses the real read-only profile, labels draft fields and discards drafts when leaving a section', () => {
  render(<DashboardProfileContext.Provider value={{ id: 'synthetic', display_name: 'Test Owner', email: 'owner@example.test', avatar_url: '', created_at: '', updated_at: '' }}><SettingsScreen /></DashboardProfileContext.Provider>);
  expect(screen.getByText('owner@example.test')).toBeTruthy();
  const name = screen.getByLabelText('Display name preview') as HTMLInputElement;
  expect(name.value).toBe('Test Owner');
  fireEvent.change(name, { target: { value: 'Draft only' } });
  fireEvent.click(screen.getByRole('button', { name: 'Account' }));
  expect(screen.getByText('owner@example.test')).toBeTruthy();
  expect(screen.getByRole('button', { name: 'Manage email' }).hasAttribute('disabled')).toBe(true);
  fireEvent.click(screen.getByRole('button', { name: 'General' }));
  expect((screen.getByLabelText('Display name preview') as HTMLInputElement).value).toBe('Test Owner');
});

it('offers accessible local theme previews without global or storage changes', () => {
  const storage = vi.spyOn(Storage.prototype, 'setItem');
  const bodyClass = document.body.className;
  render(<SettingsScreen />);
  const appearance = screen.getByRole('button', { name: 'Appearance' });
  appearance.focus();
  expect(document.activeElement).toBe(appearance);
  fireEvent.click(appearance);
  expect(appearance.getAttribute('aria-current')).toBe('page');
  fireEvent.click(screen.getByRole('radio', { name: 'Dark' }));
  expect((screen.getByRole('radio', { name: 'Dark' }) as HTMLInputElement).checked).toBe(true);
  expect(screen.getByRole('status').textContent).toContain('Dark selected for this preview only');
  expect(document.body.className).toBe(bodyClass);
  expect(storage).not.toHaveBeenCalled();
});

it('never sends connection/payment mutations or claims fake connected accounts', () => {
  const fetcher = vi.spyOn(globalThis, 'fetch');
  render(<SettingsScreen />);
  fireEvent.click(screen.getByRole('button', { name: 'Plugins / Connections' }));
  expect(screen.getByText('No bank connections to display')).toBeTruthy();
  expect(screen.getByText(/contact details, not live account integrations/)).toBeTruthy();
  fireEvent.change(screen.getByLabelText('Venmo username preview'), { target: { value: '@test-only' } });
  for (const name of ['Connect a bank', 'Save payment details', 'Remove payment details']) {
    const action = screen.getByRole('button', { name });
    expect(action.hasAttribute('disabled')).toBe(true);
    fireEvent.click(action);
  }
  expect(fetcher).not.toHaveBeenCalled();
});

it('does not expose Settings content during unauthenticated direct access', async () => {
  render(<DashboardShell><SettingsScreen /></DashboardShell>);
  await screen.findByText('Sign in to access your dashboard.');
  expect(screen.queryByRole('heading', { name: 'Settings' })).toBeNull();
  expect(screen.queryByLabelText('Display name preview')).toBeNull();
});
