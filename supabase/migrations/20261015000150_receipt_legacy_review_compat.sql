-- Ordered before normalization002. Preserve contract-valid legacy free-text
-- dates/incomplete drafts rather than failing the entire forward migration.
-- On databases already normalized (draft/preview columns gone), this is a data
-- no-op. Existing applied migrations are unchanged.
create table public.receipt_legacy_review_audit (
 receipt_id uuid primary key references public.receipts(id) on delete cascade,
 owner_id uuid not null references public.profiles(id) on delete cascade,
 old_value jsonb not null,
 new_value jsonb not null,
 review_reasons text[] not null,
 created_at timestamptz not null default now()
);
alter table public.receipt_legacy_review_audit enable row level security;
revoke all on public.receipt_legacy_review_audit from anon,authenticated,service_role;
grant select on public.receipt_legacy_review_audit to authenticated;
grant select,insert on public.receipt_legacy_review_audit to service_role;
create policy receipt_legacy_audit_owner_read on public.receipt_legacy_review_audit for select to authenticated using(owner_id=(select auth.uid()));

DO $$
declare
 r record;original_draft jsonb;original_preview jsonb;d jsonb;v jsonb;
 person jsonb;line jsonb;share jsonb;people jsonb;items jsonb;shares jsonb;mapping jsonb;
 mapped_id text;raw_date text;safe_date text;parsed_date date;matches integer;
 needs_review boolean;reasons text[];
begin
 if not exists(select from information_schema.columns where table_schema='public' and table_name='receipts' and column_name='draft') then return;end if;
 -- Dynamic query is intentionally not planned on already-normalized databases.
 for r in execute 'select id,owner_id,draft,preview from public.receipts where draft is not null for update' loop
  original_draft:=r.draft;original_preview:=r.preview;d:=r.draft;v:=r.preview;
  if jsonb_typeof(v)='null' then v:=null;end if;
  needs_review:=false;reasons:=array['IDENTIFIERS_UPGRADED'];
  raw_date:=coalesce(d->>'date','');safe_date:='';
  if raw_date ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' then
   begin
    parsed_date:=raw_date::date;
    if to_char(parsed_date,'YYYY-MM-DD')=raw_date then safe_date:=raw_date;end if;
   exception when invalid_datetime_format or datetime_field_overflow then safe_date:='';end;
  end if;
  if raw_date<>safe_date then needs_review:=true;reasons:=array_append(reasons,'DATE_REVIEW');end if;
  d:=jsonb_set(d,'{date}',to_jsonb(safe_date));
  if coalesce(d->>'tax_method','') not in ('','equal','proportional') then
   d:=jsonb_set(d,'{tax_method}','""');needs_review:=true;reasons:=array_append(reasons,'TAX_METHOD_REVIEW');end if;
  if coalesce(d->>'tip_method','') not in ('','equal','proportional') then
   d:=jsonb_set(d,'{tip_method}','""');needs_review:=true;reasons:=array_append(reasons,'TIP_METHOD_REVIEW');end if;
  -- Old IDs were opaque case-sensitive strings and could repeat across receipts,
  -- collide after UUID conversion, or repeat for participants in incomplete drafts.
  -- Fresh internal UUIDs preserve every line/name; the original exact IDs remain
  -- in audit. Never guess which duplicate/unknown participant an item meant.
  people:='[]';mapping:='[]';
  for person in select value from jsonb_array_elements(d->'participants') loop
   mapped_id:=gen_random_uuid()::text;
   mapping:=mapping||jsonb_build_array(jsonb_build_object('old_id',person->>'id','new_id',mapped_id));
   people:=people||jsonb_build_array(jsonb_set(person,'{id}',to_jsonb(mapped_id)));
  end loop;
  items:='[]';
  for line in select value from jsonb_array_elements(d->'items') loop
   line:=jsonb_set(line,'{id}',to_jsonb(gen_random_uuid()::text));
   if coalesce(line->>'participant_id','')<>'' then
    select count(*),min(value->>'new_id') into matches,mapped_id from jsonb_array_elements(mapping) where value->>'old_id'=line->>'participant_id';
    if matches=1 then line:=jsonb_set(line,'{participant_id}',to_jsonb(mapped_id));
    else line:=jsonb_set(line,'{participant_id}','""');needs_review:=true;reasons:=array_append(reasons,'ASSIGNMENT_REVIEW');end if;
   end if;
   items:=items||jsonb_build_array(line);
  end loop;
  d:=jsonb_set(jsonb_set(d,'{items}',items),'{participants}',people);
  if v is not null then
   shares:='[]';
   for share in select value from jsonb_array_elements(v->'shares') loop
    select count(*),min(value->>'new_id') into matches,mapped_id from jsonb_array_elements(mapping) where value->>'old_id'=share->>'participant_id';
    if matches=1 then shares:=shares||jsonb_build_array(jsonb_set(share,'{participant_id}',to_jsonb(mapped_id)));
    else needs_review:=true;reasons:=array_append(reasons,'PREVIEW_REVIEW');end if;
   end loop;
   v:=jsonb_set(v,'{shares}',shares);
  end if;
  -- Exact historical shares remain audit evidence. A changed/ambiguous field must
  -- not retain a currently verified preview until the owner explicitly reviews.
  if needs_review then v:=null;end if;
  insert into public.receipt_legacy_review_audit(receipt_id,owner_id,old_value,new_value,review_reasons)
  values(r.id,r.owner_id,jsonb_build_object('draft',original_draft,'preview',original_preview),jsonb_build_object('draft',d,'preview',v),reasons);
  execute 'update public.receipts set draft=$1,preview=$2,error_code=case when $3 then ''LEGACY_REVIEW_REQUIRED'' else error_code end where id=$4'
  using d,v,needs_review,r.id;
 end loop;
end $$;

-- Compatible with both sides of002: JSON projection avoids referring to a column
-- that does not exist yet. Only explicit verified recalculation resolves the flag.
create function public.receipt_legacy_review_resolved() returns trigger
language plpgsql security invoker set search_path='' as $$
begin
 if old.error_code='LEGACY_REVIEW_REQUIRED' and coalesce((to_jsonb(new)->>'has_draft_preview')::boolean,false) then new.error_code:=null;end if;
 return new;
end $$;
create trigger receipt_legacy_review_resolved before update of version on public.receipts for each row execute function public.receipt_legacy_review_resolved();
revoke all on function public.receipt_legacy_review_resolved() from public,anon,authenticated;
grant execute on function public.receipt_legacy_review_resolved() to service_role;
