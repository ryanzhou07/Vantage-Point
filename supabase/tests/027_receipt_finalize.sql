begin;
insert into auth.users(id,email,raw_user_meta_data) values
 ('00000000-0000-4000-8000-0000000000d1','fin-one@example.invalid','{}'),
 ('00000000-0000-4000-8000-0000000000d2','fin-two@example.invalid','{}');
select set_config('test.r',(public.receipt_create('00000000-0000-4000-8000-0000000000d1',repeat('a',64),'image/png')->>'id'),true);
DO $$ declare o uuid:='00000000-0000-4000-8000-0000000000d1';x uuid:='00000000-0000-4000-8000-0000000000d2';r uuid:=current_setting('test.r')::uuid;
 d jsonb;pv jsonb;doc jsonb;v bigint;c jsonb;
begin
 update public.receipts set status='NEEDS_REVIEW' where id=r;
 select version into v from public.receipts where id=r;
 d:=jsonb_build_object('merchant','Cafe','date','2026-10-29','currency','USD','subtotal_minor',800,'tax_minor',0,'tip_minor',0,'discount_minor',0,'total_minor',800,'tax_method','equal','tip_method','equal',
  'items',jsonb_build_array(jsonb_build_object('id','i1','name','Soup','quantity',1,'amount_minor',500,'participant_id','bob'),jsonb_build_object('id','i2','name','Tea','quantity',1,'amount_minor',300,'participant_id','me')),
  'participants',jsonb_build_array(jsonb_build_object('id','me','name','Ada','included',true,'is_owner',true),jsonb_build_object('id','bob','name','Bob','included',true)));
 pv:=jsonb_build_object('shares',jsonb_build_array(jsonb_build_object('participant_id','me','items_minor',300,'tax_minor',0,'tip_minor',0,'total_minor',300),jsonb_build_object('participant_id','bob','items_minor',500,'tax_minor',0,'tip_minor',0,'total_minor',500)));
 doc:=public.receipt_save(o,r,v,d,null);v:=(doc->>'version')::bigint;
 begin perform public.receipt_finalize(o,r,v);raise exception 'finalized without preview';exception when raise_exception then if sqlerrm<>'RECEIPT_INVALID_REVIEW' then raise;end if;end;
 doc:=public.receipt_save(o,r,v,d,pv);v:=(doc->>'version')::bigint;
 begin perform public.receipt_finalize(x,r,v);raise exception 'cross-owner finalize';exception when raise_exception then if sqlerrm<>'RECEIPT_NOT_FOUND' then raise;end if;end;
 begin perform public.receipt_finalize(o,r,v-1);raise exception 'stale version';exception when raise_exception then if sqlerrm<>'RECEIPT_CONFLICT' then raise;end if;end;
 doc:=public.receipt_finalize(o,r,v);
 if doc->>'status'<>'FINALIZED' then raise exception 'not finalized';end if;
 if (select count(*) from public.receivables where receipt_id=r)<>1 or (select amount_owed_minor from public.receivables where receipt_id=r)<>500 then raise exception 'receivable wrong';end if;
 begin perform public.receipt_finalize(o,r,v+1);raise exception 'double finalize';exception when raise_exception then if sqlerrm<>'RECEIPT_NOT_REVIEWABLE' then raise;end if;end;
 begin perform public.receipt_save(o,r,v+1,d,null);raise exception 'edited finalized';exception when raise_exception then if sqlerrm<>'RECEIPT_NOT_REVIEWABLE' then raise;end if;end;
 begin perform public.receipt_delete(o,r);raise exception 'deleted finalized';exception when raise_exception then if sqlerrm<>'RECEIPT_NOT_REVIEWABLE' then raise;end if;end;
 c:=(public.receivables_search(o,null,null,null,null,25,null)->'receivables'->0);
 if c->>'participant_name'<>'Bob' or c->>'status'<>'OWED' then raise exception 'list wrong';end if;
 if c->>'merchant'<>'Cafe' or c->>'payment_method' is not null then raise exception 'snapshot wrong';end if;
 if jsonb_array_length(public.receivables_search(x,null,null,null,null,25,null)->'receivables')<>0 then raise exception 'receivables leak';end if;
 begin perform public.receivable_set_paid(x,(c->>'id')::uuid,true);raise exception 'cross-owner paid';exception when raise_exception then if sqlerrm<>'RECEIPT_NOT_FOUND' then raise;end if;end;
 perform public.receivable_set_paid(o,(c->>'id')::uuid,true);
 if (select status from public.receivables where id=(c->>'id')::uuid)<>'PAID' or (select paid_at from public.receivables where id=(c->>'id')::uuid) is null then raise exception 'not paid';end if;
 perform public.receivable_set_paid(o,(c->>'id')::uuid,true);
 if (select count(*) from public.receipt_events where receipt_id=r and event_type='RECEIVABLE_PAID')<>1 then raise exception 'paid not idempotent';end if;
 perform public.receivable_set_paid(o,(c->>'id')::uuid,false);
 if (select status from public.receivables where id=(c->>'id')::uuid)<>'OWED' then raise exception 'not unpaid';end if;
end $$;
rollback;
