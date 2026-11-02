-- VP-015c: owner deletes a receipt. Child rows (items, participants, claims, sessions, events)
-- cascade. Deletions still count toward the 20/day upload quota so delete cannot be used to
-- bypass the paid-vision limit. A receipt being processed cannot be deleted.
create table public.receipt_deletions (
 owner_id uuid not null,
 deleted_at timestamptz not null default now()
);
create index receipt_deletions_owner on public.receipt_deletions(owner_id,deleted_at);
alter table public.receipt_deletions enable row level security;
revoke all on public.receipt_deletions from anon,authenticated;
grant select,insert on public.receipt_deletions to service_role;

create function public.receipt_delete(p_owner uuid,p_id uuid) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;
begin
 perform pg_advisory_xact_lock(hashtextextended(p_owner::text,808));
 select * into r from public.receipts where owner_id=p_owner and id=p_id for update;
 if not found then raise exception 'RECEIPT_NOT_FOUND';end if;
 if r.status='PROCESSING' then raise exception 'RECEIPT_NOT_REVIEWABLE';end if;
 insert into public.receipt_deletions(owner_id) values(p_owner);
 delete from public.receipts where id=p_id;
 return jsonb_build_object('image_path',r.image_path);
end $$;
revoke all on function public.receipt_delete(uuid,uuid) from public,anon,authenticated;
grant execute on function public.receipt_delete(uuid,uuid) to service_role;

create or replace function public.receipt_create(p_owner uuid,p_hash text,p_mime text) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;rid uuid;
begin
 perform pg_advisory_xact_lock(hashtextextended(p_owner::text,808));
 select * into r from public.receipts where owner_id=p_owner and image_hash=p_hash;
 if found then return public.receipt_document(p_owner,r.id);end if;
 if (select count(*) from public.receipts where owner_id=p_owner and created_at>now()-interval '1 day')
  +(select count(*) from public.receipt_deletions where owner_id=p_owner and deleted_at>now()-interval '1 day')>=20
 or (select count(*) from public.receipts where owner_id=p_owner and status in ('DRAFT','PROCESSING'))>=5 then raise exception 'RECEIPT_QUOTA';end if;
 rid:=gen_random_uuid();
 insert into public.receipts(id,owner_id,image_path,image_hash,image_mime) values(rid,p_owner,p_owner::text||'/'||rid::text||case when p_mime='image/png' then '.png' else '.jpg' end,p_hash,p_mime) returning * into r;
 return public.receipt_document(p_owner,r.id);
end $$;
