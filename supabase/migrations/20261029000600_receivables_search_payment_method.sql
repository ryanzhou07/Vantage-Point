-- VP-021: participant payment method (Venmo/Zelle), snapshotted onto receivables at finalization.
-- VP-023: indexed, keyset-paginated receivables search over snapshot columns on receivables.

alter table public.receipt_participants add column payment_method text check(payment_method in ('VENMO','ZELLE'));
alter table public.receivables
 add column payment_method text check(payment_method in ('VENMO','ZELLE')),
 add column participant_name text not null default '' check(char_length(participant_name)<=100),
 add column merchant_name text not null default '' check(char_length(merchant_name)<=200),
 add column receipt_date date;
-- Finalized receivables are immutable, so the snapshot cannot go stale.
update public.receivables c set participant_name=p.name,merchant_name=r.merchant_name,receipt_date=r.receipt_date
 from public.receipts r,public.receipt_participants p
 where r.id=c.receipt_id and p.receipt_id=c.receipt_id and p.id=c.participant_id;

-- receivables_list is replaced by receivables_search (and its c.*, r.merchant_name join is ambiguous now).
drop function public.receivables_list(uuid);
drop index public.receivables_owner;
create index receivables_owner_page on public.receivables(owner_id,created_at desc,id desc);
create index receivables_owner_status on public.receivables(owner_id,status,created_at desc,id desc);
create index receivables_owner_date on public.receivables(owner_id,receipt_date);
-- shortcut: name/merchant text matching filters rows already narrowed by the owner index (a few thousand per owner), so no trigram/prefix indexes; measured at 20k rows the planner prefers the owner btree. Add pg_trgm GIN indexes if one owner ever holds ~100k receivables.

create or replace function public.receipt_finalize(p_owner uuid,p_id uuid,p_version bigint) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;
begin
 select * into r from public.receipts where owner_id=p_owner and id=p_id for update;
 if not found then raise exception 'RECEIPT_NOT_FOUND';end if;
 if r.version<>p_version or p_version is null then raise exception 'RECEIPT_CONFLICT';end if;
 if r.status<>'NEEDS_REVIEW' or not r.has_owner_draft then raise exception 'RECEIPT_NOT_REVIEWABLE';end if;
 if not r.has_draft_preview
 or exists(select 1 from public.receipt_items i where i.receipt_id=p_id and not exists(select 1 from public.allocations a where a.receipt_item_id=i.id))
 then raise exception 'RECEIPT_INVALID_REVIEW';end if;
 insert into public.receivables(receipt_id,owner_id,participant_id,items_minor,tax_minor,tip_minor,amount_owed_minor,currency,payment_method,participant_name,merchant_name,receipt_date)
 select p_id,p_owner,s.participant_id,s.items_minor,s.tax_minor,s.tip_minor,s.total_minor,r.currency,p.payment_method,p.name,r.merchant_name,r.receipt_date
 from public.receipt_draft_shares s join public.receipt_participants p on p.receipt_id=s.receipt_id and p.id=s.participant_id
 where s.receipt_id=p_id and not p.is_owner and s.total_minor>0;
 update public.split_sessions set status='INVALIDATED',invalidated_at=now() where receipt_id=p_id and status='ACTIVE';
 update public.receipts set status='FINALIZED',finalized_at=now(),version=version+1,updated_at=now() where id=p_id;
 insert into public.receipt_events(receipt_id,event_type,actor_user_id,old_value,new_value)
 values(p_id,'RECEIPT_FINALIZED',p_owner,null,jsonb_build_object('receivables',(select count(*) from public.receivables where receipt_id=p_id),'owed_minor',(select coalesce(sum(amount_owed_minor),0) from public.receivables where receipt_id=p_id)));
 return public.receipt_document(p_owner,p_id);
end $$;

-- A participant picks how they will pay while the split session is open (same trust as claims: the session token).
create function public.split_set_payment_method(p_hash text,p_now timestamptz,p_participant uuid,p_method text) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;
begin
 r:=public.split_session_lock(p_hash,p_now);
 if p_method is null or p_method not in ('VENMO','ZELLE') then raise exception 'SPLIT_INVALID_METHOD' using errcode='P0001';end if;
 update public.receipt_participants set payment_method=p_method where receipt_id=r.id and id=p_participant and included;
 if not found then raise exception 'SPLIT_PARTICIPANT_NOT_FOUND' using errcode='P0001';end if;
 return jsonb_build_object('payment_method',p_method);
