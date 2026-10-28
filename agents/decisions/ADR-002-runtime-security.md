# VP-002 runtime remediation and temporary lint dependency exception

Owner approved on 2026-10-11 by responding “Yes go ahead” to the Orchestrator's
proposed Go remediation and narrowly scoped temporary braces exception. Independent
Integration/Review must still assess the implementation; this is no self-approval.

## Go

VP-001 Go 1.24.2 baseline HTTP API had 29 reachable standard-library findings under
govulncheck 1.1.4. Latest affected ranges require Go 1.26.9, including:
https://pkg.go.dev/vuln/GO-2026-6617
https://pkg.go.dev/vuln/GO-2026-6612
https://pkg.go.dev/vuln/GO-2026-6613
The baseline calls http.ListenAndServe and HTTP/TLS parser paths. A scanner call
trace is a conservative reachability finding, not proof each route is exploitable.
Pin Go 1.26.9 in go.mod, CI and developer docs and rerun all backend/security checks.
Go release changes may affect future dependencies; the current API has no external
Go dependencies. No authentication/product architecture is changed.

### 2026-10-28 patch update

`GO-2026-6809` affects `crypto/x509` before Go 1.26.10 and was reachable through
the API's HTTP server certificate-verification path. Pin Go 1.26.10 in active
runtime configuration, CI, and developer documentation and rerun the same backend
and security checks. This is the minimum fixed patch release in the existing Go
1.26 line; it does not change application behavior or architecture. Historical
1.26.9 evidence above remains an accurate record of the earlier remediation.

## Braces (temporary accepted residual risk)

https://github.com/advisories/GHSA-vfj7-8cjw-p6xm affects braces <=3.0.3 and lists no
patched version. Installed path is exactly:
`apps__web>eslint-config-next>@next/eslint-plugin-next>fast-glob>micromatch>braces`.
This is a development lint dependency; current app routes do not accept brace
patterns or import it. Deeply nested untrusted patterns can terminate the lint
process; do not treat this as an absence of risk, especially in PR tooling.

The complete pnpm audit still runs and reports the finding and raw vulnerability
totals. Exception applies only to this advisory, braces 3.0.3, this exact lint-only
path, high severity and no-patch advisory state. Other high/critical findings fail.
Changed scope/version, an available patch, or expiry at 2026-11-11 UTC fails closed.
Remove the exception when a supported fix is published or before broader glob
input handling is introduced; assess alternatives at expiry. No global allowlist,
production runtime exclusion, or secret-scan suppression is authorized.
