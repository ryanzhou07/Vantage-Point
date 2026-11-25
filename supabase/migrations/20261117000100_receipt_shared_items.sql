-- Shared items: several participants may claim one line; their allocation rows split the line
-- evenly in exact cents (remainder cents go to the lowest participant IDs), always summing to the line.
alter table public.allocations drop constraint allocations_receipt_item_id_key;
alter table public.allocations add constraint allocations_item_participant_key unique(receipt_item_id,participant_id);
drop trigger receipt_whole_line on public.allocations;
drop function public.receipt_allocation_whole_line();

create function public.receipt_item_split(p_item uuid) returns void
language sql security invoker set search_path='' as $$
 update public.allocations a set amount_minor=s.base+(case when s.rn<=s.rem then 1 else 0 end),updated_at=now()
 from (select x.id,row_number() over(order by x.participant_id) rn,
  i.total_price_minor/count(*) over() base,i.total_price_minor%count(*) over() rem
  from public.allocations x join public.receipt_items i on i.id=x.receipt_item_id where x.receipt_item_id=p_item) s
 where a.id=s.id
$$;
revoke all on function public.receipt_item_split(uuid) from public,anon,authenticated;
grant execute on function public.receipt_item_split(uuid) to service_role;

create or replace function public.receipt_document(p_owner uuid,p_id uuid) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;d jsonb;v jsonb;proposal jsonb;
begin
 select * into r from public.receipts where owner_id=p_owner and id=p_id;
 if not found then raise exception 'RECEIPT_NOT_FOUND';end if;
 proposal:=r.proposal;
 if proposal is not null then
  proposal:=jsonb_set(proposal,'{items}',coalesce((select jsonb_agg(jsonb_set(line,'{id}',to_jsonb(public.receipt_internal_id(r.id,'item',coalesce(line->>'id',n::text))::text)) order by n)
  from jsonb_array_elements(proposal->'items') with ordinality as lines(line,n)),'[]'::jsonb));
 end if;
 if r.has_owner_draft then
  d:=jsonb_build_object('merchant',r.merchant_name,'date',coalesce(r.receipt_date::text,''),'currency',r.currency,
  'subtotal_minor',r.subtotal_minor,'tax_minor',r.tax_minor,'tip_minor',r.tip_minor,'discount_minor',r.discount_minor,'total_minor',r.total_minor,
  'tax_method',r.tax_allocation_method,'tip_method',r.tip_allocation_method,
  'items',coalesce((select jsonb_agg(jsonb_build_object('id',i.id,'name',i.description,'quantity',i.quantity,'amount_minor',i.total_price_minor,'participant_ids',coalesce((select jsonb_agg(a.participant_id order by a.participant_id) from public.allocations a where a.receipt_item_id=i.id),'[]'::jsonb)) order by i.position)
    from public.receipt_items i where i.receipt_id=r.id),'[]'::jsonb),
  'participants',coalesce((select jsonb_agg(jsonb_build_object('id',p.id,'name',p.name,'included',p.included,'is_owner',p.is_owner) order by p.position) from public.receipt_participants p where p.receipt_id=r.id),'[]'::jsonb));
 end if;
 if r.has_draft_preview then
  v:=jsonb_build_object('draft',true,'total_minor',r.total_minor,'shares',coalesce((select jsonb_agg(jsonb_build_object('participant_id',s.participant_id,'items_minor',s.items_minor,'tax_minor',s.tax_minor,'tip_minor',s.tip_minor,'total_minor',s.total_minor) order by s.participant_id) from public.receipt_draft_shares s where s.receipt_id=r.id),'[]'::jsonb));
 end if;
 return jsonb_build_object('id',r.id,'owner_id',r.owner_id,'image_path',r.image_path,'image_hash',r.image_hash,'image_mime',r.image_mime,'status',r.status,'version',r.version,'proposal',proposal,'draft',d,'preview',v,'error_code',r.error_code,'retries',r.retries,'created_at',r.created_at);
end $$;

