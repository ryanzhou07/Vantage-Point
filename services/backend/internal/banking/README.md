# Banking module

Provider: Plaid (ADR-011). Trial plan caps Production at 10 Items and `/item/remove` does not restore the cap, so tests and CI use Sandbox only; never create Production Items in tests.

## Secrets setup (ADR-009)

`BANKING_SECRET_SOURCE=local|aws`; anything else fails startup.

Local: files `plaid-client-id`, `plaid-secret` in `~/.config/vantage-point/banking/` (override `BANKING_SECRET_DIR`), dir mode 0700, files 0600. `TOKEN_ENCRYPTION_KEY` = base64 of 32 random bytes (`openssl rand -base64 32`). Never put these in the repo.

AWS: `BANKING_SECRET_ENV=<env>`; reads `vantage-point/<env>/{plaid-client-id,plaid-secret,token-encryption-key}` from Secrets Manager via the default credential chain (IAM role) and `AWS_REGION`. IAM: `secretsmanager:GetSecretValue` on `vantage-point/<env>/*` only.

`PLAID_ENV=sandbox|production` (default `sandbox`; anything else fails startup). Use the Sandbox secret with `sandbox`, the Production secret with `production`.

Token blobs: `key_version(1) || nonce(12) || AES-256-GCM(ciphertext+tag)`, AAD = user ID + connection ID.

## Connection API (VP-018c)

See `agents/contracts/BANKING_CONNECTIONS.md`. Needs `SUPABASE_URL` and `SUPABASE_SERVICE_ROLE_KEY` besides the secrets above; any missing piece disables only Banking, never auth. Provider calls use `github.com/plaid/plaid-go/v49` v49.0.0 (exact pin) behind the `Provider` interface.
