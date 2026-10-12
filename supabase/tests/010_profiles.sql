begin;
insert into auth.users(id,email,raw_user_meta_data) values ('00000000-0000-4000-8000-000000000001','one@example.invalid','{"full_name":"One"}'), ('00000000-0000-4000-8000-000000000002','two@example.invalid','{"full_name":"Two"}');
update auth.users set raw_user_meta_data=raw_user_meta_data where id in ('00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002');
DO $$ BEGIN
 if (select count(*) from public.profiles where id in ('00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002')) <> 2 then raise exception 'Provisioning failed'; end if;
 begin insert into public.profiles(id) values ('00000000-0000-4000-8000-000000000001'); raise exception 'Duplicate allowed'; exception when unique_violation then null; end;
 begin insert into public.profiles(id) values ('00000000-0000-4000-8000-000000000099'); raise exception 'Orphan allowed'; exception when foreign_key_violation then null; end;
END $$;
set local role authenticated;
select set_config('request.jwt.claim.sub', '00000000-0000-4000-8000-000000000001', true);
DO $$ BEGIN
 if (select count(*) from public.profiles) <> 1 then raise exception 'Cross-user read'; end if;
 if (select display_name from public.profiles) <> 'One' then raise exception 'Wrong owner'; end if;
 begin update public.profiles set display_name='forged' where id='00000000-0000-4000-8000-000000000002'; raise exception 'Write granted'; exception when insufficient_privilege then null; end;
 begin insert into public.profiles(id) values ('00000000-0000-4000-8000-000000000003'); raise exception 'Insert granted'; exception when insufficient_privilege then null; end;
 begin delete from public.profiles; raise exception 'Delete granted'; exception when insufficient_privilege then null; end;
END $$;
select set_config('request.jwt.claim.sub', '00000000-0000-4000-8000-000000000002', true);
DO $$ BEGIN if (select count(*) from public.profiles) <> 1 or (select display_name from public.profiles) <> 'Two' then raise exception 'Second-user isolation'; end if; END $$;
reset role;
set local role anon;
DO $$ BEGIN begin perform * from public.profiles; raise exception 'Anonymous read granted'; exception when insufficient_privilege then null; end; END $$;
reset role;
delete from auth.users where id='00000000-0000-4000-8000-000000000001';
DO $$ BEGIN if exists(select from public.profiles where id='00000000-0000-4000-8000-000000000001') then raise exception 'Cascade failed'; end if; END $$;
rollback;
