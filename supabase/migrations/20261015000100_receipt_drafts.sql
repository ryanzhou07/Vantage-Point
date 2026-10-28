-- Receipt-owned draft slice. No browser mutation grants and no finalized state.
create table public.receipts (
 id uuid primary key default gen_random_uuid(),
 owner_id uuid not null references public.profiles(id) on delete cascade,
 image_path text not null unique,
 image_hash text not null check (image_hash ~ '^[a-f0-9]{64}$'),
 image_mime text not null check (image_mime in ('image/png','image/jpeg')),
 status text not null default 'DRAFT' check(status in ('DRAFT','PROCESSING','NEEDS_REVIEW','PROCESSING_FAILED')),
 version bigint not null default 1 check(version>0),
 proposal jsonb,
 draft jsonb,
 preview jsonb,
 error_code text,
 retries integer not null default 0 check(retries between 0 and 2),
 created_at timestamptz not null default now(),
 updated_at timestamptz not null default now(),
 unique(owner_id,image_hash),
 check(octet_length(coalesce(proposal::text,''))<=131072),
 check(octet_length(coalesce(draft::text,''))<=131072),
 check(octet_length(coalesce(preview::text,''))<=32768)
);
create index receipts_owner_created on public.receipts(owner_id,created_at desc,id);
create table public.receipt_vision_jobs (
 receipt_id uuid primary key references public.receipts(id) on delete cascade,
 state text not null default 'QUEUED' check(state in ('QUEUED','RUNNING','DONE','FAILED')),
 attempts integer not null default 0 check(attempts between 0 and 9),
 available_at timestamptz not null default now(),
 lease_until timestamptz,
 lease_token uuid,
 updated_at timestamptz not null default now()
);
create index receipt_jobs_ready on public.receipt_vision_jobs(available_at) where state in ('QUEUED','RUNNING');
alter table public.receipts enable row level security;
alter table public.receipt_vision_jobs enable row level security;
revoke all on public.receipts,public.receipt_vision_jobs from anon,authenticated;
grant select on public.receipts to authenticated;
grant all on public.receipts,public.receipt_vision_jobs to service_role;
create policy receipt_owner_read on public.receipts for select to authenticated using(owner_id=(select auth.uid()));
insert into storage.buckets(id,name,public,file_size_limit,allowed_mime_types)
 values('receipt-images','receipt-images',false,5242880,array['image/png','image/jpeg']);
create policy receipt_image_owner_read on storage.objects for select to authenticated
 using(bucket_id='receipt-images' and exists(select 1 from public.receipts r where r.image_path=name and r.owner_id=(select auth.uid())));

-- Service-only entry points. p_owner comes exclusively from verified Foundation context.
create function public.receipt_create(p_owner uuid,p_hash text,p_mime text) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts; rid uuid;
begin
 perform pg_advisory_xact_lock(hashtextextended(p_owner::text,808));
 select * into r from public.receipts where owner_id=p_owner and image_hash=p_hash;
 if found then return to_jsonb(r); end if;
 if (select count(*) from public.receipts where owner_id=p_owner and created_at>now()-interval '1 day')>=20
 or (select count(*) from public.receipts where owner_id=p_owner and status in ('DRAFT','PROCESSING'))>=5 then
 raise exception 'RECEIPT_QUOTA' using errcode='P0001'; end if;
 rid:=gen_random_uuid();
 insert into public.receipts(id,owner_id,image_path,image_hash,image_mime)
 values(rid,p_owner,p_owner::text||'/'||rid::text||case when p_mime='image/png' then '.png' else '.jpg' end,p_hash,p_mime) returning * into r;
 return to_jsonb(r);
end $$;
create function public.receipt_enqueue(p_owner uuid,p_id uuid) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;
begin
 select * into r from public.receipts where owner_id=p_owner and id=p_id for update;
 if not found then raise exception 'RECEIPT_NOT_FOUND';end if;
 if r.status='DRAFT' then
 update public.receipts set status='PROCESSING',version=version+1,updated_at=now() where id=p_id returning * into r;
 insert into public.receipt_vision_jobs(receipt_id) values(p_id) on conflict do nothing;
 end if;
 return to_jsonb(r);
