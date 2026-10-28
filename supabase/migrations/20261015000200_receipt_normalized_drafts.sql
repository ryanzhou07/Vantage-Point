-- Forward-only correction: advisory vision JSON is retained; reviewed money,
-- items, participants, assignments and preview shares are relational authority.
alter table public.receipts
 add column merchant_name text not null default '' check(length(merchant_name)<=200),
 add column receipt_date date,
 add column currency text not null default 'USD' check(currency='USD'),
 add column subtotal_minor bigint not null default 0 check(subtotal_minor between 0 and 100000000),
 add column tax_minor bigint not null default 0 check(tax_minor between 0 and 100000000),
 add column tip_minor bigint not null default 0 check(tip_minor between 0 and 100000000),
 add column discount_minor bigint not null default 0 check(discount_minor between 0 and 100000000),
 add column total_minor bigint not null default 0 check(total_minor between 0 and 100000000),
 add column tax_allocation_method text not null default '' check(tax_allocation_method in ('','equal','proportional')),
 add column tip_allocation_method text not null default '' check(tip_allocation_method in ('','equal','proportional')),
 add column has_owner_draft boolean not null default false,
 add column has_draft_preview boolean not null default false,
 add column revision integer not null default 1 check(revision=1);

create table public.receipt_items (
 id uuid primary key,
 receipt_id uuid not null references public.receipts(id) on delete cascade,
 receipt_revision integer not null default 1 check(receipt_revision=1),
 description text not null check(length(btrim(description)) between 1 and 200),
 quantity integer not null check(quantity between 1 and 1000),
 total_price_minor bigint not null check(total_price_minor between 0 and 100000000),
 position integer not null check(position between 0 and 199),
 created_at timestamptz not null default now(),updated_at timestamptz not null default now(),
 unique(receipt_id,id),
 unique(receipt_id,position) deferrable initially deferred
);
create table public.receipt_participants (
 id uuid primary key,
 receipt_id uuid not null references public.receipts(id) on delete cascade,
 name text not null check(length(btrim(name)) between 1 and 100),
 included boolean not null,
 position integer not null check(position between 0 and 39),
 created_at timestamptz not null default now(),updated_at timestamptz not null default now(),
 unique(receipt_id,id),
 unique(receipt_id,position) deferrable initially deferred
);
create table public.allocations (
 id uuid primary key default gen_random_uuid(),
 receipt_id uuid not null references public.receipts(id) on delete cascade,
 receipt_revision integer not null default 1 check(receipt_revision=1),
 receipt_item_id uuid not null unique,
 participant_id uuid not null,
 amount_minor bigint not null check(amount_minor between 0 and 100000000),
 percentage_bps integer not null default 10000 check(percentage_bps=10000),
 status text not null default 'DRAFT' check(status='DRAFT'),
 source text not null default 'OWNER_DRAFT' check(source='OWNER_DRAFT'),
 created_at timestamptz not null default now(),updated_at timestamptz not null default now(),
 foreign key(receipt_id,receipt_item_id) references public.receipt_items(receipt_id,id) on delete cascade,
 foreign key(receipt_id,participant_id) references public.receipt_participants(receipt_id,id) on delete cascade
);
create index allocations_receipt on public.allocations(receipt_id);
create table public.receipt_draft_shares (
 receipt_id uuid not null references public.receipts(id) on delete cascade,
 participant_id uuid not null,
 items_minor bigint not null check(items_minor between 0 and 100000000),
 tax_minor bigint not null check(tax_minor between 0 and 100000000),
 tip_minor bigint not null check(tip_minor between 0 and 100000000),
 total_minor bigint not null check(total_minor between 0 and 100000000 and total_minor=items_minor+tax_minor+tip_minor),
 primary key(receipt_id,participant_id),
 foreign key(receipt_id,participant_id) references public.receipt_participants(receipt_id,id) on delete cascade
);
create table public.receipt_events (
 id uuid primary key default gen_random_uuid(),
 receipt_id uuid not null references public.receipts(id) on delete cascade,
 receipt_revision integer not null default 1 check(receipt_revision=1),
 event_type text not null check(event_type in ('OWNER_DRAFT_SAVED','NORMALIZED_DRAFT_MIGRATED')),
 actor_user_id uuid not null references public.profiles(id),
 old_value jsonb,new_value jsonb,
 created_at timestamptz not null default now()
);
create index receipt_events_receipt on public.receipt_events(receipt_id,created_at);

