# ADR-008 — Initial Receipt vision provider boundary

Status: owner-approved provider decision,2026-10-15; implementation still requires independent review.

Owner initially requested Gemini2.5 Flash. The explicit synthetic access test returnedHTTP404. The owner then approved **gemini-3.5-flash-lite**, whose final production-adapter synthetic-image smoke passed. Model switches require new owner approval; no automatic model or AWS fallback.

Use a server-side Extractor interface returning normalized advisory Proposal/error types. Gemini uses the verified GenerateContent responseMimeType + typed responseSchema request contract. Model payloads never define domain/schema/UI financial truth; strict Go integer validation, owner review and exact draft calculations apply independently of provider. No API keys in browser/logs/Git.

AWS compatibility means a future Bedrock/Textract adapter can implement this interface, using the existing private Supabase image bytes. No AWS resources, S3 migration, runtime AWS adapter or paid fallback is introduced. Preserve the one-database modular Go API + bounded Receipt worker architecture.

References: https://ai.google.dev/gemini-api/docs/models/gemini-3.5-flash-lite and https://ai.google.dev/api/generate-content . Live evidence and limitations: agents/evidence/VP-008.md.
