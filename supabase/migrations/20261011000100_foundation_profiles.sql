create table public.profiles (
  id uuid primary key references auth.users(id) on delete cascade,
  display_name text not null default '' check (length(display_name) <= 200),
  email text not null default '' check (length(email) <= 320),
  avatar_url text not null default '' check (length(avatar_url) <= 2048),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
alter table public.profiles enable row level security;
revoke all on public.profiles from anon, authenticated;
grant select on public.profiles to authenticated;
create policy profiles_owner_read on public.profiles for select to authenticated using ((select auth.uid()) = id);
-- Identity metadata is provisioned by Auth, never by a browser-supplied UUID.
create function public.provision_profile() returns trigger language plpgsql security definer set search_path = '' as $$
begin
  insert into public.profiles(id, display_name, email, avatar_url)
  values(new.id, left(coalesce(new.raw_user_meta_data->>'full_name', ''), 200),
    left(coalesce(new.email, ''), 320), left(coalesce(new.raw_user_meta_data->>'avatar_url', ''), 2048))
  on conflict(id) do nothing;
  return new;
end;
$$;
revoke all on function public.provision_profile() from public, anon, authenticated;
create trigger foundation_profile_created after insert or update of email, raw_user_meta_data on auth.users for each row execute function public.provision_profile();
-- Existing canonical users are backfilled safely on migration.
insert into public.profiles(id, display_name, email, avatar_url)
select id, left(coalesce(raw_user_meta_data->>'full_name',''),200), left(coalesce(email,''),320), left(coalesce(raw_user_meta_data->>'avatar_url',''),2048) from auth.users
on conflict(id) do nothing;