alter table public.receipt_items enable row level security;
alter table public.receipt_participants enable row level security;
alter table public.allocations enable row level security;
alter table public.receipt_draft_shares enable row level security;
alter table public.receipt_events enable row level security;
revoke all on public.receipt_items,public.receipt_participants,public.allocations,public.receipt_draft_shares,public.receipt_events from anon,authenticated;
grant select on public.receipt_items,public.receipt_participants,public.allocations,public.receipt_draft_shares,public.receipt_events to authenticated;
grant all on public.receipt_items,public.receipt_participants,public.allocations,public.receipt_draft_shares to service_role;
grant select,insert on public.receipt_events to service_role;
create policy receipt_item_owner_read on public.receipt_items for select to authenticated using(exists(select 1 from public.receipts r where r.id=receipt_id and r.owner_id=(select auth.uid())));
create policy receipt_participant_owner_read on public.receipt_participants for select to authenticated using(exists(select 1 from public.receipts r where r.id=receipt_id and r.owner_id=(select auth.uid())));
create policy receipt_allocation_owner_read on public.allocations for select to authenticated using(exists(select 1 from public.receipts r where r.id=receipt_id and r.owner_id=(select auth.uid())));
create policy receipt_share_owner_read on public.receipt_draft_shares for select to authenticated using(exists(select 1 from public.receipts r where r.id=receipt_id and r.owner_id=(select auth.uid())));
create policy receipt_event_owner_read on public.receipt_events for select to authenticated using(exists(select 1 from public.receipts r where r.id=receipt_id and r.owner_id=(select auth.uid())));

-- Legacy advisory/client keys map to receipt-scoped UUIDs. These identifiers are
-- not security tokens; authorization always uses canonical owner context/RLS.
create function public.receipt_internal_id(p_receipt uuid,p_kind text,p_value text) returns uuid
language sql immutable strict set search_path='' as $$
 select case when p_value ~* '^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$'
 then p_value::uuid else md5(p_receipt::text||':'||p_kind||':'||p_value)::uuid end
$$;
create function public.receipt_allocation_whole_line() returns trigger
language plpgsql security invoker set search_path='' as $$
declare amount bigint;
begin
 select total_price_minor into amount from public.receipt_items where id=new.receipt_item_id and receipt_id=new.receipt_id for update;
 if not found or new.amount_minor<>amount then raise exception 'RECEIPT_INVALID_ALLOCATION' using errcode='23514';end if;
 return new;
end $$;
create trigger receipt_whole_line before insert or update on public.allocations for each row execute function public.receipt_allocation_whole_line();

-- JSON below is a response projection only, never persisted financial authority.
create function public.receipt_document(p_owner uuid,p_id uuid) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;d jsonb;v jsonb;proposal jsonb;
begin
 select * into r from public.receipts where owner_id=p_owner and id=p_id;
 if not found then raise exception 'RECEIPT_NOT_FOUND';end if;
 proposal:=r.proposal;
 if proposal is not null then
  proposal:=jsonb_set(proposal,'{items}',coalesce((select jsonb_agg(jsonb_set(line,'{id}',to_jsonb(public.receipt_internal_id(r.id,'item',coalesce(line->>'id',n::text))::text)) order by n)
  from jsonb_array_elements(proposal->'items') with ordinality as lines(line,n)),'[]'::jsonb));
 end if;
 if r.has_owner_draft then
  d:=jsonb_build_object('merchant',r.merchant_name,'date',coalesce(r.receipt_date::text,''),'currency',r.currency,
  'subtotal_minor',r.subtotal_minor,'tax_minor',r.tax_minor,'tip_minor',r.tip_minor,'discount_minor',r.discount_minor,'total_minor',r.total_minor,
  'tax_method',r.tax_allocation_method,'tip_method',r.tip_allocation_method,
  'items',coalesce((select jsonb_agg(jsonb_build_object('id',i.id,'name',i.description,'quantity',i.quantity,'amount_minor',i.total_price_minor,'participant_id',coalesce(a.participant_id::text,'')) order by i.position)
    from public.receipt_items i left join public.allocations a on a.receipt_item_id=i.id where i.receipt_id=r.id),'[]'::jsonb),
  'participants',coalesce((select jsonb_agg(jsonb_build_object('id',p.id,'name',p.name,'included',p.included) order by p.position) from public.receipt_participants p where p.receipt_id=r.id),'[]'::jsonb));
 end if;
 if r.has_draft_preview then
  v:=jsonb_build_object('draft',true,'total_minor',r.total_minor,'shares',coalesce((select jsonb_agg(jsonb_build_object('participant_id',s.participant_id,'items_minor',s.items_minor,'tax_minor',s.tax_minor,'tip_minor',s.tip_minor,'total_minor',s.total_minor) order by s.participant_id) from public.receipt_draft_shares s where s.receipt_id=r.id),'[]'::jsonb));
 end if;
 return jsonb_build_object('id',r.id,'owner_id',r.owner_id,'image_path',r.image_path,'image_hash',r.image_hash,'image_mime',r.image_mime,'status',r.status,'version',r.version,'proposal',proposal,'draft',d,'preview',v,'error_code',r.error_code,'retries',r.retries,'created_at',r.created_at);
