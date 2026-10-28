// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { ConnectionsPanel } from './ConnectionsPanel';
import * as api from './api';

vi.mock('./api');
const conn = { id: 'c1', provider: 'TELLER', institution_name: 'Test Bank', status: 'ACTIVE', created_at: '2026-01-02T00:00:00Z' };
let opts: Parameters<NonNullable<typeof window.TellerConnect>['setup']>[0];
const open = vi.fn();

beforeEach(() => {
  vi.stubEnv('NEXT_PUBLIC_TELLER_APPLICATION_ID', 'app_test');
  vi.mocked(api.listConnections).mockResolvedValue([]);
  window.TellerConnect = { setup: (o) => { opts = o; return { open }; } };
});
afterEach(() => { cleanup(); vi.unstubAllEnvs(); vi.resetAllMocks(); delete window.TellerConnect; });

const start = async () => { fireEvent.click(await screen.findByRole('button', { name: 'Connect a bank' })); await waitFor(() => expect(open).toHaveBeenCalled()); };
const success = { accessToken: 'tok', enrollment: { id: 'enr_1', institution: { name: 'Test Bank' } } };

it('shows the empty state and opens Connect with sandbox config', async () => {
  render(<ConnectionsPanel />);
  expect(await screen.findByText('No bank connections yet')).toBeTruthy();
  await start();
  expect(opts.applicationId).toBe('app_test');
  expect(opts.environment).toBe('sandbox');
});

it('enrolls on success, sends no user id, and lists the institution', async () => {
  vi.mocked(api.enroll).mockResolvedValue({ connection: conn, created: true });
  render(<ConnectionsPanel />);
  await start();
  opts.onSuccess(success);
  expect(await screen.findByText('Test Bank connected.')).toBeTruthy();
  expect(api.enroll).toHaveBeenCalledWith({ enrollment_id: 'enr_1', access_token: 'tok', institution_name: 'Test Bank' });
  expect(screen.getByRole('list', { name: 'Connected institutions' }).children).toHaveLength(1);
});

it('duplicate enroll keeps one row', async () => {
  vi.mocked(api.listConnections).mockResolvedValue([conn]);
  vi.mocked(api.enroll).mockResolvedValue({ connection: conn, created: false });
  render(<ConnectionsPanel />);
  await screen.findByText(/ACTIVE/);
  await start();
  opts.onSuccess(success);
  expect(await screen.findByText('Test Bank was already connected.')).toBeTruthy();
  expect(screen.getByRole('list').children).toHaveLength(1);
});

it('exit and failure send nothing and show safe messages', async () => {
  render(<ConnectionsPanel />);
  await start();
  opts.onExit();
  expect(await screen.findByText(/cancelled/)).toBeTruthy();
  await start();
  opts.onFailure({ raw: 'provider detail' });
  expect(await screen.findByRole('alert')).toBeTruthy();
  expect(screen.queryByText(/provider detail/)).toBeNull();
  expect(api.enroll).not.toHaveBeenCalled();
});

it('disconnects after the API confirms', async () => {
  vi.mocked(api.listConnections).mockResolvedValue([conn]);
  vi.mocked(api.disconnect).mockResolvedValue();
  render(<ConnectionsPanel />);
  fireEvent.click(await screen.findByRole('button', { name: 'Disconnect Test Bank' }));
  expect(await screen.findByText('No bank connections yet')).toBeTruthy();
  expect(api.disconnect).toHaveBeenCalledWith('c1');
});

it('shows a load error with retry and an unconfigured notice', async () => {
  vi.mocked(api.listConnections).mockRejectedValueOnce(new Error('down')).mockResolvedValue([]);
  render(<ConnectionsPanel />);
  expect(await screen.findByText('down')).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
  expect(await screen.findByText('No bank connections yet')).toBeTruthy();
  cleanup();
  vi.stubEnv('NEXT_PUBLIC_TELLER_APPLICATION_ID', '');
  render(<ConnectionsPanel />);
  expect(screen.getByText(/not configured/)).toBeTruthy();
  expect((screen.getByRole('button', { name: 'Connect a bank' }) as HTMLButtonElement).disabled).toBe(true);
});
