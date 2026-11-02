-- VP-015c: token-gated participant join and whole-item claims. A claim is the existing
-- allocations row (unique per item), so the database picks the single winner of a race.
-- Both functions re-validate the session under the receipt row lock and bump receipts.version,
-- so a stale owner save conflicts instead of overwriting claims. Draft shares go stale and are cleared.
create function public.split_session_lock(p_hash text,p_now timestamptz) returns public.receipts
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;rid uuid;
begin
 select x.receipt_id into rid from public.split_sessions x
 where x.token_hash=p_hash and x.status='ACTIVE' and p_now<x.expires_at;
 if not found then raise exception 'SESSION_INVALID' using errcode='P0001';end if;
 select * into r from public.receipts where id=rid for update;
 if r.status='FINALIZED' then raise exception 'SESSION_INVALID' using errcode='P0001';end if;
 if r.status<>'NEEDS_REVIEW' or not r.has_owner_draft then raise exception 'SPLIT_NOT_READY' using errcode='P0001';end if;
 return r;
end $$;

create function public.split_join_add(p_hash text,p_now timestamptz,p_name text) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;n integer;pid uuid:=gen_random_uuid();
begin
 r:=public.split_session_lock(p_hash,p_now);
 if length(btrim(coalesce(p_name,''))) not between 1 and 100 then raise exception 'SPLIT_INVALID_NAME' using errcode='P0001';end if;
 select count(*) into n from public.receipt_participants where receipt_id=r.id;
 if n>=40 then raise exception 'SPLIT_FULL' using errcode='P0001';end if;
 insert into public.receipt_participants(id,receipt_id,name,included,position) values(pid,r.id,btrim(p_name),true,n);
 update public.receipts set version=version+1,has_draft_preview=false,updated_at=now() where id=r.id;
 delete from public.receipt_draft_shares where receipt_id=r.id;
 return jsonb_build_object('id',pid,'name',btrim(p_name));
end $$;

create function public.split_claim(p_hash text,p_now timestamptz,p_participant uuid,p_item uuid,p_release boolean) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;amount bigint;holder uuid;
begin
 r:=public.split_session_lock(p_hash,p_now);
 if not exists(select 1 from public.receipt_participants where receipt_id=r.id and id=p_participant and included) then raise exception 'SPLIT_PARTICIPANT_NOT_FOUND' using errcode='P0001';end if;
 select total_price_minor into amount from public.receipt_items where receipt_id=r.id and id=p_item;
 if not found then raise exception 'SPLIT_ITEM_NOT_FOUND' using errcode='P0001';end if;
 select participant_id into holder from public.allocations where receipt_item_id=p_item;
 if p_release then
  if holder=p_participant then delete from public.allocations where receipt_item_id=p_item;end if;
 elsif holder is null then
  insert into public.allocations(receipt_id,receipt_item_id,participant_id,amount_minor) values(r.id,p_item,p_participant,amount);
 elsif holder<>p_participant then raise exception 'SPLIT_CLAIM_CONFLICT' using errcode='P0001';end if;
 update public.receipts set version=version+1,has_draft_preview=false,updated_at=now() where id=r.id;
 delete from public.receipt_draft_shares where receipt_id=r.id;
 return jsonb_build_object('receipt_id',r.id);
end $$;

revoke all on function public.split_session_lock(text,timestamptz),public.split_join_add(text,timestamptz,text),public.split_claim(text,timestamptz,uuid,uuid,boolean) from public,anon,authenticated;
grant execute on function public.split_session_lock(text,timestamptz),public.split_join_add(text,timestamptz,text),public.split_claim(text,timestamptz,uuid,uuid,boolean) to service_role;