end $$;
create function public.receipt_documents(p_owner uuid) returns jsonb
language sql security invoker set search_path='' as $$
 select coalesce(jsonb_agg(public.receipt_document(p_owner,id) order by created_at desc,id desc),'[]'::jsonb)
 from (select id,created_at from public.receipts where owner_id=p_owner order by created_at desc,id desc limit 20) r
$$;

create or replace function public.receipt_save(p_owner uuid,p_id uuid,p_version bigint,p_draft jsonb,p_preview jsonb) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;line jsonb;person jsonb;share jsonb;n integer;old_document jsonb;item_id uuid;person_id uuid;
begin
 select * into r from public.receipts where owner_id=p_owner and id=p_id for update;
 if not found then raise exception 'RECEIPT_NOT_FOUND';end if;
 if r.version<>p_version or p_version is null then raise exception 'RECEIPT_CONFLICT';end if;
 if r.status<>'NEEDS_REVIEW' then raise exception 'RECEIPT_NOT_REVIEWABLE';end if;
 if p_draft is null or jsonb_typeof(p_draft->'items')<>'array' or jsonb_array_length(p_draft->'items') not between 1 and 200
 or jsonb_typeof(p_draft->'participants')<>'array' or jsonb_array_length(p_draft->'participants')>40 then raise exception 'RECEIPT_INVALID_REVIEW';end if;
 old_document:=public.receipt_document(p_owner,p_id);
 delete from public.receipt_draft_shares where receipt_id=p_id;
 delete from public.allocations where receipt_id=p_id;
 -- Before collaboration, replace the whole owner draft under its row lock.
 -- Stable UUIDs persist; no guest claims/submissions/receivables exist in this slice.
 delete from public.receipt_items where receipt_id=p_id;
 delete from public.receipt_participants where receipt_id=p_id;
 n:=0;
 for person in select value from jsonb_array_elements(p_draft->'participants') loop
  person_id:=public.receipt_internal_id(p_id,'participant',person->>'id');
  insert into public.receipt_participants(id,receipt_id,name,included,position) values(person_id,p_id,person->>'name',(person->>'included')::boolean,n);
  n:=n+1;
 end loop;
 n:=0;
 for line in select value from jsonb_array_elements(p_draft->'items') loop
  item_id:=public.receipt_internal_id(p_id,'item',line->>'id');
  insert into public.receipt_items(id,receipt_id,description,quantity,total_price_minor,position) values(item_id,p_id,line->>'name',(line->>'quantity')::integer,(line->>'amount_minor')::bigint,n);
  if coalesce(line->>'participant_id','')<>'' then
   insert into public.allocations(receipt_id,receipt_item_id,participant_id,amount_minor) values(p_id,item_id,public.receipt_internal_id(p_id,'participant',line->>'participant_id'),(line->>'amount_minor')::bigint);
  end if;
  n:=n+1;
 end loop;
 update public.receipts set merchant_name=p_draft->>'merchant',receipt_date=nullif(p_draft->>'date','')::date,currency=p_draft->>'currency',
 subtotal_minor=(p_draft->>'subtotal_minor')::bigint,tax_minor=(p_draft->>'tax_minor')::bigint,tip_minor=(p_draft->>'tip_minor')::bigint,
 discount_minor=(p_draft->>'discount_minor')::bigint,total_minor=(p_draft->>'total_minor')::bigint,
 tax_allocation_method=p_draft->>'tax_method',tip_allocation_method=p_draft->>'tip_method',
 has_owner_draft=true,has_draft_preview=(p_preview is not null),version=version+1,updated_at=now() where id=p_id returning * into r;
 if p_preview is not null then
  if r.discount_minor<>0 or r.subtotal_minor+r.tax_minor+r.tip_minor<>r.total_minor
  or (select sum(total_price_minor) from public.receipt_items where receipt_id=p_id)<>r.subtotal_minor
  or (select count(*) from public.allocations where receipt_id=p_id)<>n
  or r.tax_allocation_method='' or r.tip_allocation_method='' then raise exception 'RECEIPT_INVALID_REVIEW';end if;
  for share in select value from jsonb_array_elements(p_preview->'shares') loop
   insert into public.receipt_draft_shares(receipt_id,participant_id,items_minor,tax_minor,tip_minor,total_minor)
   values(p_id,public.receipt_internal_id(p_id,'participant',share->>'participant_id'),(share->>'items_minor')::bigint,(share->>'tax_minor')::bigint,(share->>'tip_minor')::bigint,(share->>'total_minor')::bigint);
  end loop;
  if (select count(*) from public.receipt_draft_shares where receipt_id=p_id)<>(select count(*) from public.receipt_participants where receipt_id=p_id)
  or (select coalesce(sum(tax_minor),0) from public.receipt_draft_shares where receipt_id=p_id)<>r.tax_minor
  or (select coalesce(sum(tip_minor),0) from public.receipt_draft_shares where receipt_id=p_id)<>r.tip_minor
  or (select coalesce(sum(total_minor),0) from public.receipt_draft_shares where receipt_id=p_id)<>r.total_minor
  or exists(select 1 from public.receipt_draft_shares s where s.receipt_id=p_id and s.items_minor<>(select coalesce(sum(a.amount_minor),0) from public.allocations a where a.receipt_id=p_id and a.participant_id=s.participant_id))
  or exists(select 1 from public.receipt_participants p join public.allocations a on a.participant_id=p.id where p.receipt_id=p_id and not p.included)
  then raise exception 'RECEIPT_INVALID_REVIEW';end if;
 end if;
 insert into public.receipt_events(receipt_id,event_type,actor_user_id,old_value,new_value) values(p_id,'OWNER_DRAFT_SAVED',p_owner,old_document->'draft',public.receipt_document(p_owner,p_id)->'draft');
 return public.receipt_document(p_owner,p_id);
