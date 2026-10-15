DO $$ declare after_value text;begin
 select md5(jsonb_build_object(
 'receipts',(select jsonb_agg(to_jsonb(r) order by id) from public.receipts r),
 'items',(select jsonb_agg(to_jsonb(i) order by id) from public.receipt_items i),
 'events',(select jsonb_agg(to_jsonb(e) order by id) from public.receipt_events e))::text) into after_value;
 if after_value<>(select value from public.receipt_noop_fingerprint) then raise exception 'Already-normalized state changed';end if;
 if exists(select from public.receipt_legacy_review_audit) then raise exception 'Already-normalized audit fabricated';end if;
end $$;
