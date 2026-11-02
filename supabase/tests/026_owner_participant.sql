begin;
insert into auth.users(id,email,raw_user_meta_data) values('00000000-0000-4000-8000-0000000000c1','owner-p@example.invalid','{}');
select set_config('test.r',(public.receipt_create('00000000-0000-4000-8000-0000000000c1',repeat('f',64),'image/png')->>'id'),true);
DO $$ declare o uuid:='00000000-0000-4000-8000-0000000000c1';r uuid:=current_setting('test.r')::uuid;d jsonb;doc jsonb;v bigint;
begin
 update public.receipts set status='NEEDS_REVIEW' where id=r;
 select version into v from public.receipts where id=r;
 d:=jsonb_build_object('merchant','Cafe','date','2026-10-29','currency','USD','subtotal_minor',500,'tax_minor',0,'tip_minor',0,'discount_minor',0,'total_minor',500,'tax_method','equal','tip_method','equal',
  'items',jsonb_build_array(jsonb_build_object('id','i1','name','Soup','quantity',1,'amount_minor',500,'participant_id','')),
  'participants',jsonb_build_array(jsonb_build_object('id','me','name','Ada','included',true,'is_owner',true)));
 doc:=public.receipt_save(o,r,v,d,null);
 if (doc->'draft'->'participants'->0->>'is_owner')::boolean is not true then raise exception 'owner flag not projected';end if;
 v:=(doc->>'version')::bigint;
 begin perform public.receipt_save(o,r,v,jsonb_set(d,'{participants}','[]'::jsonb),null);raise exception 'owner removed';exception when raise_exception then if sqlerrm<>'RECEIPT_INVALID_REVIEW' then raise;end if;end;
 begin perform public.receipt_save(o,r,v,jsonb_set(d,'{participants,0,name}','"Eve"'),null);raise exception 'owner renamed';exception when raise_exception then if sqlerrm<>'RECEIPT_INVALID_REVIEW' then raise;end if;end;
 begin perform public.receipt_save(o,r,v,jsonb_set(d,'{participants}',d->'participants'||jsonb_build_array(jsonb_build_object('id','b','name','Bob','included',true,'is_owner',true))),null);raise exception 'two owners';exception when unique_violation then null;end;
 perform public.receipt_save(o,r,v,d,null);
end $$;
rollback;
