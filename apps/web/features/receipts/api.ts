import { authClient } from '../../lib/auth/client';

async function authorizedFetch(path: string, init: RequestInit): Promise<Response> {
  const { data, error } = await authClient().auth.getSession();
  if (error || !data.session) throw new Error('Sign in again to access receipts.');

  return fetch(`${process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'}/api/v1/receipts${path}`, {
    ...init,
    headers: { ...init.headers, Authorization: `Bearer ${data.session.access_token}` },
    credentials: 'omit', cache: 'no-store', signal: init.signal ?? AbortSignal.timeout(20000),
  });
}

export async function receiptRequest<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await authorizedFetch(path, init);
  const body = await response.json();
  if (!response.ok) {
    const messages: Record<string, string> = {
      RECEIPT_CONFLICT: 'This receipt changed. Reload it before editing again; your unsaved changes have not been applied.',
      RECEIPT_SETUP_REQUIRED: 'Receipt processing is not configured. Ask the project owner to complete the server setup.',
      RATE_LIMITED: 'Too many requests. Wait a minute and try again.',
    };
    const code = body.error?.code as string;
    // Receipt backend errors are bounded safe messages; never render raw upstream output.
    throw new Error(messages[code] ?? (typeof body.error?.message === 'string' ? body.error.message.slice(0, 500) : 'Receipt request failed. Try again.'));
  }
  return body as T;
}
export const postJSON = (value: unknown): RequestInit => ({ method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(value) });

export async function receiptImage(id: string): Promise<Blob> {
  const response = await authorizedFetch(`/${id}/image`, {});
  if (!response.ok) throw new Error('The private receipt image could not be loaded. Try again.');
  return response.blob();
}
