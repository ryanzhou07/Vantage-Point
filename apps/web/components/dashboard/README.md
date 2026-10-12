# Dashboard presentation (VP-005)

`app/dashboard/layout.tsx` composes `DashboardShell`; the overview and destination pages compose their respective presentation components. `navigation.ts` owns the typed MVP route inventory. Unknown sections return Next.js not-found.

- `DashboardShell({ children: ReactNode })` owns the mounted shell's session display and mobile navigation. It consumes Foundation `authClient`, `loadProfile` and `signOut` unchanged. It never stores tokens or caches profiles. Auth events immediately clear prior private content; aborted/stale loads cannot republish it. The keyed content subtree resets per profile identity.
- `DashboardNavigation()` provides standard links, a current-page marker, keyboard focus styles and grouped destinations.
- `DashboardOverview()` links to availability explanations. It displays no balances, counts or fabricated financial data.
- `FeatureUnavailable({ destination: DashboardDestination })` states the feature has not been implemented. Only a return link is interactive; no domain mutation/provider requests exist.
- `createDashboardSession(load, publish)` manages one shell's cancelable profile read lifecycle. A deferred callback avoids SDK operations inside auth event callbacks. Disposal cancels both queued and active work.

The frontend guard prevents private profile presentation; API authorization remains Foundation's responsibility. Static unavailable content is public-safe and contains no user/domain data. Banking/Receipt data integration and authoritative financial semantics require later approved work orders.

VP-004 approved shared presentation components must replace temporary local presentation classes before final VP-005 review. No shared design system is introduced here. Existing Foundation public auth contracts stay unchanged.

Tests currently cover stale/two-user responses, immediate private-state clearing, signed-out/error/retry, auth-event deduplication, cleanup, route inventory, active-link semantics and truthful unavailable content. Static render tests do not establish browser responsiveness or keyboard interactions. Actual DOM interaction and mobile/tablet/desktop visual verification follow coordinated VP-004 tooling integration; they remain completion gates.