end $$;
create function public.receipt_save(p_owner uuid,p_id uuid,p_version bigint,p_draft jsonb,p_preview jsonb) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;
begin
 select * into r from public.receipts where owner_id=p_owner and id=p_id for update;
 if not found then raise exception 'RECEIPT_NOT_FOUND';end if;
 if r.version<>p_version then raise exception 'RECEIPT_CONFLICT';end if;
 if r.status<>'NEEDS_REVIEW' then raise exception 'RECEIPT_NOT_REVIEWABLE';end if;
 update public.receipts set draft=p_draft,preview=p_preview,version=version+1,updated_at=now() where id=p_id returning * into r;
 return to_jsonb(r);
end $$;
create function public.receipt_retry(p_owner uuid,p_id uuid,p_version bigint) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;
begin
 select * into r from public.receipts where owner_id=p_owner and id=p_id for update;
 if not found then raise exception 'RECEIPT_NOT_FOUND';end if;
 if r.version<>p_version then raise exception 'RECEIPT_CONFLICT';end if;
 if r.status<>'PROCESSING_FAILED' or r.retries>=2 then raise exception 'RECEIPT_RETRY_LIMIT';end if;
 update public.receipts set status='PROCESSING',error_code=null,retries=retries+1,version=version+1,updated_at=now() where id=p_id returning * into r;
 update public.receipt_vision_jobs set state='QUEUED',available_at=now()+interval '5 seconds',lease_until=null,lease_token=null where receipt_id=p_id;
 return to_jsonb(r);
end $$;
create function public.receipt_claim() returns jsonb
language plpgsql security invoker set search_path='' as $$
declare j public.receipt_vision_jobs;r public.receipts;
begin
 perform pg_advisory_xact_lock(808008);
 if (select count(*) from public.receipt_vision_jobs where state='RUNNING' and lease_until>now())>=2 then return null;end if;
 select * into j from public.receipt_vision_jobs
 where (state='QUEUED' and available_at<=now()) or(state='RUNNING' and lease_until<now())
 order by available_at,receipt_id for update skip locked limit 1;
 if not found then return null;end if;
 if j.attempts>=9 then
 update public.receipt_vision_jobs set state='FAILED' where receipt_id=j.receipt_id;
 update public.receipts set status='PROCESSING_FAILED',error_code='VISION_RETRY_EXHAUSTED',version=version+1,updated_at=now() where id=j.receipt_id;
 return null;end if;
 update public.receipt_vision_jobs set state='RUNNING',attempts=attempts+1,lease_token=gen_random_uuid(),lease_until=now()+interval '90 seconds',updated_at=now() where receipt_id=j.receipt_id returning * into j;
 select * into r from public.receipts where id=j.receipt_id;
 return jsonb_build_object('receipt',to_jsonb(r),'lease_token',j.lease_token,'attempts',j.attempts);
end $$;
create function public.receipt_finish(p_id uuid,p_lease uuid,p_proposal jsonb,p_error text,p_retry boolean) returns boolean
language plpgsql security invoker set search_path='' as $$
declare j public.receipt_vision_jobs;
begin
 select * into j from public.receipt_vision_jobs where receipt_id=p_id for update;
 if not found or j.state<>'RUNNING' or j.lease_token<>p_lease or j.lease_until<now() then return false;end if;
 if p_error is null then
 update public.receipt_vision_jobs set state='DONE',lease_until=null where receipt_id=p_id;
 update public.receipts set status='NEEDS_REVIEW',proposal=p_proposal,error_code=null,version=version+1,updated_at=now() where id=p_id;
 elsif p_retry and j.attempts%3<>0 and j.attempts<9 then
 update public.receipt_vision_jobs set state='QUEUED',available_at=now()+make_interval(secs=>power(2,j.attempts%3)::int+random()*3),lease_until=null where receipt_id=p_id;
 else
 update public.receipt_vision_jobs set state='FAILED',lease_until=null where receipt_id=p_id;
 update public.receipts set status='PROCESSING_FAILED',error_code=p_error,version=version+1,updated_at=now() where id=p_id;
 end if;
 return true;
end $$;
revoke all on function public.receipt_create(uuid,text,text),public.receipt_enqueue(uuid,uuid),public.receipt_save(uuid,uuid,bigint,jsonb,jsonb),public.receipt_retry(uuid,uuid,bigint),public.receipt_claim(),public.receipt_finish(uuid,uuid,jsonb,text,boolean) from public,anon,authenticated;
grant execute on function public.receipt_create(uuid,text,text),public.receipt_enqueue(uuid,uuid),public.receipt_save(uuid,uuid,bigint,jsonb,jsonb),public.receipt_retry(uuid,uuid,bigint),public.receipt_claim(),public.receipt_finish(uuid,uuid,jsonb,text,boolean) to service_role;
