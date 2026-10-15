# VP-008 — Budget receipt extraction options

Research checked 2026-10-15; shortlist, not an approved provider/architecture change. Prices and model availability must be rechecked at implementation. No student-specific API discount is assumed; a consumer/student chatbot subscription is not evidence of API credits.

| Candidate | Published cost | Practical fit / limits |
| --- | --- | --- |
| Gemini 2.5 Flash-Lite | Free tier subject to quotas; standard paid $0.10 / million input tokens (including images), $0.40 / million output tokens | Best first hosted candidate to benchmark: image to structured proposal in one call. Free-tier content may be used to improve Google products; prototype with synthetic/redacted images. Paid-tier treatment differs. |
| Mistral OCR 4.1 | $4 / 1,000 pages standard | Predictable roughly $0.004 per one-page image for OCR alone; receipt-domain JSON mapping/validation may require additional processing and cost. |
| PaddleOCR | Apache-2.0 open-source code; no hosted per-call fee when self-hosted | Local/private OCR candidate with CPU and other hardware backends. Compute, packaging and maintenance still cost effort; plain OCR requires receipt structure extraction. Test the selected OCR model/package on the actual development machine before promising latency. |

Sources:
- https://ai.google.dev/gemini-api/docs/pricing#gemini-2.5-flash-lite
- https://docs.mistral.ai/inference/pricing
- https://github.com/PaddlePaddle/PaddleOCR

Illustrative Gemini cost, not a measured receipt estimate: 2,000 total billed input tokens plus 1,000 output tokens = $0.0006 per receipt, or $0.60 / 1,000 receipts, excluding retries and extra calls. Real image token counts and output lengths must be measured. Mistral OCR alone is $4 / 1,000 single-page receipts before any structured parsing.

Recommendation: benchmark Gemini Flash-Lite first for fastest low-cost prototype; compare local PaddleOCR if private offline processing/no API spend is a priority. Keep Mistral as the specialist OCR comparison. Do not select based on OCR marketing accuracy: use synthetic receipt fixtures covering blur, rotation, quantities, discounts, tax/tip and odd cents. Record item/amount exact-match, reconciliation errors, schema failures, p50/p95 latency and measured cost. Owner review stays mandatory and backend integer calculations stay authoritative for every provider.

Implementation remains gated on owner provider approval/configuration; this research does not authorize a paid service, a new Python runtime/service in the deployed architecture or upload of real receipts to a free-tier provider.

## AWS options requested by owner
- Amazon Textract AnalyzeExpense: receipt/invoice-specific structured extraction. AWS publishes an Oregon example at $0.01/page for the first million pages ($10 / 1,000 single-page receipts); region/tier and other services can change the bill. https://aws.amazon.com/textract/pricing/ and https://docs.aws.amazon.com/textract/latest/dg/analyzing-document-expense.html
- Amazon Nova Lite on Bedrock: multimodal candidate for receipt-to-JSON, published AWS overview $0.06 / million input tokens and $0.24 / million output tokens. Check selected region/model availability and current quote before implementation. Illustrative 2,000 billed input + 1,000 output tokens = $0.00036 per receipt, not a measured receipt cost. https://docs.aws.amazon.com/bedrock/latest/userguide/model-card-amazon-nova-lite.html and https://d1.awsstatic.com/onedam/marketing-channels/website/aws/en_US/events/approved/reinvent-2025/reinvent/2024/slides/aim/AIM398-NEW_Practical-generative-AI-using-Amazon-Nova.pdf
- AWS new-customer Free Tier advertises $100 at signup and up to $100 additional earned credits, subject to eligibility/expiry/service coverage. This is not a guaranteed student discount or a statement that the owner's existing account qualifies. AWS Educate offers learning/labs, not evidence of unrestricted project API credits. https://aws.amazon.com/free/free-tier-faqs/ and https://aws.amazon.com/education/awseducate/

Add Nova Lite and Textract to the synthetic-receipt benchmark. Do not add S3 storage or replace private Supabase Storage simply to use an AWS extraction adapter; architecture changes still require approval. No AWS resources created or paid models enabled by this research.

## Owner-approved implementation choice — 2026-10-15

This historical shortlist is superseded for VP-008 by the explicit owner-approved `gemini-3.5-flash-lite` choice after2.5 access failed. See `agents/guides/RECEIPT_SETUP.md` and `agents/evidence/VP-008.md` for actual access/extraction evidence. No student discount, current per-receipt price or AWS runtime integration is inferred from the shortlist.
