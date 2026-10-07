# Vantage-Point Foundation Requirements

## 1. Purpose

The Foundation domain establishes the shared identity, authentication, and user-profile behavior used across Vantage-Point.

Banking, Receipts, and Dashboard functionality must rely on this shared identity rather than creating separate user systems.

---

## 2. Authentication Provider

Vantage-Point will use **Supabase Auth** as the authentication provider for MVP 1.

Supabase Auth is responsible for:

- User authentication
- Authentication sessions
- Google OAuth integration
- Issuing authentication tokens

Vantage-Point will not implement its own authentication system.

---

## 3. Supported Sign-In Method

For MVP 1, Vantage-Point will support:

- **Google OAuth only**

Email/password authentication is disabled for MVP 1.

A user must successfully authenticate through a valid Google account before becoming an authenticated Vantage-Point user.

No unauthenticated user may access protected Vantage-Point functionality.

---

## 4. Canonical Vantage-Point User

Every authenticated person has one canonical Vantage-Point user identity.

Conceptually:

```text
Google Account
      ↓
Supabase Auth
      ↓
Vantage-Point User
      ↓
Application Profile
      ↓
Banking Data
Receipt Data
Dashboard Data
```

The Supabase Auth user ID should serve as the stable identity reference used by Vantage-Point application data unless a later approved architecture decision introduces a different internal identifier.

---

## 5. User Profile

Vantage-Point should maintain an application-level profile associated with the authenticated identity.

Initial profile information may include:

- User ID
- Display name
- Email
- Profile image URL where available
- Created timestamp
- Updated timestamp

The profile should not duplicate Google or Supabase authentication secrets.

---

## 6. New User Creation

When a person successfully authenticates with Google for the first time:

1. Google authenticates the user.
2. Supabase Auth establishes the authenticated identity.
3. Vantage-Point ensures an application profile exists.
4. The user is allowed into the authenticated application.

Profile creation must be safe to retry.

Repeated authentication must not create duplicate profiles.

---

## 7. Authenticated Requests

Protected backend requests must establish the authenticated user from a valid authentication token.

Conceptually:

```text
Browser
   ↓
Google authentication
   ↓
Supabase Auth session/token
   ↓
Vantage-Point backend
   ↓
Token validation
   ↓
Authenticated user ID
   ↓
Domain authorization
```

The backend must not trust a frontend-supplied `user_id` as proof of identity.

---

## 8. Backend User Context

After authentication succeeds, backend code should have access to a standard authenticated-user context.

Conceptually:

```text
ctx.user_id
```

Banking and Receipt functionality should use this authenticated context rather than accepting ownership identity directly from the client.

---

## 9. Authorization Foundation

Foundation owns shared authentication and authorization infrastructure.

Foundation should provide mechanisms that allow domain code to determine the authenticated user.

Domain-specific access rules remain owned by the relevant domain.

Examples:

Foundation determines:

```text
Who is making the request?
```

Receipt determines:

```text
Does this user own this receipt?
```

Banking determines:

```text
Does this financial account belong to this user?
```

---

## 10. Protected Application Areas

Authenticated Vantage-Point functionality should require a valid authenticated session.

Examples include:

- Main dashboard
- Connected financial institutions
- Financial accounts
- Transactions
- Receipt ownership
- Receipt history
- Receivables

Unauthenticated users should not access another user's private Vantage-Point data.

---

## 11. Receipt Participants

Temporary receipt participants are not required to have Vantage-Point accounts.

Participant access is handled separately by the Receipt domain through temporary receipt-session authorization.

Therefore:

```text
Authenticated Vantage-Point User
        ≠
Temporary Receipt Participant
```

The Foundation domain must not create permanent users merely because someone participates in a receipt split.

---

## 12. Session Behavior

Authenticated users should remain signed in according to normal secure Supabase session behavior.

The application should support:

- Existing session restoration
- Expired-session handling
- Sign out
- Authentication failure handling

Protected requests with invalid or expired authentication must fail rather than silently falling back to unauthenticated access.

---

## 13. Sign Out

Users must be able to sign out.

Signing out should end the active application authentication session on that client.

After sign-out, protected pages and protected backend functionality should no longer be available without authenticating again through Google.

---

## 14. Account Deletion

Full account deletion is **not required for the first MVP implementation unless necessary for launch requirements**.

The architecture should avoid making future account deletion unnecessarily difficult.

Deletion policy, financial-data retention, and provider disconnection behavior should be defined separately before account-deletion functionality is implemented.

---

## 15. Authentication Secrets

Vantage-Point must not store:

- Google passwords
- Raw Google OAuth secrets in application tables
- Raw authentication secrets that Supabase Auth is responsible for managing

Application secrets such as Supabase service credentials must remain outside frontend code and source control.

---

## 16. Frontend Security

The frontend may know the identity of the signed-in user for presentation purposes.

However, frontend state is not authoritative for access control.

