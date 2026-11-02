-- Banking-owned connection slice. Credentials are application-encrypted (ADR-009) and never browser-readable.
create table public.institution_connections (
  id uuid primary key,
  user_id uuid not null references public.profiles(id) on delete cascade,
  provider text not null check (provider in ('plaid')),
  provider_connection_id text not null check (length(provider_connection_id) between 1 and 200),
  institution_name text not null default '' check (length(institution_name) <= 200),
  status text not null default 'ACTIVE' check (status in ('ACTIVE','NEEDS_REAUTH','ERROR')),
  encrypted_credential bytea not null check (length(encrypted_credential) between 29 and 4096),
  key_version smallint not null check (key_version between 1 and 255),
  last_sync_attempt_at timestamptz,
  last_successful_sync_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (user_id, provider, provider_connection_id)
);
alter table public.institution_connections enable row level security;
revoke all on public.institution_connections from anon, authenticated;
-- Column grant deliberately excludes encrypted_credential.
grant select (id, user_id, provider, institution_name, status, last_sync_attempt_at, last_successful_sync_at, created_at, updated_at) on public.institution_connections to authenticated;
grant all on public.institution_connections to service_role;
create policy institution_connections_owner_read on public.institution_connections for select to authenticated using ((select auth.uid()) = user_id);

-- Service-role RPCs; the Go API passes the verified user id.
create function public.banking_enroll(p_user uuid, p_id uuid, p_provider text, p_pconn text, p_name text, p_cred text, p_ver int)
returns jsonb language plpgsql security definer set search_path = '' as $$
declare r public.institution_connections; created boolean := true;
begin
  insert into public.institution_connections(id, user_id, provider, provider_connection_id, institution_name, encrypted_credential, key_version)
  values (p_id, p_user, p_provider, p_pconn, left(p_name, 200), decode(p_cred, 'base64'), p_ver)
  on conflict (user_id, provider, provider_connection_id) do nothing
  returning * into r;
  if r.id is null then
    created := false;
    select * into r from public.institution_connections where user_id = p_user and provider = p_provider and provider_connection_id = p_pconn;
  end if;
  return jsonb_build_object('created', created, 'connection', jsonb_build_object('id', r.id, 'provider', r.provider, 'institution_name', r.institution_name, 'status', r.status, 'last_sync_attempt_at', r.last_sync_attempt_at, 'last_successful_sync_at', r.last_successful_sync_at, 'created_at', r.created_at));
end $$;
create function public.banking_connections(p_user uuid, p_id uuid default null)
returns jsonb language sql security definer set search_path = '' as $$
  select coalesce(jsonb_agg(jsonb_build_object('id', id, 'provider', provider, 'institution_name', institution_name, 'status', status, 'last_sync_attempt_at', last_sync_attempt_at, 'last_successful_sync_at', last_successful_sync_at, 'created_at', created_at) order by created_at, id), '[]'::jsonb)
  from public.institution_connections where user_id = p_user and (p_id is null or id = p_id);
$$;
-- Server-only: returns the encrypted blob so the API can decrypt and revoke at the provider.
create function public.banking_credential(p_user uuid, p_id uuid)
returns jsonb language sql security definer set search_path = '' as $$
  select coalesce((select jsonb_build_object('credential', encode(encrypted_credential, 'base64'), 'key_version', key_version) from public.institution_connections where user_id = p_user and id = p_id), 'null'::jsonb);
$$;
create function public.banking_disconnect(p_user uuid, p_id uuid)
returns boolean language plpgsql security definer set search_path = '' as $$
begin
  delete from public.institution_connections where user_id = p_user and id = p_id;
  return found;
end $$;
revoke all on function public.banking_enroll(uuid,uuid,text,text,text,text,int), public.banking_connections(uuid,uuid), public.banking_credential(uuid,uuid), public.banking_disconnect(uuid,uuid) from public, anon, authenticated;
grant execute on function public.banking_enroll(uuid,uuid,text,text,text,text,int), public.banking_connections(uuid,uuid), public.banking_credential(uuid,uuid), public.banking_disconnect(uuid,uuid) to service_role;
