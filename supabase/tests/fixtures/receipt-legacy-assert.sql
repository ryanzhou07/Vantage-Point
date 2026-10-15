DO $$ declare document jsonb;begin
 document:=public.receipt_document('00000000-0000-4000-8000-000000000088','00000000-0000-4000-8000-000000000089');
 if document->>'version'<>'6' or document->'draft'->>'merchant'<>'Owner corrected cafe' then raise exception 'Legacy draft/version lost';end if;
 if (select count(*) from public.receipt_items)<>2 or(select count(*) from public.receipt_participants)<>2 or(select count(*) from public.allocations)<>2 or(select count(*) from public.receipt_draft_shares)<>2 then raise exception 'Legacy relational rows missing';end if;
 if (select sum(total_minor) from public.receipt_draft_shares)<>302 or(select total_minor from public.receipt_draft_shares where participant_id=public.receipt_internal_id('00000000-0000-4000-8000-000000000089','participant','a'))<>101 then raise exception 'Legacy exact cents changed';end if;
 if not exists(select from public.receipt_events where event_type='NORMALIZED_DRAFT_MIGRATED' and old_value->'draft'->>'merchant'='Owner corrected cafe' and old_value->'preview'->>'total_minor'='302') then raise exception 'Legacy audit missing';end if;
 if exists(select from information_schema.columns where table_schema='public' and table_name='receipts' and column_name in ('draft','preview')) then raise exception 'JSON remains authority';end if;
 if (select data_type from information_schema.columns where table_schema='public' and table_name='receipts' and column_name='total_minor')<>'bigint' then raise exception 'Money not BIGINT';end if;
end $$;
