begin;
insert into auth.users(id,email,raw_user_meta_data) values ('00000000-0000-4000-8000-0000000000a1','a@example.invalid','{}'), ('00000000-0000-4000-8000-0000000000b2','b@example.invalid','{}');
set local role service_role;
DO $$ DECLARE r jsonb; r2 jsonb; BEGIN
 r := public.banking_enroll('00000000-0000-4000-8000-0000000000a1','10000000-0000-4000-8000-000000000001','TELLER','enr_1','Bank A',encode('\x01'::bytea || decode(repeat('ab',40),'hex'),'base64'),1);
 if (r->>'created')::boolean is not true then raise exception 'first enroll not created'; end if;
 r2 := public.banking_enroll('00000000-0000-4000-8000-0000000000a1','10000000-0000-4000-8000-000000000002','TELLER','enr_1','Bank A',encode(decode(repeat('cd',40),'hex'),'base64'),1);
 if (r2->>'created')::boolean or r2->'connection'->>'id' <> '10000000-0000-4000-8000-000000000001' then raise exception 'duplicate enroll not idempotent'; end if;
 if r::text like '%credential%' then raise exception 'credential in public shape'; end if;
 if jsonb_array_length(public.banking_connections('00000000-0000-4000-8000-0000000000b2')) <> 0 then raise exception 'B lists A'; end if;
 if public.banking_credential('00000000-0000-4000-8000-0000000000b2','10000000-0000-4000-8000-000000000001') <> 'null'::jsonb then raise exception 'B reads A credential'; end if;
 if public.banking_disconnect('00000000-0000-4000-8000-0000000000b2','10000000-0000-4000-8000-000000000001') then raise exception 'B disconnected A'; end if;
 begin perform public.banking_enroll('00000000-0000-4000-8000-0000000000a1','10000000-0000-4000-8000-000000000003','TELLER','enr_2','x','AA==',1); raise exception 'short credential allowed'; exception when check_violation then null; end;
END $$;
reset role;
set local role authenticated;
select set_config('request.jwt.claim.sub','00000000-0000-4000-8000-0000000000b2',true);
DO $$ BEGIN
 if (select count(*) from public.institution_connections) <> 0 then raise exception 'RLS cross-user read'; end if;
 begin perform encrypted_credential from public.institution_connections; raise exception 'credential column readable'; exception when insufficient_privilege then null; end;
 begin insert into public.institution_connections(id,user_id,provider,provider_connection_id,encrypted_credential,key_version) values (gen_random_uuid(),'00000000-0000-4000-8000-0000000000b2','TELLER','x',decode(repeat('ab',40),'hex'),1); raise exception 'browser insert granted'; exception when insufficient_privilege then null; end;
 begin perform public.banking_connections('00000000-0000-4000-8000-0000000000a1'); raise exception 'rpc granted'; exception when insufficient_privilege then null; end;
END $$;
select set_config('request.jwt.claim.sub','00000000-0000-4000-8000-0000000000a1',true);
DO $$ BEGIN if (select count(*) from public.institution_connections) <> 1 then raise exception 'owner cannot read own'; end if; END $$;
reset role;
set local role anon;
DO $$ BEGIN begin perform id from public.institution_connections; raise exception 'anon read'; exception when insufficient_privilege then null; end; END $$;
reset role;
set local role service_role;
DO $$ BEGIN
 if not public.banking_disconnect('00000000-0000-4000-8000-0000000000a1','10000000-0000-4000-8000-000000000001') then raise exception 'owner disconnect failed'; end if;
 if public.banking_credential('00000000-0000-4000-8000-0000000000a1','10000000-0000-4000-8000-000000000001') <> 'null'::jsonb then raise exception 'credential survived'; end if;
END $$;
reset role;
delete from auth.users where id='00000000-0000-4000-8000-0000000000a1';
rollback;
