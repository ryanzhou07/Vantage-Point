# Banking module

## Secrets setup (ADR-009)

`TELLER_SECRET_SOURCE=local|aws`; anything else fails startup.

Local: files `teller-cert`, `teller-key`, `teller-app-id` in `~/.config/vantage-point/teller/` (override `TELLER_SECRET_DIR`), dir mode 0700, files 0600. `TOKEN_ENCRYPTION_KEY` = base64 of 32 random bytes (`openssl rand -base64 32`). Never put these in the repo.

AWS: `TELLER_SECRET_ENV=<env>`; reads `vantage-point/<env>/{teller-cert,teller-key,teller-app-id,token-encryption-key}` from Secrets Manager via the default credential chain (IAM role) and `AWS_REGION`. IAM: `secretsmanager:GetSecretValue` on `vantage-point/<env>/*` only.

Token blobs: `key_version(1) || nonce(12) || AES-256-GCM(ciphertext+tag)`, AAD = user ID + connection ID.

## Teller probe (VP-009c, removable)

`GET /api/v1/banking/probe` exists only when `TELLER_PROBE_ENABLED=true`, `TELLER_ENV=sandbox`, `APP_ENV` is set and not `production`, and `TELLER_PROBE_TOKEN` (a Sandbox token) is set. A violation stops startup; flag off means no route. Removal: delete `probe.go`, `probe_test.go`, and the `banking.MustRegisterProbe` line plus the `banking` import in `cmd/api/main.go`.
