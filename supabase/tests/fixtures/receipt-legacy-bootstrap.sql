-- Minimal Auth/Storage schema for a disposable forward-migration test database.
-- Actual Auth/Storage API and RLS behavior is separately tested on real Supabase.
create schema auth;
create table auth.users(id uuid primary key,email text,raw_user_meta_data jsonb);
create function auth.uid() returns uuid language sql stable as $$select nullif(current_setting('request.jwt.claim.sub',true),'')::uuid$$;
create schema storage;
create table storage.buckets(id text primary key,name text,public boolean,file_size_limit bigint,allowed_mime_types text[]);
create table storage.objects(id uuid primary key default gen_random_uuid(),bucket_id text references storage.buckets(id),name text);
alter table storage.objects enable row level security;
grant usage on schema auth,storage to anon,authenticated,service_role;
grant select on storage.objects to anon,authenticated;
