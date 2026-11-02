begin;
insert into auth.users(id,email,raw_user_meta_data) values
 ('00000000-0000-4000-8000-0000000000e1','search-one@example.invalid','{}'),
 ('00000000-0000-4000-8000-0000000000e2','search-two@example.invalid','{}');
-- 800 finalized-style receipts x 25 participants = 20000 receivables for owner one, 3 for owner two.
insert into public.receipts(id,owner_id,image_path,image_hash,image_mime,status,merchant_name,receipt_date)
 select ('00000000-0000-4000-9000-'||lpad(n::text,12,'0'))::uuid,'00000000-0000-4000-8000-0000000000e1','e1/'||n,lpad(n::text,64,'0'),'image/png','FINALIZED','Cafe '||n,date '2026-01-01'+n from generate_series(1,800) n;
insert into public.receipts(id,owner_id,image_path,image_hash,image_mime,status,merchant_name,receipt_date)
 values('00000000-0000-4000-9000-00000000ffff','00000000-0000-4000-8000-0000000000e2','e2/x',repeat('e',64),'image/png','FINALIZED','Other cafe',date '2026-02-01');
insert into public.receipt_participants(id,receipt_id,name,included,position)
 select ('00000000-0000-4000-a000-'||lpad((n*100+m)::text,12,'0'))::uuid,('00000000-0000-4000-9000-'||lpad(n::text,12,'0'))::uuid,'Person '||m,true,m from generate_series(1,800) n,generate_series(1,25) m;
insert into public.receipt_participants(id,receipt_id,name,included,position) values('00000000-0000-4000-a000-00000000ffff','00000000-0000-4000-9000-00000000ffff','Stranger',true,0);
insert into public.receivables(receipt_id,owner_id,participant_id,items_minor,tax_minor,tip_minor,amount_owed_minor,participant_name,merchant_name,receipt_date,status,paid_at,created_at,payment_method)
 select p.receipt_id,'00000000-0000-4000-8000-0000000000e1',p.id,100*p.position,0,0,100*p.position,p.name,r.merchant_name,r.receipt_date,
  case when p.position%5=0 then 'PAID' else 'OWED' end,case when p.position%5=0 then now() end,now()-(p.position||' minutes')::interval-(random()*1000||' seconds')::interval,case when p.position%2=0 then 'ZELLE' else 'VENMO' end
 from public.receipt_participants p join public.receipts r on r.id=p.receipt_id where r.owner_id='00000000-0000-4000-8000-0000000000e1';
insert into public.receivables(receipt_id,owner_id,participant_id,items_minor,tax_minor,tip_minor,amount_owed_minor,participant_name,merchant_name,receipt_date)
 values('00000000-0000-4000-9000-00000000ffff','00000000-0000-4000-8000-0000000000e2','00000000-0000-4000-a000-00000000ffff',700,0,0,700,'Stranger','Other cafe',date '2026-02-01');
create temp table seen_ids(id uuid primary key);
analyze public.receivables;
DO $$ declare o uuid:='00000000-0000-4000-8000-0000000000e1';x uuid:='00000000-0000-4000-8000-0000000000e2';
 res jsonb;cur jsonb:=null;row jsonb;pages int:=0;plan text:='';line text;want bigint;
begin
 -- paging visits every row exactly once, with a row being marked paid between pages
 loop
  res:=public.receivables_search(o,null,null,null,null,50,cur);pages:=pages+1;
  for row in select * from jsonb_array_elements(res->'receivables') loop
   insert into seen_ids values((row->>'id')::uuid); -- a duplicate violates the primary key
  end loop;
  if pages=2 then perform public.receivable_set_paid(o,(res->'receivables'->0->>'id')::uuid,true);end if;
  exit when jsonb_typeof(res->'next')='null' or pages>500;
  cur:=res->'next';
 end loop;
 if (select count(*) from seen_ids)<>20000 or pages<>400 then raise exception 'paging coverage % rows % pages',(select count(*) from seen_ids),pages;end if;
 -- limit is clamped
 if jsonb_array_length(public.receivables_search(o,null,null,null,null,5000,null)->'receivables')<>50 then raise exception 'limit not clamped';end if;
 -- wildcards are literal
 if jsonb_array_length(public.receivables_search(o,null,null,null,'%',50,null)->'receivables')<>0 then raise exception 'percent acted as wildcard';end if;
 if jsonb_array_length(public.receivables_search(o,null,null,null,'_e',50,null)->'receivables')<>0 then raise exception 'underscore acted as wildcard';end if;
 -- 2 characters use prefix, 3+ use contains; merchant search works; filters combine
 if jsonb_array_length(public.receivables_search(o,null,null,null,'Pe',50,null)->'receivables')<>50 then raise exception 'prefix search';end if;
 if jsonb_array_length(public.receivables_search(o,null,null,null,'cafe 799',50,null)->'receivables')<>25 then raise exception 'merchant search';end if;
 if jsonb_array_length(public.receivables_search(o,'PAID',date '2026-01-08',date '2026-01-08','cafe',50,null)->'receivables')<>5 then raise exception 'combined filters';end if;
 -- outstanding is the exact integer sum over the filtered set, not the page
 select coalesce(sum(amount_owed_minor),0) into want from public.receivables where owner_id=o and status='OWED' and merchant_name='Cafe 799';
 if (public.receivables_search(o,null,null,null,'cafe 799',5,null)->>'outstanding_minor')::bigint<>want then raise exception 'outstanding not over filtered set';end if;
 -- owner isolation
 if jsonb_array_length(public.receivables_search(x,null,null,null,null,50,null)->'receivables')<>1 or (public.receivables_search(x,null,null,null,'person',50,null)->>'outstanding_minor')::bigint<>0 then raise exception 'cross-owner leak';end if;
 -- indexes are usable (planner forced off sequential scans because the table is small)
 set local enable_seqscan=off;
 for line in execute 'explain select * from public.receivables_filtered(''00000000-0000-4000-8000-0000000000e1'',''OWED'',null,null,null)' loop plan:=plan||line;end loop;
 if plan like '%Seq Scan%' then raise exception 'status filter scans the table: %',plan;end if;
 plan:='';
 for line in execute 'explain select * from public.receivables_filtered(''00000000-0000-4000-8000-0000000000e1'',null,null,null,''cafe 799'')' loop plan:=plan||line;end loop;
 if plan not like '%Index Cond: (owner_id%' then raise exception 'text search is not narrowed by the owner index: %',plan;end if;
 plan:='';
 for line in execute 'explain select * from public.receivables_filtered(''00000000-0000-4000-8000-0000000000e1'',null,date ''2026-01-02'',date ''2026-01-09'',null)' loop plan:=plan||line;end loop;
 if plan not like '%receivables_owner_date%' then raise exception 'date index unused: %',plan;end if;
end $$;
-- payment method: only VENMO/ZELLE are accepted by the database
DO $$ begin
 begin update public.receipt_participants set payment_method='PAYPAL' where id='00000000-0000-4000-a000-00000000ffff';raise exception 'bad method accepted';exception when check_violation then null;end;
end $$;
rollback;
