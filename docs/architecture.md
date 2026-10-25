# Lightweight architecture and minimal operations

Review revision: 3, 2026-10-25. Parent A3 authorized the M2.1 core.
The lightweight scope is unchanged. Implemented behavior is distinguished below
from remaining operations/release targets; no production-readiness claim is made.

M2.1 actual modules: `cmd/qoauth` owns explicit init/recover/serve commands;
`internal/identity` handles bounded Argon2id, TOTP, encryption and private files;
`internal/oidc` owns the production OP adapter and browser/admin API; `internal/store`
owns SQLite transactions; `internal/webassets` embeds the Semi frontend in `web/src`.
`examples/go-rp` is an independent coreos/x/oauth2 client. `internal/fixture` remains
test-only. No config/session/service framework was needed.

Schema migrations 001-004 are ordered and checksum-checked. Tables are
`schema_migrations`, `users`, `clients`, `assignments`, `auth_requests`, `grants`,
`tokens`, `recovery_codes`, `activity`, `accounts`, `installation`, `browser_sessions`,
`client_config`, `consents`, and `login_requests`. There are no tenant/organization
tables. A password authentication event survives MFA cookie rotation and makes
same-second `max_age=0` reauthentication explicit. Unauthenticated transactions
bind to a browser first, then atomically to a user/session/consent before code issuance.
Automatic SSO completion rechecks existing consent in that same transaction and
cannot recreate withdrawn consent. Protocol duplicates are rejected before decoding;
the internal callback is GET-only and browser-bound.

## Decisions

| ID | Recommendation | Reason / condition |
|---|---|---|
| ADR-01 | One Go process with embedded static UI | Simple binary deployment; no microservices, Node runtime or plugin framework |
| ADR-02 | One local SQLite database, modernc driver first | No external DB service; validate atomic token operations and both native targets |
| ADR-03 | One installation, one configured HTTPS issuer | Users and app clients only; no organization/tenant product or row-level-security machinery |
| ADR-04 | Evaluate zitadel/oidc OP library with patched dependencies | Reuse protocol machinery; application still owns identity, sessions, recovery and transactions |
| ADR-05 | Opaque access tokens + authenticated online introspection | Clear central revocation; online dependency and RP-session limitations are explicit |
| ADR-06 | React + TypeScript + Vite + Semi Design, npm lockfile | Aligns with inspected Exta browser UI; test CSP/accessibility and bundle compatibility |
| ADR-07 | Simple SQL activity records and structured logs | No event bus, distributed cache, audit pipeline or message queue |

## Internal boundaries, without a service framework

HTTP/OIDC handlers call small user, client, credential and session modules backed
by one SQLite store. Admin endpoints call the same authorization and mutation
functions. Static assets handle presentation, never authority. The actual module
list above is sufficient; do not add packages only to match an earlier directory sketch.

Store users, password verifiers, TOTP credentials, recovery codes, client settings,
app assignments, consents, browser sessions, short-lived login transactions/codes,
access-token records, refresh families, key versions and security activity.
Use foreign keys and explicit transactions. No generic tenant abstraction or
enterprise role/policy engine. Admin and regular-user checks must occur server-side.

## Protocol and client isolation

Use a stable configured issuer such as `https://auth.example.test`. Discovery
and token `iss` must match it exactly; do not infer it from untrusted Host or proxy
headers. Subjects are immutable random user identifiers, not email addresses.
Code+S256 is required for every interactive client as a product rule. Public
clients have no secret; confidential clients also authenticate at the token endpoint.
Manually register exact callback and logout URIs. Restrict client scopes, audience,
grant type and user assignment. No password grant, implicit flow, anonymous dynamic
registration or wildcard redirects.

