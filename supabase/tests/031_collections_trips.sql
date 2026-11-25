-- VP-029: collections/trips membership, invites and signed-in member claims.
begin;
insert into auth.users(id,email,raw_user_meta_data) values
 ('00000000-0000-4000-8000-0000000000b1','owner@example.invalid','{}'),
 ('00000000-0000-4000-8000-0000000000b2','member@example.invalid','{}'),
 ('00000000-0000-4000-8000-0000000000b3','stranger@example.invalid','{}');
select set_config('test.r',(public.receipt_create('00000000-0000-4000-8000-0000000000b1',repeat('e',64),'image/png')->>'id'),true);
DO $$ declare o uuid:='00000000-0000-4000-8000-0000000000b1';m uuid:='00000000-0000-4000-8000-0000000000b2';s uuid:='00000000-0000-4000-8000-0000000000b3';
 r uuid:=current_setting('test.r')::uuid;i uuid:=gen_random_uuid();t uuid;t2 uuid;h text:=repeat('a',64);now_ timestamptz:='2026-11-22 12:00:00+00';pid uuid;
begin
 t:=(public.trip_create(o,'  Trip A ')->>'id')::uuid;
 if (select name from public.collections where id=t)<>'Trip A' then raise exception 'name not trimmed';end if;
 begin perform public.trip_create(o,'   ');raise exception 'blank name';exception when raise_exception then if sqlerrm<>'TRIP_INVALID_NAME' then raise;end if;end;
 -- only members see a trip
 begin perform public.trip_detail(s,t);raise exception 'stranger saw trip';exception when raise_exception then if sqlerrm<>'TRIP_NOT_FOUND' then raise;end if;end;
 if jsonb_array_length(public.trip_list(s))<>0 then raise exception 'stranger list';end if;
 -- invite: valid once, expires, revoked by a newer link, stranger cannot create one
 begin perform public.trip_invite(m,t,h,now_+interval '1 day');raise exception 'non-owner invite';exception when raise_exception then if sqlerrm<>'TRIP_NOT_FOUND' then raise;end if;end;
 perform public.trip_invite(o,t,h,now_+interval '1 day');
 begin perform public.trip_join(m,h,now_+interval '1 day');raise exception 'joined at expiry';exception when raise_exception then if sqlerrm<>'TRIP_INVITE_INVALID' then raise;end if;end;
 perform public.trip_invite(o,t,repeat('b',64),now_+interval '1 day');
 begin perform public.trip_join(m,h,now_);raise exception 'revoked invite worked';exception when raise_exception then if sqlerrm<>'TRIP_INVITE_INVALID' then raise;end if;end;
 perform public.trip_join(m,repeat('b',64),now_);
 perform public.trip_join(m,repeat('b',64),now_); -- idempotent
 if (select count(*) from public.collection_members where collection_id=t)<>2 then raise exception 'member count';end if;
 -- filing: only the receipt owner, only into a trip they belong to
 begin perform public.receipt_set_collection(s,r,t);raise exception 'stranger filed';exception when raise_exception then if sqlerrm<>'TRIP_NOT_FOUND' then raise;end if;end;
 t2:=(public.trip_create(m,'Other')->>'id')::uuid;
 begin perform public.receipt_set_collection(o,r,t2);raise exception 'filed into foreign trip';exception when raise_exception then if sqlerrm<>'TRIP_NOT_FOUND' then raise;end if;end;
 begin perform public.receipt_set_collection(m,r,t);raise exception 'non-owner filed receipt';exception when raise_exception then if sqlerrm<>'RECEIPT_NOT_FOUND' then raise;end if;end;
 perform public.receipt_set_collection(o,r,t);
 -- drafts are not visible to the trip; confirmed receipts are
 if jsonb_array_length(public.trip_detail(m,t)->'receipts')<>0 then raise exception 'draft leaked';end if;
 begin perform public.trip_claim(m,t,r,i,false);raise exception 'claim before ready';exception when raise_exception then if sqlerrm not in('SPLIT_NOT_READY') then raise;end if;end;
 update public.receipts set status='NEEDS_REVIEW',has_owner_draft=true where id=r;
 insert into public.receipt_items(id,receipt_id,description,quantity,total_price_minor,position) values(i,r,'Soup',1,500,0);
 if jsonb_array_length(public.trip_detail(m,t)->'receipts')<>1 then raise exception 'receipt not visible to member';end if;
 -- member claims as themselves; stranger cannot
 begin perform public.trip_claim(s,t,r,i,false);raise exception 'stranger claimed';exception when raise_exception then if sqlerrm<>'TRIP_NOT_FOUND' then raise;end if;end;
 pid:=(public.trip_claim(m,t,r,i,false)->>'participant_id')::uuid;
 perform public.trip_claim(m,t,r,i,false); -- idempotent
 if (select count(*) from public.receipt_participants where receipt_id=r and user_id=m)<>1 then raise exception 'duplicate participant';end if;
 if (select name from public.receipt_participants where id=pid)<>'member' then raise exception 'participant name from profile';end if;
 if (select amount_minor from public.allocations where receipt_item_id=i and participant_id=pid)<>500 then raise exception 'not allocated';end if;
 if (public.trip_receipt_access(m,t,r)->>'participant_id')::uuid<>pid then raise exception 'access view';end if;
 perform public.trip_claim(m,t,r,i,true);
 if exists(select 1 from public.allocations where receipt_item_id=i) then raise exception 'release failed';end if;
 -- finalized receipts cannot be claimed
 update public.receipts set status='FINALIZED' where id=r;
 begin perform public.trip_claim(m,t,r,i,false);raise exception 'claimed finalized';exception when raise_exception then if sqlerrm<>'SPLIT_NOT_READY' then raise;end if;end;
 -- leaving: members leave themselves, owner cannot be removed, others cannot remove
 begin perform public.trip_remove_member(m,t,o);raise exception 'removed owner';exception when raise_exception then if sqlerrm<>'TRIP_FORBIDDEN' then raise;end if;end;
 perform public.trip_remove_member(m,t,m);
 begin perform public.trip_detail(m,t);raise exception 'left member still sees trip';exception when raise_exception then if sqlerrm<>'TRIP_NOT_FOUND' then raise;end if;end;
 -- deleting a trip un-files receipts instead of deleting them
 perform public.trip_delete(o,t);
 if (select collection_id from public.receipts where id=r) is not null then raise exception 'receipt kept stale trip';end if;
 begin perform public.trip_delete(o,t);raise exception 'double delete';exception when raise_exception then if sqlerrm<>'TRIP_NOT_FOUND' then raise;end if;end;
end $$;
rollback;
