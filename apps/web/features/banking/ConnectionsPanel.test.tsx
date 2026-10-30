// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { ConnectionsPanel } from './ConnectionsPanel';
import * as api from './api';

vi.mock('./api');
const conn = { id: 'c1', provider: 'plaid', institution_name: 'Test Bank', status: 'ACTIVE', created_at: '2026-01-02T00:00:00Z' };
let opts: Parameters<NonNullable<typeof window.Plaid>['create']>[0];
const open = vi.fn();

beforeEach(() => {
  vi.mocked(api.listConnections).mockResolvedValue([]);
  vi.mocked(api.createLinkToken).mockResolvedValue('link-sandbox-1');
  window.Plaid = { create: (o) => { opts = o; return { open }; } };
});
afterEach(() => { cleanup(); vi.resetAllMocks(); delete window.Plaid; });

const start = async () => { fireEvent.click(await screen.findByRole('button', { name: 'Connect a bank' })); await waitFor(() => expect(open).toHaveBeenCalled()); };

it('shows the empty state and opens Link with the server-issued token', async () => {
  render(<ConnectionsPanel />);
  expect(await screen.findByText('No bank connections yet')).toBeTruthy();
  await start();
  expect(opts.token).toBe('link-sandbox-1');
});

it('posts only the public token on success and lists the institution', async () => {
  vi.mocked(api.connect).mockResolvedValue({ connection: conn, created: true });
  render(<ConnectionsPanel />);
  await start();
  opts.onSuccess('public-sandbox-1');
  expect(await screen.findByText('Test Bank connected.')).toBeTruthy();
  expect(api.connect).toHaveBeenCalledWith('public-sandbox-1');
  expect(screen.getByRole('list', { name: 'Connected institutions' }).children).toHaveLength(1);
});

it('duplicate connect keeps one row', async () => {
  vi.mocked(api.listConnections).mockResolvedValue([conn]);
  vi.mocked(api.connect).mockResolvedValue({ connection: conn, created: false });
  render(<ConnectionsPanel />);
  await screen.findByText(/ACTIVE/);
  await start();
  opts.onSuccess('public-sandbox-2');
  expect(await screen.findByText('Test Bank was already connected.')).toBeTruthy();
  expect(screen.getByRole('list').children).toHaveLength(1);
});

it('cancel sends nothing; a Link error shows a safe message', async () => {
  render(<ConnectionsPanel />);
  await start();
  opts.onExit(null);
  expect(await screen.findByText(/cancelled/)).toBeTruthy();
  await start();
  opts.onExit({ error_message: 'provider detail' });
  expect(await screen.findByRole('alert')).toBeTruthy();
  expect(screen.queryByText(/provider detail/)).toBeNull();
  expect(api.connect).not.toHaveBeenCalled();
});

it('shows a link-token error without opening Link', async () => {
  vi.mocked(api.createLinkToken).mockRejectedValue(new Error('Too many attempts. Wait a minute and try again.'));
  render(<ConnectionsPanel />);
  fireEvent.click(await screen.findByRole('button', { name: 'Connect a bank' }));
  expect(await screen.findByText(/Too many attempts/)).toBeTruthy();
  expect(open).not.toHaveBeenCalled();
});

it('disconnects after the API confirms', async () => {
  vi.mocked(api.listConnections).mockResolvedValue([conn]);
  vi.mocked(api.disconnect).mockResolvedValue();
  render(<ConnectionsPanel />);
  fireEvent.click(await screen.findByRole('button', { name: 'Disconnect Test Bank' }));
  expect(await screen.findByText('No bank connections yet')).toBeTruthy();
  expect(api.disconnect).toHaveBeenCalledWith('c1');
});

it('shows a load error with retry', async () => {
  vi.mocked(api.listConnections).mockRejectedValueOnce(new Error('down')).mockResolvedValue([]);
  render(<ConnectionsPanel />);
  expect(await screen.findByText('down')).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
  expect(await screen.findByText('No bank connections yet')).toBeTruthy();
});
