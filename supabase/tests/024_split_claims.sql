begin;
insert into auth.users(id,email,raw_user_meta_data) values('00000000-0000-4000-8000-0000000000a1','claims@example.invalid','{}');
select set_config('test.r',(public.receipt_create('00000000-0000-4000-8000-0000000000a1',repeat('d',64),'image/png')->>'id'),true);
DO $$ declare t timestamptz:='2026-10-29 12:00:00+00';o uuid:='00000000-0000-4000-8000-0000000000a1';r uuid:=current_setting('test.r')::uuid;
 i uuid:=gen_random_uuid();a uuid;b uuid;h text:=repeat('c',64);v bigint;
begin
 perform public.split_session_open(o,r,h,t,300);
 begin perform public.split_join_add(h,t,'Early');raise exception 'joined before owner draft';exception when raise_exception then if sqlerrm<>'SPLIT_NOT_READY' then raise;end if;end;
 update public.receipts set status='NEEDS_REVIEW',has_owner_draft=true where id=r;
 insert into public.receipt_items(id,receipt_id,description,quantity,total_price_minor,position) values(i,r,'Soup',1,500,0);
 a:=(public.split_join_add(h,t,'  Ann ')->>'id')::uuid;
 b:=(public.split_join_add(h,t,'Bob')->>'id')::uuid;
 if (select name from public.receipt_participants where id=a)<>'Ann' then raise exception 'name not trimmed';end if;
 begin perform public.split_join_add(h,t,'   ');raise exception 'blank name';exception when raise_exception then if sqlerrm<>'SPLIT_INVALID_NAME' then raise;end if;end;
 select version into v from public.receipts where id=r;
 perform public.split_claim(h,t,a,i,false);
 perform public.split_claim(h,t,a,i,false); -- idempotent for the holder
 begin perform public.split_claim(h,t,b,i,false);raise exception 'stole item';exception when raise_exception then if sqlerrm<>'SPLIT_CLAIM_CONFLICT' then raise;end if;end;
 perform public.split_claim(h,t,b,i,true); -- non-holder release is a no-op
 if (select participant_id from public.allocations where receipt_item_id=i)<>a then raise exception 'claim lost';end if;
 if (select version from public.receipts where id=r)<=v then raise exception 'version not bumped';end if;
 begin perform public.split_claim(h,t,gen_random_uuid(),i,false);raise exception 'foreign participant';exception when raise_exception then if sqlerrm<>'SPLIT_PARTICIPANT_NOT_FOUND' then raise;end if;end;
 begin perform public.split_claim(h,t,a,gen_random_uuid(),false);raise exception 'foreign item';exception when raise_exception then if sqlerrm<>'SPLIT_ITEM_NOT_FOUND' then raise;end if;end;
 begin perform public.split_claim(h,t+interval '300 seconds',a,i,false);raise exception 'claim at expiry';exception when raise_exception then if sqlerrm<>'SESSION_INVALID' then raise;end if;end;
 perform public.split_claim(h,t,a,i,true);
 if exists(select 1 from public.allocations where receipt_item_id=i) then raise exception 'release failed';end if;
end $$;
rollback;
