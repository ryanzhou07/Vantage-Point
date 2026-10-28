import { authClient } from '../../lib/auth/client';

export type Connection = { id: string; provider: string; institution_name: string; status: string; created_at: string };
export type Enrollment = { enrollment_id: string; access_token: string; institution_name: string };

async function request(method: string, path = '', body?: unknown): Promise<Response> {
  const { data, error } = await authClient().auth.getSession();
  if (error || !data.session) throw new Error('Sign in again to manage bank connections.');
  const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'}/api/v1/banking/connections${path}`, {
    method,
    headers: { Authorization: `Bearer ${data.session.access_token}`, ...(body ? { 'Content-Type': 'application/json' } : {}) },
    body: body ? JSON.stringify(body) : undefined,
    credentials: 'omit', cache: 'no-store', signal: AbortSignal.timeout(35000),
  });
  if (!res.ok) {
    const code = (await res.json().catch(() => ({})))?.error?.code;
    const messages: Record<string, string> = {
      RATE_LIMITED: 'Too many attempts. Wait a minute and try again.',
      INVALID_ENROLLMENT: 'The bank could not verify this connection. Try connecting again.',
      PROVIDER_UNAVAILABLE: 'The bank provider is unavailable right now. Try again shortly.',
    };
    throw new Error(messages[code] ?? 'The bank connection request failed. Try again.');
  }
  return res;
}

export const listConnections = async (): Promise<Connection[]> => (await (await request('GET')).json()).connections;
// 201 = new, 200 = already connected (idempotent).
export const enroll = async (e: Enrollment): Promise<{ connection: Connection; created: boolean }> => {
  const res = await request('POST', '', e);
  return { connection: await res.json(), created: res.status === 201 };
};
export const disconnect = async (id: string): Promise<void> => { await request('DELETE', `/${encodeURIComponent(id)}`); };
