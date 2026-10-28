# Receipt list contract - VP-016

`GET /api/v1/receipts/summary?limit=1..20&cursor=` (Foundation Bearer auth; owner from verified context only; same headers/errors/limits as RECEIPT_DRAFTS.md). Keyset-paginated, newest first, max 20 rows per page.

Response: `{receipts:[{id,merchant,date,status,currency,total_minor,amount_kind,shares:[{participant_id,name,total_minor}],shares_total_minor,created_at}],next_cursor,limit,outstanding_minor,page_draft_total_minor}`. All money is integer minor units computed by the backend; the browser only formats.

- `amount_kind`: `DRAFT_PREVIEW` (saved VP-008 split preview, never owed money), `NONE` (no preview). `FINALIZED` is reserved for the future receivables task and is never emitted yet.
- `outstanding_minor`: finalized receivables only; always 0 until a finalization task exists.
- `page_draft_total_minor`: sum of draft preview shares on this page, informational.
- `next_cursor`: opaque, empty on the last page; invalid cursor or limit returns 400. The existing `GET /api/v1/receipts` recent-drafts selector is unchanged.
