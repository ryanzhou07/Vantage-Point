// @vitest-environment jsdom
import React, { StrictMode } from 'react';
import { act, cleanup, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';

const callback = vi.hoisted(() => ({ complete: vi.fn() }));
vi.mock('../../lib/auth/client', () => ({ authClient: () => ({}) }));
vi.mock('../../lib/auth/flows', () => ({ completeCallback: callback.complete }));

beforeEach(() => {
  vi.resetModules();
  vi.clearAllMocks();
  window.history.replaceState(null, '', '/auth/callback?code=synthetic-code');
});
afterEach(cleanup);

it('exchanges once in StrictMode and clears callback params before presenting a safe error and return link', async () => {
  callback.complete.mockRejectedValue(new Error('provider-detail-that-must-not-render'));
  const { CallbackPanel } = await import('./CallbackPanel');
  render(<StrictMode><CallbackPanel /></StrictMode>);
  expect(screen.getByRole('status').textContent).toContain('Verifying');
  const alert = await screen.findByRole('alert');
  expect(alert.textContent).toContain('Please return and try again');
  expect(alert.textContent).not.toContain('provider-detail');
  expect(callback.complete).toHaveBeenCalledTimes(1);
  expect(window.location.search).toBe('');
  expect(screen.getByRole('link', { name: 'Return to sign-in' }).getAttribute('href')).toBe('/');
});

it('ignores late callback completion after unmount', async () => {
  let finish!: () => void;
  callback.complete.mockReturnValue(new Promise<void>(resolve => { finish = resolve; }));
  const { CallbackPanel } = await import('./CallbackPanel');
  const view = render(<CallbackPanel />);
  await waitFor(() => expect(callback.complete).toHaveBeenCalledTimes(1));
  view.unmount();
  await act(async () => finish());
  expect(window.location.search).toBe('?code=synthetic-code');
});
