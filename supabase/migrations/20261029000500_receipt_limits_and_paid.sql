-- VP-027: lower scan limits (5/day, 20/30 days, 2 in progress, 1 retry). Deletions still count.
-- VP-020: marking paid/unpaid is idempotent: repeating it changes nothing and writes no audit event.
create or replace function public.receipt_create(p_owner uuid,p_hash text,p_mime text) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;rid uuid;
begin
 perform pg_advisory_xact_lock(hashtextextended(p_owner::text,808));
 select * into r from public.receipts where owner_id=p_owner and image_hash=p_hash;
 if found then return public.receipt_document(p_owner,r.id);end if;
 if (select count(*) from public.receipts where owner_id=p_owner and created_at>now()-interval '1 day')
  +(select count(*) from public.receipt_deletions where owner_id=p_owner and deleted_at>now()-interval '1 day')>=5
 or (select count(*) from public.receipts where owner_id=p_owner and created_at>now()-interval '30 days')
  +(select count(*) from public.receipt_deletions where owner_id=p_owner and deleted_at>now()-interval '30 days')>=20
 or (select count(*) from public.receipts where owner_id=p_owner and status in ('DRAFT','PROCESSING'))>=2 then raise exception 'RECEIPT_QUOTA';end if;
 rid:=gen_random_uuid();
 insert into public.receipts(id,owner_id,image_path,image_hash,image_mime) values(rid,p_owner,p_owner::text||'/'||rid::text||case when p_mime='image/png' then '.png' else '.jpg' end,p_hash,p_mime) returning * into r;
 return public.receipt_document(p_owner,r.id);
end $$;

create or replace function public.receipt_retry(p_owner uuid,p_id uuid,p_version bigint) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;
begin
 select * into r from public.receipts where owner_id=p_owner and id=p_id for update;
 if not found then raise exception 'RECEIPT_NOT_FOUND';end if;
 if r.version<>p_version or p_version is null then raise exception 'RECEIPT_CONFLICT';end if;
 if r.status<>'PROCESSING_FAILED' or r.retries>=1 then raise exception 'RECEIPT_RETRY_LIMIT';end if;
 update public.receipts set status='PROCESSING',error_code=null,retries=retries+1,version=version+1,updated_at=now() where id=p_id;
 update public.receipt_vision_jobs set state='QUEUED',available_at=now()+interval '5 seconds',lease_until=null,lease_token=null where receipt_id=p_id;
 return public.receipt_document(p_owner,p_id);
end $$;

create or replace function public.receivable_set_paid(p_owner uuid,p_id uuid,p_paid boolean) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare c public.receivables;
begin
 select * into c from public.receivables where id=p_id and owner_id=p_owner for update;
 if not found then raise exception 'RECEIPT_NOT_FOUND';end if;
 if (c.status='PAID')<>p_paid then
  update public.receivables set status=case when p_paid then 'PAID' else 'OWED' end,paid_at=case when p_paid then now() else null end,updated_at=now()
  where id=p_id returning * into c;
  insert into public.receipt_events(receipt_id,event_type,actor_user_id,old_value,new_value)
  values(c.receipt_id,case when p_paid then 'RECEIVABLE_PAID' else 'RECEIVABLE_UNPAID' end,p_owner,null,jsonb_build_object('receivable_id',c.id,'amount_owed_minor',c.amount_owed_minor));
 end if;
 return jsonb_build_object('id',c.id,'status',c.status,'paid_at',c.paid_at);
end $$;
