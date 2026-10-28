# ADR-009 — Banking secrets source and token encryption

Status: PROPOSED 2026-10-28; owner approval required (new AWS SDK dependency, AGENTS.md §24).

Decision: Teller client certificate/key, application id and the token-encryption key are loaded server-side through a `SecretSource` interface. Local development reads files outside the repo (`~/.config/vantage-point/teller/`, dir 0700, files 0600) and a local env key; production reads AWS Secrets Manager (`vantage-point/<env>/teller-*`) with the default IAM credential chain and a short in-memory cache. Per-user Teller access tokens are encrypted at the application level with AES-256-GCM: random 12-byte nonce, additional authenticated data = user ID + connection ID, stored key version. Encrypted blob lives in `institution_connections`.

Architecture: Banking stays a module of the single Go API and one database; no microservice. Teller is reached over mutual TLS (fail closed in Development/Production; Sandbox default). Refresh is manual (5-minute per-connection cooldown) via an idempotent `SyncConnection` reusable by a future scheduled worker. A removable, flag-guarded Sandbox probe (VP-009c) exists for verification and is deleted before production.

Consequences: adds `aws-sdk-go-v2` (Secrets Manager only). No other AWS service. Key rotation procedure is a later task.

Not verified: current Teller documentation details and certificate issuance; re-check teller.io/docs during VP-009b.
