import type { SupabaseClient } from '@supabase/supabase-js';

export type Profile = {
  id: string;
  display_name: string;
  email: string;
  avatar_url: string;
  created_at: string;
  updated_at: string;
};

export function callbackURL(origin: string) {
  const parsed = new URL(origin);
  const localHTTP = parsed.protocol === 'http:' && ['localhost', '127.0.0.1'].includes(parsed.hostname);

  if (parsed.origin !== origin || (parsed.protocol !== 'https:' && !localHTTP)) {
    throw new Error('Invalid application origin.');
  }

  return `${origin}/auth/callback`;
}

export async function signIn(client: SupabaseClient, origin: string) {
  const { error } = await client.auth.signInWithOAuth({
    provider: 'google',
    options: { redirectTo: callbackURL(origin) },
  });

  if (error) throw new Error('Google sign-in could not start. Please try again.');
}

export async function completeCallback(client: SupabaseClient, location: string) {
  const url = new URL(location);

  // Destination is always /dashboard; no caller-controlled redirect is accepted.
  if (url.searchParams.has('error') || url.searchParams.has('error_description')) {
    throw new Error('Google sign-in was canceled or could not complete. Please try again.');
  }

  const codes = url.searchParams.getAll('code');
  if (codes.length !== 1 || !codes[0] || codes[0].length > 2048) {
    throw new Error('The sign-in callback is invalid. Please try again.');
  }

  const { error } = await client.auth.exchangeCodeForSession(codes[0]);
  if (error) throw new Error('Your sign-in link expired or could not be verified. Please try again.');
}

export async function signOut(client: SupabaseClient) {
  const { error } = await client.auth.signOut({ scope: 'local' });
  if (error) throw new Error('Sign-out could not complete. Please try again.');
}

export async function loadProfile(client: SupabaseClient, api: string, signal: AbortSignal) {
  const { data, error } = await client.auth.getSession();
  if (error) throw new Error('Your session could not be loaded. Please sign in again.');
  if (!data.session) return null;

  const response = await fetch(`${api}/api/v1/me`, {
    headers: { Authorization: `Bearer ${data.session.access_token}` },
    cache: 'no-store',
    credentials: 'omit',
    signal: AbortSignal.any([signal, AbortSignal.timeout(5000)]),
  });

  if (!response.ok) {
    throw new Error(response.status === 401
      ? 'Your session expired. Please sign in again.'
      : 'Your profile could not be loaded. Please try again.');
  }

  const body = await response.json();
  return body.profile as Profile;
}