Signed ID Tokens initially use a deliberately selected interoperable algorithm
RS256 with an explicit allowlist. ID Tokens remain standard signed JWTs even when
access tokens are opaque. Verify issuer, audience/authorized party,
nonce, expiry and token purpose. Honor prompt/max_age/auth_time based on actual
authentication. ID Tokens never authorize resource APIs. Sources:
[OIDC Core](https://openid.net/specs/openid-connect-core-1_0.html),
[Discovery](https://openid.net/specs/openid-connect-discovery-1_0.html),
[RFC 9700](https://www.rfc-editor.org/rfc/rfc9700.html), verified 2026-10-25.

Release only allowed claims. An owner-entered email address is not proof of email
ownership: omit email_verified or set it false unless a real verification procedure
has succeeded. With optional SMTP disabled, do not pretend an email was verified.

## SQLite correctness and limits

Single application instance, local persistent disk, short transactions and bounded
connections. Configure and verify foreign keys and busy timeout on every connection.
Evaluate WAL with synchronous=FULL for acknowledged security-state durability;
the actual filesystem/hardware still matters. Require an embedded SQLite version
containing the WAL-reset correction (3.51.3 or applicable backport), and verify it
at runtime rather than assuming the Go module version proves it. WAL has one writer
and is not a network-filesystem/cluster strategy.
[SQLite WAL](https://www.sqlite.org/wal.html),
[modernc driver](https://pkg.go.dev/modernc.org/sqlite), verified 2026-10-25.

Consume a code/recovery token conditionally and create resulting state in one
transaction. Refresh rotation rechecks client/grant/account status, consumes the old
token and creates its successor atomically. Retain consumed-token lineage. Detected
reuse must commit family revocation even though the HTTP result is an OAuth error.
A timeout after commit can require reauthentication; do not promise retry success
or introduce an untested replay grace window. Return no usable token before commit.

High-entropy bearer handles use stored verifiers when supported by the OP contract.
Encoded opaque tokens must have authenticated integrity and revocation checks.
Password verifiers use salted Argon2id. TOTP seeds use standard AES-GCM with
user-specific authenticated context. The implemented private `keys.json` contains
RSA PKCS#8 and the AES root secret; RSA is not separately encrypted. This replaces
the earlier nested-encryption sketch: a co-located wrapping key adds no host-owner
isolation. Protect/back up this sensitive key file separately from the database.
Windows validates DACLs and current user/SYSTEM/Administrators ownership; Unix checks
owner/mode. Production startup also validates DB/WAL/SHM/journal file types and
permissions. It never repairs unsafe existing permissions or regenerates lost keys.
Temporary synthetic tests cover file corruption/context swaps and a separate-process
restart. Actual Windows executable restart preserved JWKS; native Linux remains open.

Admission caps: 4096 anonymous sessions, 2048 login transactions globally, 16 per
browser. Each creation prunes at most 64 expired records of each kind; cleanup
commits even if admission is denied. Keep valid token/replay lineage out of this
pruning path. Authenticated-session/token/activity retention remains a release gate.

## SSO, logout and revoke

| Operation | Guaranteed target behavior | Explicit boundary |
|---|---|---|
| Local app logout | App clears its own session | Does not necessarily end QoAuth SSO |
| RP-initiated logout | Validate context/return URI; end intended QoAuth session and its grants | Cannot delete another site's cookies |
| Revoke session / all QoAuth sessions | Disable selected sessions and related access/refresh grants | “All” does not mean every independent RP session |
| Disable user | Deny new login, refresh and online token validation after commit | Existing app-local authorization decisions need app cooperation |
| Introspection | Authenticated resource checks current state, audience and scopes; fail closed | No positive-result cache initially; outage means protected API denial |
| Back-channel logout | Optional later feature for participating clients | No initial promise of delivery or universal sign-out |

| Security change | Objects invalidated atomically |
|---|---|
| Disable user / complete password recovery | User sessions, grants, access/refresh tokens and unredeemed authorization codes |
| Remove assignment / withdraw consent | Matching user/client grants, access/refresh tokens and pending codes |
| Disable client / revoke its authentication credential | Affected client grants and pending codes; old credential cannot authenticate |

Authorization, code exchange, refresh, UserInfo and introspection recheck applicable
current user/client/grant/assignment state. Recovery preserves MFA and does not create
a fully authenticated session. Independent RP cookies remain the RP's responsibility.

The existing Applications configuration carries a small resource mapping: an RP has
allowed resource audiences; a confidential resource identity has its own credential
and allowed introspection audiences. Example: `web-rp` requests `notes-api`, while
the resource client `notes-api` may introspect only its own `notes-api` audience. Use registered
`client_secret_basic`; do not equate the RP with the resource, or allow every
authenticated caller to inspect every token. No resource service/policy framework.
M2.1 stores resource credentials as verifiers and returns a new secret once.
Validation created temporary synthetic registrations only. `resource` request
parameters are rejected; configured audiences cannot expand on refresh.

If an app keeps a session without fresh token validation, it must define its own
expiry/revalidation policy. The example demonstrates this limitation. Use a
Secure/HttpOnly/SameSite cookie and CSRF checks for browser state; never store
session identifiers or refresh tokens in localStorage. Sources:
[RP-initiated logout](https://openid.net/specs/openid-connect-rpinitiated-1_0.html),
[revocation](https://www.rfc-editor.org/rfc/rfc7009.html),
[introspection](https://www.rfc-editor.org/rfc/rfc7662.html), verified 2026-10-25.

## Packaging and configuration

Build static assets once, embed with Go `embed`, and serve only known UI routes.
Do not send SPA HTML for unknown API/protocol paths. No runtime CDN/fonts/Node
dependency. Strict CSP, no framing, escaped profile content, no caching of sensitive
login responses; fingerprint public static assets.

Artifacts: `qoauth-windows-amd64.exe`, `qoauth-linux-amd64`, Linux amd64 OCI image.
Aim for CGo-free builds. Native Windows/Linux and container smoke tests are still
required; cross-compilation alone is insufficient. Use a pinned minimal non-root
image, CA roots, read-only root filesystem and one explicitly mounted writable
state directory. Persist DB, configuration and protected keys outside the image.

Native HTTP listening defaults to loopback. Use the operator's existing trusted
HTTPS reverse proxy and a configured canonical HTTPS issuer; forwarded headers
cannot select the issuer. An explicitly configured `--trusted-proxy <IP>` may supply
exactly one sanitized X-Real-IP; other peers' headers are ignored. Built-in TLS
management is not required. Containers
listen on 0.0.0.0 internally, with host publication defaulting to 127.0.0.1;
container loopback is not reachable through ordinary port mapping. Actual proxy,
TLS and network changes require separate authority. SMTP is optional:
without it use a reviewed owner-assisted recovery path and local recovery codes,
not a fabricated successful email reset. No automatic firewall changes, service
installation, public binding, default credentials or real SMTP account creation.

## Remaining minimal backup, upgrade and key runbook (not yet implemented)

1. Use SQLite backup API or validated VACUUM INTO snapshot, or a documented
   stopped-service backup; do not copy only a live .db while ignoring WAL.
   Back up issuer/configuration; protect and back up the sensitive key file
   separately from database snapshots. [SQLite backup guidance](https://www.sqlite.org/backup.html).
2. Restore into a separate location in maintenance mode. Check integrity, schema,
   issuer and key availability before serving. Invalidate sessions, access/refresh
   grants, outstanding login/reset/invitation tokens and saved recovery codes.
3. Reconcile user/password/MFA and client credentials/callbacks/assignments/scopes/
   signing-key state before reopening; snapshots cannot know later changes. Disable
   uncertain clients/keys and force user recovery/re-enrollment when necessary.
   Do not claim restoration preserves unseen post-backup revocations.
4. Apply ordered checksum-verified schema migrations after a pre-upgrade snapshot.
   On failure, stop safely; restore a compatible snapshot rather than blindly run
   destructive down-migrations. State version compatibility in the upgrade guide.
5. Planned signing rotation publishes the next public key, switches signing, then
   retains old verification keys beyond relevant token lifetimes and cache/skew
   allowance. Emergency compromise stops issuance, revokes affected grants and
   communicates the need for clients to refresh trust. Test restart during rotation.
6. Log redacted errors and security actions; keep a bounded local activity history
   (proposed 30 days). Record security mutations in the same transaction as their
   activity row. This is useful history, not tamper-proof evidence against the host
   owner. Health checks distinguish process liveness from usable durable state.

No HA, remote database, scheduled backup daemon or external monitoring service is
created by this design. The user/operator can later choose scheduling and destinations.

M2.1 first-owner setup is explicit local `init`, with a durable database marker,
private keys and redirected password input. No anonymous claim endpoint exists.
A repeated init or missing database beside old keys fails closed. Interrupted owner
enrollment resumes after password login with enrollment-only authority. Host-only
`recover-owner --confirm` resets the owner to restricted MFA enrollment, revokes
existing authentication and records activity. Only temporary synthetic installations
were initialized during this task.

## Local packaging reference

Read-only inspection on 2026-10-25: Exta web/package.json, embedded asset loader,
release builder, Dockerfile/compose; Yousage frontend/package.json and embed/build
files. Exta demonstrates React/TypeScript/Vite/Semi, embedded assets, CGo-free
Windows/Linux amd64 targets and a non-root image. Yousage adds desktop/mobile
requirements irrelevant here. No neighboring file was changed, copied into QoAuth,
built or tested. [Go embed](https://pkg.go.dev/embed) confirms the standard mechanism.
