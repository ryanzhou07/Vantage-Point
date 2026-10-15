begin;
insert into auth.users(id,email,raw_user_meta_data) values
 ('00000000-0000-4000-8000-000000000081','receipt-one@example.invalid','{}'),
 ('00000000-0000-4000-8000-000000000082','receipt-two@example.invalid','{}');
select set_config('test.receipt_one',(public.receipt_create('00000000-0000-4000-8000-000000000081',repeat('a',64),'image/png')->>'id'),true);
select set_config('test.receipt_two',(public.receipt_create('00000000-0000-4000-8000-000000000082',repeat('a',64),'image/png')->>'id'),true);
insert into storage.objects(bucket_id,name) select 'receipt-images',image_path from public.receipts where id in (current_setting('test.receipt_one')::uuid,current_setting('test.receipt_two')::uuid);
DO $$ begin
 if public.receipt_create('00000000-0000-4000-8000-000000000081',repeat('a',64),'image/png')->>'id'<>current_setting('test.receipt_one') then raise exception 'duplicate expensive work';end if;
 if public.receipt_enqueue('00000000-0000-4000-8000-000000000081',current_setting('test.receipt_one')::uuid)->>'status'<>'PROCESSING' then raise exception 'enqueue failed';end if;
 perform public.receipt_enqueue('00000000-0000-4000-8000-000000000081',current_setting('test.receipt_one')::uuid);
 if (select count(*) from public.receipt_vision_jobs where receipt_id=current_setting('test.receipt_one')::uuid)<>1 then raise exception 'duplicate jobs';end if;
 begin perform public.receipt_enqueue('00000000-0000-4000-8000-000000000082',current_setting('test.receipt_one')::uuid);raise exception 'cross-owner write';exception when raise_exception then if sqlerrm<>'RECEIPT_NOT_FOUND' then raise;end if;end;
end $$;
set local role authenticated;
select set_config('request.jwt.claim.sub','00000000-0000-4000-8000-000000000081',true);
DO $$ begin
 if (select count(*) from public.receipts where id in (current_setting('test.receipt_one')::uuid,current_setting('test.receipt_two')::uuid))<>1 then raise exception 'receipt RLS leak';end if;
 if (select count(*) from storage.objects where bucket_id='receipt-images')<>1 then raise exception 'image RLS leak';end if;
 begin update public.receipts set status='NEEDS_REVIEW';raise exception 'browser mutation';exception when insufficient_privilege then null;end;
 begin perform public.receipt_create('00000000-0000-4000-8000-000000000082',repeat('b',64),'image/png');raise exception 'browser privileged function';exception when insufficient_privilege then null;end;
 begin perform * from public.receipt_vision_jobs;raise exception 'browser jobs leak';exception when insufficient_privilege then null;end;
end $$;
reset role;
DO $$ declare j jsonb;accepted boolean;begin
 j:=public.receipt_claim();
 if j is null then raise exception 'durable job not claimable';end if;
 accepted:=public.receipt_finish((j->'receipt'->>'id')::uuid,gen_random_uuid(),'{}',null,false);
 if accepted then raise exception 'wrong lease accepted';end if;
 accepted:=public.receipt_finish((j->'receipt'->>'id')::uuid,(j->>'lease_token')::uuid,'{}',null,false);
 if not accepted then raise exception 'valid lease rejected';end if;
 if public.receipt_finish((j->'receipt'->>'id')::uuid,(j->>'lease_token')::uuid,'{}',null,false) then raise exception 'duplicate finish accepted';end if;
 perform public.receipt_save('00000000-0000-4000-8000-000000000081',current_setting('test.receipt_one')::uuid,3,'{}',null);
 begin perform public.receipt_save('00000000-0000-4000-8000-000000000081',current_setting('test.receipt_one')::uuid,3,'{}',null);raise exception 'stale overwrite';exception when raise_exception then if sqlerrm<>'RECEIPT_CONFLICT' then raise;end if;end;
end $$;
-- Lease expiry reclaims work with a fresh fencing token; retries are capped.
DO $$ declare j jsonb;k jsonb;rid uuid;begin
 rid:=current_setting('test.receipt_two')::uuid;
 perform public.receipt_enqueue('00000000-0000-4000-8000-000000000082',rid);
 j:=public.receipt_claim();
 update public.receipt_vision_jobs set lease_until=now()-interval '1 second' where receipt_id=rid;
 k:=public.receipt_claim();
 if j->>'lease_token'=k->>'lease_token' or (k->>'attempts')::int<>2 then raise exception 'lease recovery/fencing failed';end if;
 if public.receipt_finish(rid,(j->>'lease_token')::uuid,'{}',null,false) then raise exception 'stale worker overwrote job';end if;
 perform public.receipt_finish(rid,(k->>'lease_token')::uuid,null,'VISION_UNAVAILABLE',true);
 if (select state from public.receipt_vision_jobs where receipt_id=rid)<>'QUEUED' then raise exception 'transient retry not queued';end if;
 update public.receipt_vision_jobs set available_at=now()-interval '1 second' where receipt_id=rid;
 k:=public.receipt_claim();
 perform public.receipt_finish(rid,(k->>'lease_token')::uuid,null,'VISION_UNAVAILABLE',true);
 if (select status from public.receipts where id=rid)<>'PROCESSING_FAILED' then raise exception 'automatic retries unbounded';end if;
end $$;
-- Database-wide paid-work backpressure and owner upload quota.
DO $$ declare j jsonb;k jsonb;r jsonb;n integer;begin
 for n in 1..3 loop
  r:=public.receipt_create('00000000-0000-4000-8000-000000000081',repeat((n+2)::text,64),'image/png');
  perform public.receipt_enqueue('00000000-0000-4000-8000-000000000081',(r->>'id')::uuid);
 end loop;
 j:=public.receipt_claim();k:=public.receipt_claim();
 if j is null or k is null or public.receipt_claim() is not null then raise exception 'global worker concurrency bound';end if;
 perform public.receipt_create('00000000-0000-4000-8000-000000000081',repeat('8',64),'image/png');
 perform public.receipt_create('00000000-0000-4000-8000-000000000081',repeat('9',64),'image/png');
 begin perform public.receipt_create('00000000-0000-4000-8000-000000000081',repeat('f',64),'image/png');raise exception 'active quota bypass';exception when raise_exception then if sqlerrm<>'RECEIPT_QUOTA' then raise;end if;end;
end $$;
set local role anon;
DO $$ begin
 begin perform * from public.receipts;raise exception 'anonymous receipt read';exception when insufficient_privilege then null;end;
 if exists(select from storage.objects where bucket_id='receipt-images') then raise exception 'anonymous image read';end if;
end $$;
reset role;
DO $$ begin if (select public from storage.buckets where id='receipt-images') then raise exception 'public images';end if;end $$;
rollback;
