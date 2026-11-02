-- VP-027: the scan limits are what cap model cost, so each number has a test that fails if it changes.
begin;
insert into auth.users(id,email,raw_user_meta_data) values
 ('00000000-0000-4000-8000-0000000000f1','limits-day@example.invalid','{}'),
 ('00000000-0000-4000-8000-0000000000f2','limits-month@example.invalid','{}'),
 ('00000000-0000-4000-8000-0000000000f3','limits-retry@example.invalid','{}');
DO $$ declare d uuid:='00000000-0000-4000-8000-0000000000f1';m uuid:='00000000-0000-4000-8000-0000000000f2';t uuid:='00000000-0000-4000-8000-0000000000f3';
 r jsonb;n integer;v bigint;
begin
 -- 5 scans per day (finished receipts, so the in-progress limit is not what stops the sixth)
 for n in 1..5 loop
  r:=public.receipt_create(d,lpad(n::text,64,'a'),'image/png');
  update public.receipts set status='NEEDS_REVIEW' where id=(r->>'id')::uuid;
 end loop;
 begin perform public.receipt_create(d,repeat('b',64),'image/png');raise exception 'sixth scan in a day allowed';exception when raise_exception then if sqlerrm<>'RECEIPT_QUOTA' then raise;end if;end;
 -- 20 scans per 30 days (19 older ones, then the 20th succeeds and the 21st is refused although today is nearly empty)
 insert into public.receipts(id,owner_id,image_path,image_hash,image_mime,status,created_at)
  select gen_random_uuid(),m,'f2/'||i,lpad(i::text,64,'c'),'image/png','NEEDS_REVIEW',now()-interval '10 days' from generate_series(1,19) i;
 r:=public.receipt_create(m,repeat('d',64),'image/png');
 update public.receipts set status='NEEDS_REVIEW' where id=(r->>'id')::uuid;
 begin perform public.receipt_create(m,repeat('e',64),'image/png');raise exception 'twenty-first scan in 30 days allowed';exception when raise_exception then if sqlerrm<>'RECEIPT_QUOTA' then raise;end if;end;
 -- 1 retry per receipt
 r:=public.receipt_create(t,repeat('1',64),'image/png');
 update public.receipts set status='PROCESSING_FAILED',error_code='VISION_UNAVAILABLE' where id=(r->>'id')::uuid;
 select version into v from public.receipts where id=(r->>'id')::uuid;
 perform public.receipt_retry(t,(r->>'id')::uuid,v);
 update public.receipts set status='PROCESSING_FAILED',error_code='VISION_UNAVAILABLE' where id=(r->>'id')::uuid;
 select version into v from public.receipts where id=(r->>'id')::uuid;
 begin perform public.receipt_retry(t,(r->>'id')::uuid,v);raise exception 'second retry allowed';exception when raise_exception then if sqlerrm<>'RECEIPT_RETRY_LIMIT' then raise;end if;end;
end $$;
rollback;
