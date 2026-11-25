-- VP-029: collections group a user's receipts ("Trip A"). A collection with one member is a private
-- category; once others join through an invite link it is a Trip: members-only, signed-in claims.
-- Depends on 20261117000100 (receipt_item_split). All access goes through service-role functions.
create table public.collections (
 id uuid primary key default gen_random_uuid(),
 owner_id uuid not null references public.profiles(id) on delete cascade,
 name text not null check(length(btrim(name)) between 1 and 80),
 created_at timestamptz not null default now()
);
create index collections_owner on public.collections(owner_id);
create table public.collection_members (
 collection_id uuid not null references public.collections(id) on delete cascade,
 user_id uuid not null references public.profiles(id) on delete cascade,
 joined_at timestamptz not null default now(),
 primary key(collection_id,user_id)
);
create index collection_members_user on public.collection_members(user_id);
create table public.collection_invites (
 id uuid primary key default gen_random_uuid(),
 collection_id uuid not null references public.collections(id) on delete cascade,
 token_hash text not null unique check(token_hash ~ '^[a-f0-9]{64}$'),
 expires_at timestamptz not null,
 revoked_at timestamptz,
 created_at timestamptz not null default now()
);
alter table public.receipts add column collection_id uuid references public.collections(id) on delete set null;
create index receipts_collection on public.receipts(collection_id) where collection_id is not null;
alter table public.receipt_participants add column user_id uuid references public.profiles(id) on delete set null;
create unique index receipt_participants_user on public.receipt_participants(receipt_id,user_id) where user_id is not null;
alter table public.collections enable row level security;
alter table public.collection_members enable row level security;
alter table public.collection_invites enable row level security;
revoke all on public.collections,public.collection_members,public.collection_invites from anon,authenticated;
grant select,insert,update,delete on public.collections,public.collection_members,public.collection_invites to service_role;
grant select on public.profiles to service_role; -- trip_detail/trip_list read member names; Foundation-owned table, needs Integration Review

-- The owner's receipt document also reports which collection it is filed in.
create or replace function public.receipt_document(p_owner uuid,p_id uuid) returns jsonb
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
  'items',coalesce((select jsonb_agg(jsonb_build_object('id',i.id,'name',i.description,'quantity',i.quantity,'amount_minor',i.total_price_minor,'participant_ids',coalesce((select jsonb_agg(a.participant_id order by a.participant_id) from public.allocations a where a.receipt_item_id=i.id),'[]'::jsonb)) order by i.position)
    from public.receipt_items i where i.receipt_id=r.id),'[]'::jsonb),
  'participants',coalesce((select jsonb_agg(jsonb_build_object('id',p.id,'name',p.name,'included',p.included,'is_owner',p.is_owner) order by p.position) from public.receipt_participants p where p.receipt_id=r.id),'[]'::jsonb));
 end if;
 if r.has_draft_preview then
  v:=jsonb_build_object('draft',true,'total_minor',r.total_minor,'shares',coalesce((select jsonb_agg(jsonb_build_object('participant_id',s.participant_id,'items_minor',s.items_minor,'tax_minor',s.tax_minor,'tip_minor',s.tip_minor,'total_minor',s.total_minor) order by s.participant_id) from public.receipt_draft_shares s where s.receipt_id=r.id),'[]'::jsonb));
 end if;
 return jsonb_build_object('id',r.id,'owner_id',r.owner_id,'image_path',r.image_path,'image_hash',r.image_hash,'image_mime',r.image_mime,'status',r.status,'version',r.version,'proposal',proposal,'draft',d,'preview',v,'error_code',r.error_code,'retries',r.retries,'created_at',r.created_at,'collection_id',r.collection_id);
end $$;

create function public.trip_create(p_owner uuid,p_name text) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare c public.collections;
begin
 perform pg_advisory_xact_lock(hashtextextended(p_owner::text,809));
 if length(btrim(coalesce(p_name,''))) not between 1 and 80 then raise exception 'TRIP_INVALID_NAME';end if;
 if (select count(*) from public.collections where owner_id=p_owner)>=50 then raise exception 'TRIP_LIMIT';end if;
 insert into public.collections(owner_id,name) values(p_owner,btrim(p_name)) returning * into c;
 insert into public.collection_members(collection_id,user_id) values(c.id,p_owner);
 return jsonb_build_object('id',c.id,'name',c.name);
end $$;

