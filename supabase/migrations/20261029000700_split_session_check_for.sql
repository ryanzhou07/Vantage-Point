-- Joining polls every 15 s per participant. Return the participant's payment method from the session check
-- that already runs, instead of a third call per poll. Replaces split_participant_method from 20261029000600.
create function public.split_session_check_for(p_hash text,p_now timestamptz,p_participant uuid) returns jsonb
language sql security invoker set search_path='' as $$
 select c||jsonb_build_object('payment_method',(select p.payment_method from public.receipt_participants p where p.receipt_id=(c->>'receipt_id')::uuid and p.id=p_participant))
 from (select public.split_session_check(p_hash,p_now) as c) x
$$;
drop function public.split_participant_method(text,timestamptz,uuid);
revoke all on function public.split_session_check_for(text,timestamptz,uuid) from public,anon,authenticated;
grant execute on function public.split_session_check_for(text,timestamptz,uuid) to service_role;
