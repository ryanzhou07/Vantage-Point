begin;
insert into auth.users(id,email,raw_user_meta_data) values('00000000-0000-4000-8000-000000000091','normalized-one@example.invalid','{}'),('00000000-0000-4000-8000-000000000092','normalized-two@example.invalid','{}');
select set_config('test.normalized_one',(public.receipt_create('00000000-0000-4000-8000-000000000091',repeat('1',64),'image/png')->>'id'),true);
select set_config('test.normalized_two',(public.receipt_create('00000000-0000-4000-8000-000000000092',repeat('1',64),'image/png')->>'id'),true);
update public.receipts set status='NEEDS_REVIEW' where id in(current_setting('test.normalized_one')::uuid,current_setting('test.normalized_two')::uuid);
select set_config('test.normalized_draft','{"merchant":"Reviewed cafe","date":"2026-10-15","currency":"USD","items":[{"id":"one","name":"Meal","quantity":1,"amount_minor":100,"participant_id":"a"},{"id":"two","name":"Drink","quantity":2,"amount_minor":200,"participant_id":"b"}],"subtotal_minor":300,"tax_minor":0,"tip_minor":0,"discount_minor":0,"total_minor":300,"participants":[{"id":"a","name":"Student A","included":true},{"id":"b","name":"Student B","included":true}],"tax_method":"equal","tip_method":"proportional"}',true);
select set_config('test.normalized_preview','{"draft":true,"total_minor":300,"shares":[{"participant_id":"a","items_minor":100,"tax_minor":0,"tip_minor":0,"total_minor":100},{"participant_id":"b","items_minor":200,"tax_minor":0,"tip_minor":0,"total_minor":200}]}',true);
DO $$ declare r uuid;d jsonb;begin
 r:=current_setting('test.normalized_one')::uuid;
 d:=public.receipt_save('00000000-0000-4000-8000-000000000091',r,1,current_setting('test.normalized_draft')::jsonb,current_setting('test.normalized_preview')::jsonb);
 if (select subtotal_minor from public.receipts where id=r)<>300 or (select count(*) from public.receipt_items where receipt_id=r)<>2 or(select count(*) from public.allocations where receipt_id=r)<>2 then raise exception 'Normalized authority absent';end if;
 if d->'preview'->>'total_minor'<>'300' or d->'draft'->'items'->0->>'amount_minor'<>'100' then raise exception 'Projection not derived correctly';end if;
 begin perform public.receipt_save('00000000-0000-4000-8000-000000000091',r,1,current_setting('test.normalized_draft')::jsonb,null);raise exception 'stale normalized overwrite';exception when raise_exception then if sqlerrm<>'RECEIPT_CONFLICT' then raise;end if;end;
 begin perform public.receipt_save('00000000-0000-4000-8000-000000000091',r,2,jsonb_set(current_setting('test.normalized_draft')::jsonb,'{total_minor}','1'),current_setting('test.normalized_preview')::jsonb);raise exception 'invalid preview accepted';exception when raise_exception then if sqlerrm<>'RECEIPT_INVALID_REVIEW' then raise;end if;end;
 if (select version from public.receipts where id=r)<>2 or (select total_minor from public.receipts where id=r)<>300 then raise exception 'Failed save partially committed';end if;
 begin update public.receipt_items set total_price_minor=total_price_minor-1 where receipt_id=r;raise exception 'item amount broke accepted assignment';exception when check_violation then null;end;
 begin update public.allocations set amount_minor=amount_minor+1 where receipt_id=r;raise exception 'overallocation allowed';exception when check_violation then null;end;
 begin insert into public.allocations(receipt_id,receipt_item_id,participant_id,amount_minor) select receipt_id,receipt_item_id,participant_id,amount_minor from public.allocations where receipt_id=r limit 1;raise exception 'duplicate whole-line allocation';exception when unique_violation then null;end;
 insert into public.receipt_participants(id,receipt_id,name,included,position) values('00000000-0000-4000-8000-000000000093',current_setting('test.normalized_two')::uuid,'Other receipt person',true,0);
 begin update public.allocations set participant_id='00000000-0000-4000-8000-000000000093' where receipt_id=r;raise exception 'cross-receipt FK bypass';exception when foreign_key_violation then null;end;
 begin update public.receipts set tax_minor=-1 where id=r;raise exception 'negative money';exception when check_violation then null;end;
 if exists(select from information_schema.columns where table_schema='public' and table_name='receipts' and column_name in('draft','preview')) then raise exception 'JSON authority retained';end if;
 if (select data_type from information_schema.columns where table_schema='public' and table_name='allocations' and column_name='amount_minor')<>'bigint' then raise exception 'Allocation money not BIGINT';end if;
end $$;
set local role authenticated;
select set_config('request.jwt.claim.sub','00000000-0000-4000-8000-000000000091',true);
DO $$ begin
 if (select count(*) from public.receipt_items)<>2 or(select count(*) from public.receipt_participants)<>2 or(select count(*) from public.allocations)<>2 or(select count(*) from public.receipt_draft_shares)<>2 then raise exception 'Normalized owner RLS leak/missing';end if;
 begin update public.receipt_items set total_price_minor=0;raise exception 'browser typed write';exception when insufficient_privilege then null;end;
 begin perform public.receipt_document('00000000-0000-4000-8000-000000000092',current_setting('test.normalized_two')::uuid);raise exception 'browser privileged projection';exception when insufficient_privilege then null;end;
end $$;
select set_config('request.jwt.claim.sub','00000000-0000-4000-8000-000000000092',true);
DO $$ begin
 if exists(select from public.receipt_items) or exists(select from public.allocations) or exists(select from public.receipt_draft_shares) or exists(select from public.receipt_events) or(select count(*) from public.receipt_participants)<>1 then raise exception 'Other owner normalized data leaked';end if;
end $$;
reset role;
DO $$ declare d jsonb;begin
 d:=public.receipt_save('00000000-0000-4000-8000-000000000091',current_setting('test.normalized_one')::uuid,2,current_setting('test.normalized_draft')::jsonb,null);
 if d->'preview'<>'null'::jsonb or exists(select from public.receipt_draft_shares where receipt_id=current_setting('test.normalized_one')::uuid) then raise exception 'Old preview survived edit';end if;
end $$;
rollback;
