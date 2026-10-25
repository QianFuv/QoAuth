# Lightweight product scope

Review revision: 2, 2026-10-25. This replaces the earlier enterprise-oriented
proposal following the user's explicit scope correction. Nothing is implemented.

## Intended result

QoAuth is a small, self-hosted identity service for a person or small team with a
few applications. One Go executable contains its web UI; one local SQLite database
stores durable state. Deliver Windows amd64 and Linux amd64 binaries and a Linux
amd64 Docker image. A normal installation needs no external database, Redis,
message broker, Node runtime, cluster or cloud account.

The goal is a complete small authentication system, not enterprise IAM parity.
An owner can manage users and applications, users can sign in with SSO, applications
can use standard OIDC, and the owner can revoke access and restore a backup.

## Responsibilities and scenarios

| Responsibility | QoAuth | Application/operator |
|---|---|---|
| Authentication | Local accounts, password/MFA verification, SSO session, OIDC identity | Correct RP callback and application session |
| OAuth authorization | Registered clients, allowed scopes/audiences, consent and token lifecycle | Enforce API scope and record-level permissions |
| Directory | A single installation's users and app assignments | No HR directory, tenant hierarchy or organization lifecycle |
| Operations | Simple logs, health, backup/restore, migrations, key rotation | Protect host, state directory, HTTPS configuration and backups |

OAuth delegates resource access; OIDC adds identity authentication. An ID Token is
not an API access token. External identity is `(issuer, subject)`, not an email
address. Sources verified 2026-10-25:
[OIDC Core](https://openid.net/specs/openid-connect-core-1_0.html),
[OAuth security BCP](https://www.rfc-editor.org/rfc/rfc9700.html).

Primary scenarios: sign into two small web apps without re-entering credentials;
disable a user or revoke a session; add one application's callback/scopes through
the UI; recover a lost factor without bypassing MFA; move the service to another
host using a verified backup. There is one trust domain and one configured issuer.

## Required for the initial usable release

| Area | Bounded commitment |
|---|---|
| Users | Owner-created users, enable/disable, password change and controlled password recovery; no public signup by default |
| Login and SSO | Local password login, one installation-wide SSO session, correct reauthentication and consent behavior |
| OIDC/OAuth | Discovery/JWKS, code + S256 PKCE, signed ID Tokens, UserInfo, scoped opaque access tokens, introspection, revocation, refresh rotation |
| Clients | Public vs confidential type, exact registered callback/logout URIs, allowed audiences/scopes and user app assignment; owner-managed registration |
| Permissions | Two administrative roles: administrator and regular user; per-app assignments/scopes, server-side enforcement; no universal policy language |
| Sessions | Session list and revoke; logout current/all; user disable blocks new login/refresh/online access validation; RP-initiated logout |
| MFA/recovery | TOTP and single-use recovery codes; administrators must enroll MFA; users may enroll or be required by policy; reset does not remove MFA |
| UI/API | Small Users, Applications, Sessions, Activity and Settings screens, plus self-service password/MFA/session pages; minimal documented admin API used by UI |
| Logs | Redacted structured operational logs and simple security activity records with actor/action/target/time/result; bounded retention |
| Delivery | Embedded assets, SQLite data directory, native OS smoke tests, non-root Docker image, explicit migration/backup/restore instructions |
| Integration | One Go web RP example and generic OIDC/BFF instructions; happy-path and negative tests |

“Logout all” means all QoAuth sessions/grants for the selected user. Other
applications' own cookies remain their responsibility; this release must not
promise instantaneous logout of every RP. See [architecture](architecture.md).

## Can be added later, only when useful

Passkeys/WebAuthn; a small number of upstream OIDC/social providers; client
credentials for a concrete machine-client need; back-channel logout for participating
apps; an independent browser-client example; device flow for CLI; better audit
export. Each is optional, with its own implementation and security tests. The
initial release remains complete without them. No speculative framework is needed
to make these possible.

## Explicitly outside the current commitment

Complex organizations, B2B tenancy, multi-tenant isolation as a product, approval
workflows, SCIM, LDAP/AD, SAML enterprise federation, generic ABAC/ReBAC, custom
plugin execution, FAPI certification, Redis, PostgreSQL as a prerequisite,
microservices, message queues, Kubernetes, HA and multi-region deployment.
Enterprise competitors remain research references only. These are not later
mandatory milestones disguised as a backlog.

## Proposed defaults to verify

Authorization code: one use, 60 seconds. Access token: 5 minutes. Browser session:
8-hour idle / 24-hour absolute lifetime. Refresh family: 7-day idle / 30-day
absolute lifetime, bounded by grant and account status; ordinary interactive
logout also revokes the associated family. Administrative sensitive changes
require reauthentication within 5 minutes. These are policy proposals, not
requirements imposed by the standards.

Use a modest reference fixture of 100 users and 10 applications. Measure 10
non-password protocol requests/second for 10 minutes on a recorded 2-vCPU/2-GiB
machine, with no server errors or unbounded memory growth; record p95 rather than
invent a capacity guarantee. Benchmark password hashing independently with a
bounded concurrency budget. A larger supported scale requires actual evidence.

Current next action: review this lightweight scope and the short roadmap.
No business code, service, real credential or external binding is authorized yet.
