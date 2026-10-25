# Lightweight product scope

Review revision: 5, 2026-10-25. The lightweight scope remains authoritative.
A5's bounded product closure is implemented and locally verified for parent review.
Core identity, maintenance, administrator reset, disabled-client review and readiness
remain one small runtime. Targeted protocol checks passed; full certification and
native Linux/image execution are not claimed. See the finite roadmap gates.

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
| UI/API | One simple administration page for users/apps/activity, self-service password/MFA/sessions, and local operational commands; the same server-authorized admin API |
| Logs | Redacted structured operational logs and simple security activity records (event, subject, time); bounded retention, no external audit pipeline |
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

## Implemented defaults and capacity to verify

Authorization code: one use, 60 seconds. Access and ID Tokens: five minutes.
Authenticated browser session: twelve-hour absolute lifetime; no sliding idle
extension. Each refresh token lasts seven days, with a thirty-day absolute grant
limit and current account/assignment checks. Ordinary interactive logout revokes
the associated family. Sensitive administrative changes require authentication
within five minutes. These are product policies, not standards-mandated defaults.

The actual modest fixture used 100 synthetic users/10 applications and 10
non-password requests/second for ten minutes: 6,000 successes, p95 2.855 ms. It ran
on Windows/i9-12900H/20 logical CPUs/31.7 GiB, not a constrained 2-vCPU/2-GiB host.
It measured readiness/discovery/authenticated introspection with periodic refresh;
it does not establish password-login throughput, memory stability or maximum scale.
Thirty sequential Argon2id hashes measured p50 23.264 ms/p95 26.876 ms; concurrency
is bounded to four slots. A smaller-host capacity guarantee needs actual evidence.

Current handoff: A5's implementation and available isolated verification are complete
for parent review, with no new feature stage scheduled. Native Linux/image and
licensing/real-installation gates remain explicit. Real credentials, external
bindings and deployment remain outside this task.