end $$;
create function public.split_participant_method(p_hash text,p_now timestamptz,p_participant uuid) returns text
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;
begin
 r:=public.split_session_lock(p_hash,p_now);
 return (select payment_method from public.receipt_participants where receipt_id=r.id and id=p_participant);
end $$;

create function public.receivables_filtered(p_owner uuid,p_status text,p_from date,p_to date,p_q text) returns setof public.receivables
-- No SET clause: it would stop the planner inlining this into the caller's ORDER BY/LIMIT and using the indexes.
language sql stable security invoker as $$
 select c.* from public.receivables c where c.owner_id=p_owner
  and (p_status is null or c.status=p_status)
  and (p_from is null or c.receipt_date>=p_from) and (p_to is null or c.receipt_date<=p_to)
  and (p_q is null or case
   when char_length(btrim(p_q))<3 then lower(c.participant_name) like lower(replace(replace(replace(btrim(p_q),'\','\\'),'%','\%'),'_','\_'))||'%'
     or lower(c.merchant_name) like lower(replace(replace(replace(btrim(p_q),'\','\\'),'%','\%'),'_','\_'))||'%'
   else c.participant_name ilike '%'||replace(replace(replace(btrim(p_q),'\','\\'),'%','\%'),'_','\_')||'%'
     or c.merchant_name ilike '%'||replace(replace(replace(btrim(p_q),'\','\\'),'%','\%'),'_','\_')||'%' end)
$$;

-- Newest first. The sort key never changes when a row is marked paid, so paging stays stable. Keyset cursor {created_at,id}; no OFFSET, no total count.
create function public.receivables_search(p_owner uuid,p_status text,p_from date,p_to date,p_q text,p_limit integer,p_cursor jsonb) returns jsonb
language plpgsql stable security invoker set search_path='' as $$
declare lim integer:=least(greatest(coalesce(p_limit,25),1),50);rows jsonb;more boolean;nxt jsonb;
begin
 select coalesce(jsonb_agg(jsonb_build_object('id',v.id,'receipt_id',v.receipt_id,'merchant',v.merchant_name,'receipt_date',coalesce(v.receipt_date::text,''),
   'participant_id',v.participant_id,'participant_name',v.participant_name,'amount_owed_minor',v.amount_owed_minor,'currency',v.currency,'status',v.status,
   'payment_method',v.payment_method,'paid_at',v.paid_at,'created_at',v.created_at) order by v.created_at desc,v.id desc),'[]'::jsonb) into rows
 from (select f.* from public.receivables_filtered(p_owner,p_status,p_from,p_to,p_q) f
  where p_cursor is null or (f.created_at,f.id)<((p_cursor->>'created_at')::timestamptz,(p_cursor->>'id')::uuid)
  order by f.created_at desc,f.id desc limit lim+1) v;
 more:=jsonb_array_length(rows)>lim;
 if more then rows:=rows-lim;nxt:=jsonb_build_object('created_at',rows->(lim-1)->>'created_at','id',rows->(lim-1)->>'id');end if;
 return jsonb_build_object('receivables',rows,'next',nxt,
  'outstanding_minor',(select coalesce(sum(amount_owed_minor),0) from public.receivables_filtered(p_owner,p_status,p_from,p_to,p_q) f where f.status='OWED'));
end $$;

revoke all on function public.split_set_payment_method(text,timestamptz,uuid,text),public.split_participant_method(text,timestamptz,uuid),public.receivables_filtered(uuid,text,date,date,text),public.receivables_search(uuid,text,date,date,text,integer,jsonb) from public,anon,authenticated;
grant execute on function public.split_set_payment_method(text,timestamptz,uuid,text),public.split_participant_method(text,timestamptz,uuid),public.receivables_filtered(uuid,text,date,date,text),public.receivables_search(uuid,text,date,date,text,integer,jsonb) to service_role;