end $$;

-- Preserve previously saved owner drafts and exact preview cents before dropping
-- the superseded JSON authority. The old snapshot also remains append-only audit.
DO $$ declare r public.receipts;begin
 for r in select * from public.receipts where draft is not null loop
  perform public.receipt_save(r.owner_id,r.id,r.version,r.draft,r.preview);
  insert into public.receipt_events(receipt_id,event_type,actor_user_id,old_value,new_value) values(r.id,'NORMALIZED_DRAFT_MIGRATED',r.owner_id,jsonb_build_object('draft',r.draft,'preview',r.preview),null);
 end loop;
end $$;
alter table public.receipts drop column draft,drop column preview;

-- Existing creation/enqueue/retry RPCs keep their locking/quota behavior and
-- now project relational state for existing records (including duplicate upload).
create or replace function public.receipt_create(p_owner uuid,p_hash text,p_mime text) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;rid uuid;
begin
 perform pg_advisory_xact_lock(hashtextextended(p_owner::text,808));
 select * into r from public.receipts where owner_id=p_owner and image_hash=p_hash;
 if found then return public.receipt_document(p_owner,r.id);end if;
 if (select count(*) from public.receipts where owner_id=p_owner and created_at>now()-interval '1 day')>=20
 or (select count(*) from public.receipts where owner_id=p_owner and status in ('DRAFT','PROCESSING'))>=5 then raise exception 'RECEIPT_QUOTA';end if;
 rid:=gen_random_uuid();
 insert into public.receipts(id,owner_id,image_path,image_hash,image_mime) values(rid,p_owner,p_owner::text||'/'||rid::text||case when p_mime='image/png' then '.png' else '.jpg' end,p_hash,p_mime) returning * into r;
 return public.receipt_document(p_owner,r.id);
end $$;
create or replace function public.receipt_enqueue(p_owner uuid,p_id uuid) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;
begin
 select * into r from public.receipts where owner_id=p_owner and id=p_id for update;
 if not found then raise exception 'RECEIPT_NOT_FOUND';end if;
 if r.status='DRAFT' then
 update public.receipts set status='PROCESSING',version=version+1,updated_at=now() where id=p_id;
 insert into public.receipt_vision_jobs(receipt_id) values(p_id) on conflict do nothing;
 end if;
 return public.receipt_document(p_owner,p_id);
end $$;
create or replace function public.receipt_retry(p_owner uuid,p_id uuid,p_version bigint) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;
begin
 select * into r from public.receipts where owner_id=p_owner and id=p_id for update;
 if not found then raise exception 'RECEIPT_NOT_FOUND';end if;
 if r.version<>p_version or p_version is null then raise exception 'RECEIPT_CONFLICT';end if;
 if r.status<>'PROCESSING_FAILED' or r.retries>=2 then raise exception 'RECEIPT_RETRY_LIMIT';end if;
 update public.receipts set status='PROCESSING',error_code=null,retries=retries+1,version=version+1,updated_at=now() where id=p_id;
 update public.receipt_vision_jobs set state='QUEUED',available_at=now()+interval '5 seconds',lease_until=null,lease_token=null where receipt_id=p_id;
 return public.receipt_document(p_owner,p_id);
end $$;
revoke all on function public.receipt_internal_id(uuid,text,text),public.receipt_allocation_whole_line(),public.receipt_document(uuid,uuid),public.receipt_documents(uuid) from public,anon,authenticated;
grant execute on function public.receipt_internal_id(uuid,text,text),public.receipt_allocation_whole_line(),public.receipt_document(uuid,uuid),public.receipt_documents(uuid) to service_role;
