# Frontend presentation ownership (VP-004)

Foundation owns auth hooks/panels and `lib/auth/` provider adapters. App Router pages
compose features; shared UI has no provider/session/API logic. Dashboard owns its
routes/navigation/aggregation and consumes these primitives after Integration Review.

- `Button` in `apps/web/components/ui/Button.tsx`: native typed button attributes,
  `variant?: 'primary' | 'secondary'`, default type button. Native disabled semantics,
  48px minimum height and visible keyboard focus. Native handlers are caller-owned.
- `Surface`: native `HTMLAttributes<HTMLElement>`, renders a section. Callers supply
  accessible heading association; padding/layout belong to the feature.
- `Feedback`: `{ tone?: 'status' | 'error', children: ReactNode }`; status uses role
  status, error uses alert. Pass safe, user-facing text, never raw provider errors.
- `Brand`: `{ compact?: boolean }`; visual branding with decorative mark.

Global CSS tokens: `--vp-bg`, `--vp-ink`, `--vp-muted`, `--vp-accent`, `--vp-border`.
Use `vp-button vp-button--secondary` for equivalent native links; don't make navigation
buttons or place buttons inside links. Feature classes use `auth-` to avoid collisions.
These primitives cover actual auth/callback reuse; add abstractions when needed.

`useAuthSession()` returns `profile: Profile | null`, `busy: boolean`, `error: string`,
`signIn(): Promise<void>`, `signOut(): Promise<void>`, `retry(): void`. It owns initial
load, auth-event subscription, aborts, deferred reads, action deduplication and retry.
The shared API/identity/PKCE contract in FOUNDATION_AUTH.md is unchanged.
`useAuthCallback()` handles one StrictMode-safe exchange, safe errors, fixed return
path and callback-URL cleanup. Shared components do not consume these hooks.

Run `pnpm test` for deterministic flow and actual DOM/hook interaction checks.
`@testing-library/react` and `jsdom` are dev-only testing tools; each DOM test opts in
via its Vitest environment comment. Real OAuth/browser smoke remains separately
required; DOM fixtures do not claim visual or provider verification.
