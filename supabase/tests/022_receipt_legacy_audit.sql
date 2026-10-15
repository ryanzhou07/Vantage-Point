begin;
insert into auth.users(id,email,raw_user_meta_data) values('00000000-0000-4000-8000-000000000095','audit-one@example.invalid','{}'),('00000000-0000-4000-8000-000000000096','audit-two@example.invalid','{}');
select set_config('test.legacy_audit_id',(public.receipt_create('00000000-0000-4000-8000-000000000095',repeat('e',64),'image/png')->>'id'),true);
insert into public.receipt_legacy_review_audit(receipt_id,owner_id,old_value,new_value,review_reasons) values(current_setting('test.legacy_audit_id')::uuid,'00000000-0000-4000-8000-000000000095','{"draft":{"date":"Oct 14 (estimated)"},"preview":{"total_minor":280}}','{"draft":{"date":""},"preview":null}',array['DATE_REVIEW']);
set local role authenticated;
select set_config('request.jwt.claim.sub','00000000-0000-4000-8000-000000000095',true);
DO $$ begin
 if (select count(*) from public.receipt_legacy_review_audit)<>1 then raise exception 'Own migration audit missing';end if;
 begin update public.receipt_legacy_review_audit set old_value='{}';raise exception 'Browser rewrote audit';exception when insufficient_privilege then null;end;
end $$;
select set_config('request.jwt.claim.sub','00000000-0000-4000-8000-000000000096',true);
DO $$ begin if exists(select from public.receipt_legacy_review_audit) then raise exception 'Other owner raw date/financial history leaked';end if;end $$;
reset role;
set local role service_role;
DO $$ begin
 begin update public.receipt_legacy_review_audit set old_value='{}';raise exception 'Service rewrote append-only migration audit';exception when insufficient_privilege then null;end;
 begin delete from public.receipt_legacy_review_audit;raise exception 'Service deleted migration audit';exception when insufficient_privilege then null;end;
end $$;
reset role;
set local role anon;
DO $$ begin begin perform * from public.receipt_legacy_review_audit;raise exception 'Anonymous raw date/financial history leaked';exception when insufficient_privilege then null;end;end $$;
reset role;
rollback;