create or replace function public.receipt_save(p_owner uuid,p_id uuid,p_version bigint,p_draft jsonb,p_preview jsonb) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare old_owner public.receipt_participants;r public.receipts;line jsonb;person jsonb;share jsonb;n integer;old_document jsonb;item_id uuid;person_id uuid;
begin
 select * into r from public.receipts where owner_id=p_owner and id=p_id for update;
 if not found then raise exception 'RECEIPT_NOT_FOUND';end if;
 if r.version<>p_version or p_version is null then raise exception 'RECEIPT_CONFLICT';end if;
 if r.status<>'NEEDS_REVIEW' then raise exception 'RECEIPT_NOT_REVIEWABLE';end if;
 if p_draft is null or jsonb_typeof(p_draft->'items')<>'array' or jsonb_array_length(p_draft->'items') not between 1 and 200
 or jsonb_typeof(p_draft->'participants')<>'array' or jsonb_array_length(p_draft->'participants')>40 then raise exception 'RECEIPT_INVALID_REVIEW';end if;
 old_document:=public.receipt_document(p_owner,p_id);
 select * into old_owner from public.receipt_participants where receipt_id=p_id and is_owner;
 delete from public.receipt_draft_shares where receipt_id=p_id;
 delete from public.allocations where receipt_id=p_id;
 -- Before collaboration, replace the whole owner draft under its row lock.
 -- Stable UUIDs persist; no guest claims/submissions/receivables exist in this slice.
 delete from public.receipt_items where receipt_id=p_id;
 delete from public.receipt_participants where receipt_id=p_id;
 n:=0;
 for person in select value from jsonb_array_elements(p_draft->'participants') loop
  person_id:=public.receipt_internal_id(p_id,'participant',person->>'id');
  insert into public.receipt_participants(id,receipt_id,name,included,position,is_owner) values(person_id,p_id,person->>'name',(person->>'included')::boolean,n,coalesce((person->>'is_owner')::boolean,false));
  n:=n+1;
 end loop;
 -- The owner's own participant is locked: it cannot be removed, renamed or excluded once saved.
 if old_owner.id is not null and not exists(select 1 from public.receipt_participants where receipt_id=p_id and id=old_owner.id and is_owner and included and name=old_owner.name) then raise exception 'RECEIPT_INVALID_REVIEW';end if;
 n:=0;
 for line in select value from jsonb_array_elements(p_draft->'items') loop
  item_id:=public.receipt_internal_id(p_id,'item',line->>'id');
  insert into public.receipt_items(id,receipt_id,description,quantity,total_price_minor,position) values(item_id,p_id,line->>'name',(line->>'quantity')::integer,(line->>'amount_minor')::bigint,n);
  if jsonb_typeof(line->'participant_ids')='array' then
   for person in select value from jsonb_array_elements(line->'participant_ids') loop
    insert into public.allocations(receipt_id,receipt_item_id,participant_id,amount_minor) values(p_id,item_id,public.receipt_internal_id(p_id,'participant',person#>>'{}'),0);
   end loop;
   perform public.receipt_item_split(item_id);
  end if;
  n:=n+1;
 end loop;
 update public.receipts set merchant_name=p_draft->>'merchant',receipt_date=nullif(p_draft->>'date','')::date,currency=p_draft->>'currency',
 subtotal_minor=(p_draft->>'subtotal_minor')::bigint,tax_minor=(p_draft->>'tax_minor')::bigint,tip_minor=(p_draft->>'tip_minor')::bigint,
 discount_minor=(p_draft->>'discount_minor')::bigint,total_minor=(p_draft->>'total_minor')::bigint,
 tax_allocation_method=p_draft->>'tax_method',tip_allocation_method=p_draft->>'tip_method',
 has_owner_draft=true,has_draft_preview=(p_preview is not null),version=version+1,updated_at=now() where id=p_id returning * into r;
 if p_preview is not null then
  if r.discount_minor<>0 or r.subtotal_minor+r.tax_minor+r.tip_minor<>r.total_minor
  or (select sum(total_price_minor) from public.receipt_items where receipt_id=p_id)<>r.subtotal_minor
  or exists(select 1 from public.receipt_items i where i.receipt_id=p_id and not exists(select 1 from public.allocations a where a.receipt_item_id=i.id))
  or r.tax_allocation_method='' or r.tip_allocation_method='' then raise exception 'RECEIPT_INVALID_REVIEW';end if;
  for share in select value from jsonb_array_elements(p_preview->'shares') loop
   insert into public.receipt_draft_shares(receipt_id,participant_id,items_minor,tax_minor,tip_minor,total_minor)
   values(p_id,public.receipt_internal_id(p_id,'participant',share->>'participant_id'),(share->>'items_minor')::bigint,(share->>'tax_minor')::bigint,(share->>'tip_minor')::bigint,(share->>'total_minor')::bigint);
  end loop;
  if (select count(*) from public.receipt_draft_shares where receipt_id=p_id)<>(select count(*) from public.receipt_participants where receipt_id=p_id)
  or (select coalesce(sum(tax_minor),0) from public.receipt_draft_shares where receipt_id=p_id)<>r.tax_minor
  or (select coalesce(sum(tip_minor),0) from public.receipt_draft_shares where receipt_id=p_id)<>r.tip_minor
  or (select coalesce(sum(total_minor),0) from public.receipt_draft_shares where receipt_id=p_id)<>r.total_minor
  or exists(select 1 from public.receipt_draft_shares s where s.receipt_id=p_id and s.items_minor<>(select coalesce(sum(a.amount_minor),0) from public.allocations a where a.receipt_id=p_id and a.participant_id=s.participant_id))
  or exists(select 1 from public.receipt_participants p join public.allocations a on a.participant_id=p.id where p.receipt_id=p_id and not p.included)
  then raise exception 'RECEIPT_INVALID_REVIEW';end if;
 end if;
 insert into public.receipt_events(receipt_id,event_type,actor_user_id,old_value,new_value) values(p_id,'OWNER_DRAFT_SAVED',p_owner,old_document->'draft',public.receipt_document(p_owner,p_id)->'draft');
 return public.receipt_document(p_owner,p_id);
end $$;

create or replace function public.split_claim(p_hash text,p_now timestamptz,p_participant uuid,p_item uuid,p_release boolean) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;
begin
 r:=public.split_session_lock(p_hash,p_now);
 if not exists(select 1 from public.receipt_participants where receipt_id=r.id and id=p_participant and included) then raise exception 'SPLIT_PARTICIPANT_NOT_FOUND' using errcode='P0001';end if;
 if not exists(select 1 from public.receipt_items where receipt_id=r.id and id=p_item) then raise exception 'SPLIT_ITEM_NOT_FOUND' using errcode='P0001';end if;
 if p_release then
  delete from public.allocations where receipt_item_id=p_item and participant_id=p_participant;
 else
  insert into public.allocations(receipt_id,receipt_item_id,participant_id,amount_minor) values(r.id,p_item,p_participant,0) on conflict do nothing;
 end if;
 perform public.receipt_item_split(p_item);
 update public.receipts set version=version+1,has_draft_preview=false,updated_at=now() where id=r.id;
 delete from public.receipt_draft_shares where receipt_id=r.id;
 return jsonb_build_object('receipt_id',r.id);
end $$;

-- Deferred to commit so a claim can add a row and re-split in the same transaction.
create function public.receipt_allocation_sum_check() returns trigger
language plpgsql security invoker set search_path='' as $$
declare item uuid:=coalesce(new.receipt_item_id,old.receipt_item_id);total bigint;allocated bigint;n integer;
begin
 select total_price_minor into total from public.receipt_items where id=item;
 if not found then return null;end if;
 select coalesce(sum(amount_minor),0),count(*) into allocated,n from public.allocations where receipt_item_id=item;
 if n>0 and allocated<>total then raise exception 'RECEIPT_INVALID_ALLOCATION' using errcode='23514';end if;
 return null;
end $$;
revoke all on function public.receipt_allocation_sum_check() from public,anon,authenticated;
grant execute on function public.receipt_allocation_sum_check() to service_role;
create constraint trigger receipt_allocation_sum after insert or update or delete on public.allocations deferrable initially deferred for each row execute function public.receipt_allocation_sum_check();
