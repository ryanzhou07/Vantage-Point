insert into auth.users(id,email,raw_user_meta_data) values('00000000-0000-4000-8000-000000000088','normalized-noop@example.invalid','{}');
insert into public.receipts(id,owner_id,image_path,image_hash,image_mime,status,version)
values('00000000-0000-4000-8000-000000000089','00000000-0000-4000-8000-000000000088','normalized/image.png',repeat('a',64),'image/png','NEEDS_REVIEW',5);
select public.receipt_save('00000000-0000-4000-8000-000000000088','00000000-0000-4000-8000-000000000089',5,
'{"merchant":"Already normalized","date":"2026-10-14","currency":"USD","items":[{"id":"one","name":"Preserved typed item","quantity":1,"amount_minor":100,"participant_id":""}],"subtotal_minor":100,"tax_minor":0,"tip_minor":0,"discount_minor":0,"total_minor":100,"participants":[],"tax_method":"","tip_method":""}',null);
create table public.receipt_noop_fingerprint as select md5(jsonb_build_object(
 'receipts',(select jsonb_agg(to_jsonb(r) order by id) from public.receipts r),
 'items',(select jsonb_agg(to_jsonb(i) order by id) from public.receipt_items i),
 'events',(select jsonb_agg(to_jsonb(e) order by id) from public.receipt_events e))::text) as value;
