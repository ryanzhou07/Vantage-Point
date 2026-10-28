-- Receipt-owned temporary split sessions. Only a token hash is stored. Expiry is
-- decided by the backend-supplied clock (p_now) inside these service-only functions.
create table public.split_sessions (
 id uuid primary key default gen_random_uuid(),
 receipt_id uuid not null references public.receipts(id) on delete cascade,
 token_hash text not null unique check (token_hash ~ '^[a-f0-9]{64}$'),
 status text not null default 'ACTIVE' check (status in ('ACTIVE','INVALIDATED')),
 created_at timestamptz not null default now(),
 expires_at timestamptz not null,
 invalidated_at timestamptz,
 check (expires_at > created_at),
 check ((status='INVALIDATED') = (invalidated_at is not null))
);
create unique index split_sessions_one_active on public.split_sessions(receipt_id) where status='ACTIVE';
alter table public.split_sessions enable row level security;
revoke all on public.split_sessions from anon,authenticated;
grant select (id,receipt_id,status,created_at,expires_at,invalidated_at) on public.split_sessions to authenticated;
grant all on public.split_sessions to service_role;
create policy split_session_owner_read on public.split_sessions for select to authenticated
 using (exists(select 1 from public.receipts r where r.id=receipt_id and r.owner_id=(select auth.uid())));

create function public.split_session_json(s public.split_sessions) returns jsonb
language sql immutable set search_path='' as $$
 select jsonb_build_object('id',s.id,'receipt_id',s.receipt_id,'status',s.status,'created_at',s.created_at,'expires_at',s.expires_at,'invalidated_at',s.invalidated_at) $$;

create function public.split_session_open(p_owner uuid,p_receipt uuid,p_hash text,p_now timestamptz,p_ttl_seconds integer) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare s public.split_sessions;
begin
 if p_ttl_seconds not between 60 and 3600 then raise exception 'SESSION_INVALID_TTL' using errcode='P0001'; end if;
 perform 1 from public.receipts where id=p_receipt and owner_id=p_owner for update;
 if not found then raise exception 'RECEIPT_NOT_FOUND' using errcode='P0001'; end if;
 update public.split_sessions set status='INVALIDATED',invalidated_at=p_now where receipt_id=p_receipt and status='ACTIVE';
 insert into public.split_sessions(receipt_id,token_hash,created_at,expires_at)
 values(p_receipt,p_hash,p_now,p_now+make_interval(secs=>p_ttl_seconds)) returning * into s;
 return public.split_session_json(s);
end $$;

create function public.split_session_status(p_owner uuid,p_receipt uuid) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare s public.split_sessions;
begin
 perform 1 from public.receipts where id=p_receipt and owner_id=p_owner;
 if not found then raise exception 'RECEIPT_NOT_FOUND' using errcode='P0001'; end if;
 select * into s from public.split_sessions where receipt_id=p_receipt order by created_at desc,id limit 1;
 if not found then raise exception 'SESSION_NOT_FOUND' using errcode='P0001'; end if;
 return public.split_session_json(s);
end $$;

create function public.split_session_close(p_owner uuid,p_receipt uuid,p_now timestamptz) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare s public.split_sessions;
begin
 update public.split_sessions x set status='INVALIDATED',invalidated_at=p_now
 from public.receipts r where x.receipt_id=p_receipt and x.status='ACTIVE' and r.id=x.receipt_id and r.owner_id=p_owner
 returning x.* into s;
 if not found then raise exception 'SESSION_NOT_FOUND' using errcode='P0001'; end if;
 return public.split_session_json(s);
end $$;

create function public.split_session_extend(p_owner uuid,p_receipt uuid,p_now timestamptz,p_ttl_seconds integer) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare s public.split_sessions;
begin
 if p_ttl_seconds not between 60 and 3600 then raise exception 'SESSION_INVALID_TTL' using errcode='P0001'; end if;
 update public.split_sessions x set expires_at=p_now+make_interval(secs=>p_ttl_seconds)
 from public.receipts r where x.receipt_id=p_receipt and x.status='ACTIVE' and x.expires_at>p_now and r.id=x.receipt_id and r.owner_id=p_owner
 returning x.* into s;
 if not found then raise exception 'SESSION_NOT_FOUND' using errcode='P0001'; end if;
 return public.split_session_json(s);
end $$;

-- Participant-token check: valid only while ACTIVE, before expires_at, and not finalized.
create function public.split_session_check(p_hash text,p_now timestamptz) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare s public.split_sessions;
begin
 select x.* into s from public.split_sessions x join public.receipts r on r.id=x.receipt_id
 where x.token_hash=p_hash and x.status='ACTIVE' and p_now<x.expires_at and r.status<>'FINALIZED';
 if not found then raise exception 'SESSION_INVALID' using errcode='P0001'; end if;
 return public.split_session_json(s)||jsonb_build_object('token_hash',s.token_hash);
end $$;

revoke all on function public.split_session_json(public.split_sessions),public.split_session_open(uuid,uuid,text,timestamptz,integer),public.split_session_status(uuid,uuid),public.split_session_close(uuid,uuid,timestamptz),public.split_session_extend(uuid,uuid,timestamptz,integer),public.split_session_check(text,timestamptz) from public,anon,authenticated;
grant execute on function public.split_session_json(public.split_sessions),public.split_session_open(uuid,uuid,text,timestamptz,integer),public.split_session_status(uuid,uuid),public.split_session_close(uuid,uuid,timestamptz),public.split_session_extend(uuid,uuid,timestamptz,integer),public.split_session_check(text,timestamptz) to service_role;
