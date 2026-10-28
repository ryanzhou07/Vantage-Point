begin;
insert into auth.users(id,email,raw_user_meta_data) values
 ('00000000-0000-4000-8000-000000000091','split-one@example.invalid','{}'),
 ('00000000-0000-4000-8000-000000000092','split-two@example.invalid','{}');
select set_config('test.r1',(public.receipt_create('00000000-0000-4000-8000-000000000091',repeat('c',64),'image/png')->>'id'),true);
DO $$ declare a jsonb;b jsonb;t timestamptz:='2026-10-28 12:00:00+00';r uuid:=current_setting('test.r1')::uuid;o uuid:='00000000-0000-4000-8000-000000000091';x uuid:='00000000-0000-4000-8000-000000000092';begin
 a:=public.split_session_open(o,r,repeat('a',64),t,300);
 if (a->>'expires_at')::timestamptz<>t+interval '5 minutes' then raise exception 'ttl';end if;
 perform public.split_session_check(repeat('a',64),t+interval '299 seconds');
 begin perform public.split_session_check(repeat('a',64),t+interval '300 seconds');raise exception 'valid at expiry';exception when raise_exception then if sqlerrm<>'SESSION_INVALID' then raise;end if;end;
 b:=public.split_session_extend(o,r,t+interval '200 seconds',300);
 perform public.split_session_check(repeat('a',64),t+interval '499 seconds');
 b:=public.split_session_open(o,r,repeat('b',64),t+interval '10 seconds',300);
 begin perform public.split_session_check(repeat('a',64),t+interval '11 seconds');raise exception 'old token valid';exception when raise_exception then if sqlerrm<>'SESSION_INVALID' then raise;end if;end;
 begin perform public.split_session_open(x,r,repeat('d',64),t,300);raise exception 'cross-owner open';exception when raise_exception then if sqlerrm<>'RECEIPT_NOT_FOUND' then raise;end if;end;
 begin perform public.split_session_close(x,r,t);raise exception 'cross-owner close';exception when raise_exception then if sqlerrm<>'SESSION_NOT_FOUND' then raise;end if;end;
 begin perform public.split_session_status(x,r);raise exception 'cross-owner status';exception when raise_exception then if sqlerrm<>'RECEIPT_NOT_FOUND' then raise;end if;end;
 begin perform public.split_session_open(o,r,repeat('e',64),t,5);raise exception 'ttl bound';exception when raise_exception then if sqlerrm<>'SESSION_INVALID_TTL' then raise;end if;end;
 if (select count(*) from public.split_sessions where receipt_id=r and status='ACTIVE')<>1 then raise exception 'one active';end if;
 perform public.split_session_close(o,r,t+interval '20 seconds');
 begin perform public.split_session_check(repeat('b',64),t+interval '21 seconds');raise exception 'closed token valid';exception when raise_exception then if sqlerrm<>'SESSION_INVALID' then raise;end if;end;
 perform public.split_session_open(o,r,repeat('f',64),t,300);
end $$;
set local role authenticated;
select set_config('request.jwt.claim.sub','00000000-0000-4000-8000-000000000092',true);
DO $$ begin
 if (select count(*) from public.split_sessions)<>0 then raise exception 'split RLS leak';end if;
 begin perform token_hash from public.split_sessions;raise exception 'token_hash readable';exception when insufficient_privilege then null;end;
 begin insert into public.split_sessions(receipt_id,token_hash,expires_at) values(current_setting('test.r1')::uuid,repeat('9',64),now()+interval '1 minute');raise exception 'browser insert';exception when insufficient_privilege then null;end;
 begin perform public.split_session_check(repeat('f',64),now());raise exception 'browser function';exception when insufficient_privilege then null;end;
end $$;
select set_config('request.jwt.claim.sub','00000000-0000-4000-8000-000000000091',true);
DO $$ begin
 if (select count(*) from public.split_sessions where status='ACTIVE')<>1 then raise exception 'owner read';end if;
end $$;
rollback;
