# Short roadmap and acceptance conditions

Review revision: 3, 2026-10-25. The lightweight scope still supersedes the earlier
team/B2B/enterprise roadmap. Parent A3 authorized M2.1 locally. Estimates are rough
effort for one experienced engineer, not delivery promises.

| Milestone | Effort estimate | Result |
|---|---|---|
| M0 Research and plan | Completed | Official-source comparisons and this lightweight review package; local document commits |
| M1 Small feasibility gate | 3-5 working days | Validate Go OP library + SQLite transactions + embedded UI on both targets; select a patched dependency graph |
| M2 Working core | 3-5 engineer-weeks | Local users, apps/scopes, code+PKCE, SSO/sessions/revoke, TOTP/recovery, simple admin UI/logs |
| M3 Validate and package | 2-3 engineer-weeks | Protocol/security regression evidence, Go integration example, native binaries/image smoke, backup/restore and rotation guide |

M0/M1 were accepted; M2.1 core implementation and available local verification are
complete for parent review. The current executable is a real authentication service,
distinct from test-only fixture code. Seventy-nine race tests/subtests, standard RP
HTTP/browser flows, native Windows restart, source/final-binary scans, target builds
and Compose validation have evidence. Linux native/image execution remains open.
See the [single checklist](plan/2026-10-25-qoauth-overall.md#final-m21-evidence-2026-10-25).

M2.1 covers one independent RP library in two app registrations; a second RP library
and local conformance suite remain release checks. The next bounded proposal is
M2.2 minimal maintenance/restore/rotation, described with files, tests and authority
in the same checklist. No enterprise expansion or infrastructure dependency is implied.
All estimates above are planning ranges, not completion or production promises.

## M0 document acceptance

| ID | Condition | Check |
|---|---|---|
| D1 | All ten requested candidates covered with dates, primary sources and edition/license distinctions | Read matrix and source ledger; validate consequential findings |
| D2 | Required / optional / excluded scope is consistent across every current design document | Scope and cross-document review |
| D3 | UTF-8 documents, working local links, no unresolved template/conflict markers | Local document scan and `git diff --check` |
| D4 | Docs-only work on main; logical local commits; no remote, residual changes or business source | `git log`, `git ls-files`, `git remote -v`, `git status --porcelain=v1` |

## M1: feasibility, not a new product framework

| ID | Observable acceptance |
|---|---|
| F1 | Selected OP library runs code+S256, discovery, ID Token/UserInfo, introspection/revocation and RP-initiated logout in a synthetic harness; metadata advertises only supported features |
| F2 | SQLite code/refresh/recovery consumption remains atomic under 20 concurrent attempts and injected commit/process failures; no duplicate usable grants; busy handling is bounded |
| F3 | Same static UI bundle embeds into Windows/Linux amd64 executables without runtime Node/assets directory; API 404s never return SPA HTML; simple UI passes intended CSP |
| F4 | Exact Go/frontend/SQLite-engine versions, licenses and advisory remediation are recorded; actual dependency graph builds; native smoke environments are identified without unauthorized installs |

F1-F3 share one actual program, OP handler and SQLite adapter: synthetic user plus
Go RP, embedded login page, code/PKCE, signed ID Token/UserInfo, protected request,
refresh and revoke rejection, then persistence across restart. Failure injection
must reach this real storage path; independent fake-protocol/SQL/UI demos cannot
satisfy M1. Full account CRUD, production enrollment and admin UI remain M2.

Do not expand into PostgreSQL or a storage abstraction if SQLite fails a test.
Diagnose the concrete issue, then present the smallest design change if needed.

## M2: core behavior

| ID | Observable acceptance |
|---|---|
| A1 | User create/disable/password change works; unknown/disabled accounts fail; password policy, hashing, throttling and enumeration-resistance tests pass |
| A2 | TOTP needs enrollment proof and fresh authentication; OTP/recovery-code reuse fails; password reset cannot disable MFA; last administrator cannot be removed |
| A3 | Two independent RP libraries pass code+PKCE; wrong/missing PKCE, wrong client/redirect, replayed code, bad nonce/issuer/audience/signature fail |
| A4 | Application scopes/audiences/assignments are enforced server-side; consent cannot grant excess access; public clients need no embedded secret |
| A5 | Two applications reuse eligible SSO; prompt=login, prompt=none and max_age behave correctly; app sessions are distinguished from QoAuth sessions |
| A6 | Session revoke, user disable and logout reject subsequent refresh/introspection after the transaction commits; RP-initiated logout checks context and exact return URI |
| A7 | Concurrent refresh reuse yields at most one new token and then revokes the family on detected replay; clients serialize refresh and reauthenticate after ambiguity |
| A8 | Admin/self-service endpoints reject unauthorized access, CSRF and untrusted content; activity logs record security changes without secrets; accessible keyboard flows work |

## M3: release candidate evidence

| ID | Observable acceptance |
|---|---|
| R1 | Selected local OpenID Basic OP/Config OP and supported logout tests pass at recorded suite version; this is not formal certification |
| R2 | All required threat regressions pass; reachable critical/high advisory findings resolved; focused independent authentication/recovery review completed |
| R3 | Native Windows/Linux and Linux amd64 Docker smoke pass: embedded UI, login, persistence/restart, non-root container, graceful shutdown, correct permissions |
| R4 | Consistent backup restores in maintenance mode; old grants stay invalid; reconcile revoked client credentials/callbacks and signing-key state; rotation works before/after restart |
| R5 | Go RP example and generic guide pass login, expiry, wrong token, revoke, and logout; app-session limits are explicit |
| R6 | Reference small-team load and password-cost measurements recorded; disk-full/DB-locked failures are safe; no unsupported capacity promise |
| R7 | Versioned migrations, checksum/license inventory and SBOM, minimal upgrade/rollback/backup guides are delivered; publication/deployment remain separate authorization |

A real Linux runtime or container environment may be needed later. Cross-compiling
alone is not a native smoke test. If unavailable, record the exact missing check
and environment; do not install system components or claim a pass.

## Optional work and stop line

Passkeys, social/upstream OIDC, back-channel logout, machine clients and device
flow are demand-driven follow-ups. SCIM/LDAP/SAML, tenancy, approval flows and
cluster infrastructure are outside the current roadmap, not postponed obligations.

Use the existing overall-plan checklist for authorized M1 files, dependency/download
boundary, tests and logical commits; no new per-stage plan ceremony. Keep main,
one logical change per commit, no branch/worktree, push, release or deployment.
