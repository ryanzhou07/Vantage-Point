DO $$ declare document jsonb;rid uuid;begin
 rid:='00000000-0000-4000-8000-000000000089';
 document:=public.receipt_document('00000000-0000-4000-8000-000000000088',rid);
 if document->>'version'<>'6' or document->'draft'->>'merchant'<>'Owner corrected cafe' then raise exception 'Legacy draft/version lost';end if;
 if (select count(*) from public.receipt_items where receipt_id=rid)<>2 or(select count(*) from public.receipt_participants where receipt_id=rid)<>2 or(select count(*) from public.allocations where receipt_id=rid)<>2 or(select count(*) from public.receipt_draft_shares where receipt_id=rid)<>2 then raise exception 'Legacy relational rows missing';end if;
 if (select sum(total_minor) from public.receipt_draft_shares where receipt_id=rid)<>302 or(select s.total_minor from public.receipt_draft_shares s join public.receipt_participants p on p.id=s.participant_id where s.receipt_id=rid and p.name='Student A')<>101 then raise exception 'Legacy exact cents changed';end if;
 if not exists(select from public.receipt_events where receipt_id=rid and event_type='NORMALIZED_DRAFT_MIGRATED' and old_value->'draft'->>'merchant'='Owner corrected cafe' and old_value->'preview'->>'total_minor'='302') then raise exception 'Legacy audit missing';end if;
 rid:='00000000-0000-4000-8000-000000000090';
 document:=public.receipt_document('00000000-0000-4000-8000-000000000088',rid);
 if document->'draft'->>'date'<>'' or document->'error_code'<>'"LEGACY_REVIEW_REQUIRED"'::jsonb or document->'preview'<>'null'::jsonb then raise exception 'Free-text date guessed or verified';end if;
 if (select total_minor from public.receipts where id=rid)<>280 or(select sum(total_price_minor) from public.receipt_items where receipt_id=rid)<>250 then raise exception 'Estimated-date financial corrections lost';end if;
 if not exists(select from public.receipt_legacy_review_audit where receipt_id=rid and old_value->'draft'->>'date'='Oct 14 (estimated)' and old_value->'preview'->>'total_minor'='280') then raise exception 'Raw free-text date/exact historical preview lost';end if;
 -- Explicit owner recalculation resolves the review flag; no automatic calculation.
 perform public.receipt_save('00000000-0000-4000-8000-000000000088',rid,6,document->'draft',jsonb_build_object('draft',true,'total_minor',280,'shares',jsonb_build_array(jsonb_build_object('participant_id',document->'draft'->'participants'->0->>'id','items_minor',250,'tax_minor',10,'tip_minor',20,'total_minor',280))));
 if (select error_code from public.receipts where id=rid) is not null then raise exception 'Resolved legacy review flag remained';end if;
 rid:='00000000-0000-4000-8000-000000000091';
 document:=public.receipt_document('00000000-0000-4000-8000-000000000088',rid);
 if document->'draft'->>'date'<>'' or document->'draft'->>'tax_method'<>'' or document->'draft'->>'tip_method'<>'' then raise exception 'Invalid legacy date/method guessed';end if;
 if (select count(*) from public.receipt_items where receipt_id=rid)<>2 or(select count(*) from public.receipt_participants where receipt_id=rid)<>3 or exists(select from public.allocations where receipt_id=rid) then raise exception 'Ambiguous IDs/assignments not retained for owner review';end if;
 if (select total_minor from public.receipts where id=rid)<>999 or(select sum(total_price_minor) from public.receipt_items where receipt_id=rid)<>300 then raise exception 'Unreconciled financial draft was silently balanced';end if;
 if not exists(select from public.receipt_legacy_review_audit where receipt_id=rid and old_value->'draft'->'items'->0->>'participant_id'='duplicate' and old_value->'draft'->>'tax_method'='custom-later') then raise exception 'Original ambiguous state lost';end if;
 rid:='00000000-0000-4000-8000-000000000094';
 if (select count(*) from public.receipt_items where receipt_id=rid)<>1 or exists(select from public.receipt_participants where receipt_id=rid) or(select subtotal_minor from public.receipts where id=rid)<>0 then raise exception 'Empty-participant incomplete draft lost';end if;
 if exists(select from information_schema.columns where table_schema='public' and table_name='receipts' and column_name in ('draft','preview')) then raise exception 'JSON remains authority';end if;
 if (select data_type from information_schema.columns where table_schema='public' and table_name='receipts' and column_name='total_minor')<>'bigint' then raise exception 'Money not BIGINT';end if;
end $$;