create function public.trip_list(p_user uuid) returns jsonb
language sql security invoker set search_path='' as $$
 select coalesce(jsonb_agg(jsonb_build_object('id',c.id,'name',c.name,'is_owner',c.owner_id=p_user,
  'members',(select count(*) from public.collection_members m where m.collection_id=c.id),
  'receipts',(select count(*) from public.receipts r where r.collection_id=c.id),
  'total_minor',(select coalesce(sum(r.total_minor),0) from public.receipts r where r.collection_id=c.id and r.status in ('NEEDS_REVIEW','FINALIZED'))
 ) order by c.created_at desc),'[]'::jsonb)
 from public.collections c join public.collection_members me on me.collection_id=c.id and me.user_id=p_user;
$$;

create function public.trip_detail(p_user uuid,p_id uuid) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare c public.collections;
begin
 select x.* into c from public.collections x join public.collection_members m on m.collection_id=x.id and m.user_id=p_user where x.id=p_id;
 if not found then raise exception 'TRIP_NOT_FOUND';end if;
 return jsonb_build_object('id',c.id,'name',c.name,'is_owner',c.owner_id=p_user,
  'members',(select coalesce(jsonb_agg(jsonb_build_object('user_id',m.user_id,'name',coalesce(nullif(btrim(p.display_name),''),nullif(split_part(p.email,'@',1),''),'Member'),'is_owner',m.user_id=c.owner_id) order by m.joined_at),'[]'::jsonb)
   from public.collection_members m join public.profiles p on p.id=m.user_id where m.collection_id=c.id),
  'receipts',(select coalesce(jsonb_agg(jsonb_build_object('id',r.id,'merchant',r.merchant_name,'date',coalesce(r.receipt_date::text,''),'status',r.status,'total_minor',r.total_minor,'mine',r.owner_id=p_user) order by r.created_at desc),'[]'::jsonb)
   from (select * from public.receipts where collection_id=c.id and status in ('NEEDS_REVIEW','FINALIZED') order by created_at desc limit 100) r));
end $$;

create function public.trip_delete(p_owner uuid,p_id uuid) returns jsonb
language plpgsql security invoker set search_path='' as $$
begin
 delete from public.collections where id=p_id and owner_id=p_owner;
 if not found then raise exception 'TRIP_NOT_FOUND';end if;
 return jsonb_build_object('deleted',true);
end $$;

-- One live invite link per trip: creating a new one revokes the previous.
create function public.trip_invite(p_owner uuid,p_id uuid,p_hash text,p_expires timestamptz) returns jsonb
language plpgsql security invoker set search_path='' as $$
begin
 if not exists(select 1 from public.collections where id=p_id and owner_id=p_owner) then raise exception 'TRIP_NOT_FOUND';end if;
 update public.collection_invites set revoked_at=now() where collection_id=p_id and revoked_at is null;
 insert into public.collection_invites(collection_id,token_hash,expires_at) values(p_id,p_hash,p_expires);
 return jsonb_build_object('expires_at',p_expires);
end $$;

create function public.trip_join(p_user uuid,p_hash text,p_now timestamptz) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare i public.collection_invites;c public.collections;
begin
 select * into i from public.collection_invites where token_hash=p_hash and revoked_at is null and p_now<expires_at;
 if not found then raise exception 'TRIP_INVITE_INVALID';end if;
 select * into c from public.collections where id=i.collection_id for update;
 if not exists(select 1 from public.collection_members where collection_id=c.id and user_id=p_user) then
  if (select count(*) from public.collection_members where collection_id=c.id)>=40 then raise exception 'TRIP_FULL';end if;
  insert into public.collection_members(collection_id,user_id) values(c.id,p_user);
 end if;
 return jsonb_build_object('id',c.id,'name',c.name);
end $$;

-- The owner can remove anyone else; any other member can only remove themselves.
create function public.trip_remove_member(p_actor uuid,p_id uuid,p_user uuid) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare c public.collections;
begin
 select * into c from public.collections where id=p_id;
 if not found or not exists(select 1 from public.collection_members where collection_id=p_id and user_id=p_actor) then raise exception 'TRIP_NOT_FOUND';end if;
 if p_user=c.owner_id or (p_actor<>c.owner_id and p_actor<>p_user) then raise exception 'TRIP_FORBIDDEN';end if;
 delete from public.collection_members where collection_id=p_id and user_id=p_user;
 return jsonb_build_object('removed',true);
end $$;

