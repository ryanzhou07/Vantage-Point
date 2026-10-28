-- VP-019: finalization snapshot, receivables (manual paid tracking) and history.
alter table public.receipts drop constraint receipts_status_check;
alter table public.receipts add constraint receipts_status_check check(status in ('DRAFT','PROCESSING','NEEDS_REVIEW','PROCESSING_FAILED','FINALIZED'));
alter table public.receipts add column finalized_at timestamptz;
alter table public.receipt_events drop constraint receipt_events_event_type_check;
alter table public.receipt_events add constraint receipt_events_event_type_check check(event_type in ('OWNER_DRAFT_SAVED','NORMALIZED_DRAFT_MIGRATED','RECEIPT_FINALIZED','RECEIVABLE_PAID','RECEIVABLE_UNPAID'));

create table public.receivables (
 id uuid primary key default gen_random_uuid(),
 receipt_id uuid not null references public.receipts(id) on delete restrict,
 owner_id uuid not null references public.profiles(id),
 participant_id uuid not null,
 items_minor bigint not null check(items_minor between 0 and 100000000),
 tax_minor bigint not null check(tax_minor between 0 and 100000000),
 tip_minor bigint not null check(tip_minor between 0 and 100000000),
 amount_owed_minor bigint not null check(amount_owed_minor>0 and amount_owed_minor=items_minor+tax_minor+tip_minor),
 currency text not null default 'USD' check(currency='USD'),
 status text not null default 'OWED' check(status in ('OWED','PAID')),
 created_at timestamptz not null default now(),
 paid_at timestamptz,
 updated_at timestamptz not null default now(),
 check((status='PAID')=(paid_at is not null)),
 unique(receipt_id,participant_id),
 foreign key(receipt_id,participant_id) references public.receipt_participants(receipt_id,id) on delete restrict
);
create index receivables_owner on public.receivables(owner_id,status);
alter table public.receivables enable row level security;
revoke all on public.receivables from anon,authenticated;
grant select on public.receivables to authenticated;
grant all on public.receivables to service_role;
create policy receivable_owner_read on public.receivables for select to authenticated using(owner_id=(select auth.uid()));

create function public.receipt_finalize(p_owner uuid,p_id uuid,p_version bigint) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;
begin
 select * into r from public.receipts where owner_id=p_owner and id=p_id for update;
 if not found then raise exception 'RECEIPT_NOT_FOUND';end if;
 if r.version<>p_version or p_version is null then raise exception 'RECEIPT_CONFLICT';end if;
 if r.status<>'NEEDS_REVIEW' or not r.has_owner_draft then raise exception 'RECEIPT_NOT_REVIEWABLE';end if;
 -- The saved preview is cleared by every join, claim and edit, so its presence means it is current.
 if not r.has_draft_preview
 or exists(select 1 from public.receipt_items i where i.receipt_id=p_id and not exists(select 1 from public.allocations a where a.receipt_item_id=i.id))
 then raise exception 'RECEIPT_INVALID_REVIEW';end if;
 insert into public.receivables(receipt_id,owner_id,participant_id,items_minor,tax_minor,tip_minor,amount_owed_minor,currency)
 select p_id,p_owner,s.participant_id,s.items_minor,s.tax_minor,s.tip_minor,s.total_minor,r.currency
 from public.receipt_draft_shares s join public.receipt_participants p on p.receipt_id=s.receipt_id and p.id=s.participant_id
 where s.receipt_id=p_id and not p.is_owner and s.total_minor>0;
 update public.split_sessions set status='INVALIDATED',invalidated_at=now() where receipt_id=p_id and status='ACTIVE';
 update public.receipts set status='FINALIZED',finalized_at=now(),version=version+1,updated_at=now() where id=p_id;
 insert into public.receipt_events(receipt_id,event_type,actor_user_id,old_value,new_value)
 values(p_id,'RECEIPT_FINALIZED',p_owner,null,jsonb_build_object('receivables',(select count(*) from public.receivables where receipt_id=p_id),'owed_minor',(select coalesce(sum(amount_owed_minor),0) from public.receivables where receipt_id=p_id)));
 return public.receipt_document(p_owner,p_id);
end $$;

create function public.receivables_list(p_owner uuid) returns jsonb
language sql security invoker set search_path='' as $$
 select coalesce(jsonb_agg(jsonb_build_object('id',v.id,'receipt_id',v.receipt_id,'merchant',v.merchant_name,'receipt_date',coalesce(v.receipt_date::text,''),
  'participant_id',v.participant_id,'participant_name',v.name,'amount_owed_minor',v.amount_owed_minor,'currency',v.currency,'status',v.status,'paid_at',v.paid_at,'created_at',v.created_at)
  order by (v.status='PAID'),v.created_at desc,v.id),'[]'::jsonb)
 from (select c.*,r.merchant_name,r.receipt_date,p.name from public.receivables c join public.receipts r on r.id=c.receipt_id
  join public.receipt_participants p on p.receipt_id=c.receipt_id and p.id=c.participant_id where c.owner_id=p_owner order by c.created_at desc limit 500) v
$$;

create function public.receivable_set_paid(p_owner uuid,p_id uuid,p_paid boolean) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare c public.receivables;
begin
 update public.receivables set status=case when p_paid then 'PAID' else 'OWED' end,paid_at=case when p_paid then now() else null end,updated_at=now()
 where id=p_id and owner_id=p_owner returning * into c;
 if not found then raise exception 'RECEIPT_NOT_FOUND';end if;
 insert into public.receipt_events(receipt_id,event_type,actor_user_id,old_value,new_value)
 values(c.receipt_id,case when p_paid then 'RECEIVABLE_PAID' else 'RECEIVABLE_UNPAID' end,p_owner,null,jsonb_build_object('receivable_id',c.id,'amount_owed_minor',c.amount_owed_minor));
 return jsonb_build_object('id',c.id,'status',c.status,'paid_at',c.paid_at);
end $$;

-- A finalized receipt is history: it can no longer be deleted.
create or replace function public.receipt_delete(p_owner uuid,p_id uuid) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;
begin
 perform pg_advisory_xact_lock(hashtextextended(p_owner::text,808));
 select * into r from public.receipts where owner_id=p_owner and id=p_id for update;
 if not found then raise exception 'RECEIPT_NOT_FOUND';end if;
 if r.status in ('PROCESSING','FINALIZED') then raise exception 'RECEIPT_NOT_REVIEWABLE';end if;
 insert into public.receipt_deletions(owner_id) values(p_owner);
 delete from public.receipts where id=p_id;
 return jsonb_build_object('image_path',r.image_path);
end $$;

revoke all on function public.receipt_finalize(uuid,uuid,bigint),public.receivables_list(uuid),public.receivable_set_paid(uuid,uuid,boolean) from public,anon,authenticated;
grant execute on function public.receipt_finalize(uuid,uuid,bigint),public.receivables_list(uuid),public.receivable_set_paid(uuid,uuid,boolean) to service_role;
