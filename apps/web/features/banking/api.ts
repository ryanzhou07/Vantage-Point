import { authClient } from '../../lib/auth/client';

export type Connection = { id: string; provider: string; institution_name: string; status: string; created_at: string };

const MESSAGES: Record<string, string> = {
  RATE_LIMITED: 'Too many attempts. Wait a minute and try again.',
  INVALID_TOKEN: 'The bank could not verify this connection. Try connecting again.',
  PROVIDER_UNAVAILABLE: 'The bank provider is unavailable right now. Try again shortly.',
  UNAVAILABLE: 'Bank connections are not available right now.',
};

async function request(method: string, path: string, body?: unknown): Promise<Response> {
  const { data, error } = await authClient().auth.getSession();
  if (error || !data.session) throw new Error('Sign in again to manage bank connections.');
  const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'}/api/v1/banking/${path}`, {
    method,
    headers: { Authorization: `Bearer ${data.session.access_token}`, ...(body ? { 'Content-Type': 'application/json' } : {}) },
    body: body ? JSON.stringify(body) : undefined,
    credentials: 'omit', cache: 'no-store', signal: AbortSignal.timeout(35000),
  });
  if (!res.ok) {
    const code = (await res.json().catch(() => ({})))?.error?.code;
    throw new Error(MESSAGES[code] ?? 'The bank connection request failed. Try again.');
  }
  return res;
}

export const listConnections = async (): Promise<Connection[]> => (await (await request('GET', 'connections')).json()).connections;
export const createLinkToken = async (): Promise<string> => (await (await request('POST', 'link-token')).json()).link_token;
// 201 = new, 200 = already connected (idempotent). The public token is single-use and not kept.
export const connect = async (publicToken: string): Promise<{ connection: Connection; created: boolean }> => {
  const res = await request('POST', 'connections', { public_token: publicToken });
  return { connection: await res.json(), created: res.status === 201 };
};
export const disconnect = async (id: string): Promise<void> => { await request('DELETE', `connections/${encodeURIComponent(id)}`); };
