begin;
insert into auth.users(id,email,raw_user_meta_data) values
 ('00000000-0000-4000-8000-0000000000b1','del-one@example.invalid','{}'),
 ('00000000-0000-4000-8000-0000000000b2','del-two@example.invalid','{}');
select set_config('test.r',(public.receipt_create('00000000-0000-4000-8000-0000000000b1',repeat('e',64),'image/png')->>'id'),true);
DO $$ declare o uuid:='00000000-0000-4000-8000-0000000000b1';x uuid:='00000000-0000-4000-8000-0000000000b2';r uuid:=current_setting('test.r')::uuid;d jsonb;
begin
 perform public.split_session_open(o,r,repeat('9',64),now(),300);
 begin perform public.receipt_delete(x,r);raise exception 'cross-owner delete';exception when raise_exception then if sqlerrm<>'RECEIPT_NOT_FOUND' then raise;end if;end;
 update public.receipts set status='PROCESSING' where id=r;
 begin perform public.receipt_delete(o,r);raise exception 'deleted while processing';exception when raise_exception then if sqlerrm<>'RECEIPT_NOT_REVIEWABLE' then raise;end if;end;
 update public.receipts set status='NEEDS_REVIEW' where id=r;
 d:=public.receipt_delete(o,r);
 if d->>'image_path' is null then raise exception 'no image path';end if;
 if exists(select 1 from public.receipts where id=r) or exists(select 1 from public.split_sessions where receipt_id=r) then raise exception 'rows remain';end if;
 if (select count(*) from public.receipt_deletions where owner_id=o)<>1 then raise exception 'deletion not counted';end if;
end $$;
rollback;