-- Only the receipt's owner files it, and only into a trip they belong to (null un-files).
create function public.receipt_set_collection(p_owner uuid,p_receipt uuid,p_collection uuid) returns jsonb
language plpgsql security invoker set search_path='' as $$
begin
 if p_collection is not null and not exists(select 1 from public.collection_members where collection_id=p_collection and user_id=p_owner) then raise exception 'TRIP_NOT_FOUND';end if;
 update public.receipts set collection_id=p_collection where id=p_receipt and owner_id=p_owner;
 if not found then raise exception 'RECEIPT_NOT_FOUND';end if;
 return jsonb_build_object('id',p_receipt,'collection_id',p_collection);
end $$;

-- Membership check for reading a trip receipt: returns its owner and the member's participant id.
create function public.trip_receipt_access(p_user uuid,p_trip uuid,p_receipt uuid) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;pid uuid;
begin
 select * into r from public.receipts where id=p_receipt and collection_id=p_trip and status in ('NEEDS_REVIEW','FINALIZED');
 if not found or not exists(select 1 from public.collection_members where collection_id=p_trip and user_id=p_user) then raise exception 'TRIP_NOT_FOUND';end if;
 select id into pid from public.receipt_participants where receipt_id=r.id and (user_id=p_user or (is_owner and r.owner_id=p_user));
 return jsonb_build_object('owner_id',r.owner_id,'participant_id',pid);
end $$;

-- A signed-in member claims (or releases) an item. Same allocation rules as the guest split_claim.
create function public.trip_claim(p_user uuid,p_trip uuid,p_receipt uuid,p_item uuid,p_release boolean) returns jsonb
language plpgsql security invoker set search_path='' as $$
declare r public.receipts;pid uuid;n integer;nm text;
begin
 select * into r from public.receipts where id=p_receipt and collection_id=p_trip for update;
 if not found or not exists(select 1 from public.collection_members where collection_id=p_trip and user_id=p_user) then raise exception 'TRIP_NOT_FOUND';end if;
 if r.status<>'NEEDS_REVIEW' or not r.has_owner_draft then raise exception 'SPLIT_NOT_READY';end if;
 select id into pid from public.receipt_participants where receipt_id=r.id and (user_id=p_user or (is_owner and r.owner_id=p_user));
 if pid is null then
  select count(*) into n from public.receipt_participants where receipt_id=r.id;
  if n>=40 then raise exception 'SPLIT_FULL';end if;
  select coalesce(nullif(btrim(display_name),''),nullif(split_part(email,'@',1),''),'Member') into nm from public.profiles where id=p_user;
  pid:=gen_random_uuid();
  insert into public.receipt_participants(id,receipt_id,name,included,position,user_id) values(pid,r.id,left(nm,100),true,n,p_user);
 end if;
 if not exists(select 1 from public.receipt_items where receipt_id=r.id and id=p_item) then raise exception 'SPLIT_ITEM_NOT_FOUND';end if;
 if p_release then
  delete from public.allocations where receipt_item_id=p_item and participant_id=pid;
 else
  insert into public.allocations(receipt_id,receipt_item_id,participant_id,amount_minor) values(r.id,p_item,pid,0) on conflict do nothing;
 end if;
 perform public.receipt_item_split(p_item);
 update public.receipts set version=version+1,has_draft_preview=false,updated_at=now() where id=r.id;
 delete from public.receipt_draft_shares where receipt_id=r.id;
 return jsonb_build_object('receipt_id',r.id,'participant_id',pid);
end $$;

revoke all on function public.trip_create(uuid,text),public.trip_list(uuid),public.trip_detail(uuid,uuid),public.trip_delete(uuid,uuid),
 public.trip_invite(uuid,uuid,text,timestamptz),public.trip_join(uuid,text,timestamptz),public.trip_remove_member(uuid,uuid,uuid),
 public.receipt_set_collection(uuid,uuid,uuid),public.trip_receipt_access(uuid,uuid,uuid),public.trip_claim(uuid,uuid,uuid,uuid,boolean) from public,anon,authenticated;
grant execute on function public.trip_create(uuid,text),public.trip_list(uuid),public.trip_detail(uuid,uuid),public.trip_delete(uuid,uuid),
 public.trip_invite(uuid,uuid,text,timestamptz),public.trip_join(uuid,text,timestamptz),public.trip_remove_member(uuid,uuid,uuid),
 public.receipt_set_collection(uuid,uuid,uuid),public.trip_receipt_access(uuid,uuid,uuid),public.trip_claim(uuid,uuid,uuid,uuid,boolean) to service_role;
