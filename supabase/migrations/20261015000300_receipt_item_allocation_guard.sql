-- Keep whole-line allocation invariant on the item side as well as assignment
-- writes. Parent-item row locking serializes a price edit with an allocation.
create function public.receipt_item_allocation_guard() returns trigger
language plpgsql security invoker set search_path='' as $$
declare allocated bigint;assignment_count integer;
begin
 select coalesce(sum(amount_minor),0),count(*) into allocated,assignment_count
 from public.allocations where receipt_item_id=new.id;
 if assignment_count>0 and allocated<>new.total_price_minor then
 raise exception 'RECEIPT_INVALID_ALLOCATION' using errcode='23514';end if;
 return new;
end $$;
create trigger receipt_item_allocation_guard before update of total_price_minor on public.receipt_items for each row execute function public.receipt_item_allocation_guard();
revoke all on function public.receipt_item_allocation_guard() from public,anon,authenticated;
grant execute on function public.receipt_item_allocation_guard() to service_role;
