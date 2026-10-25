# Short roadmap and acceptance conditions

Review revision: 4, 2026-10-25. The lightweight scope still supersedes the earlier
team/B2B/enterprise roadmap. Parent A4 authorized M2.2 locally. Estimates are rough
effort for one experienced engineer, not delivery promises.

| Milestone | Effort estimate | Result |
|---|---|---|
| M0 Research and plan | Completed | Official-source comparisons and this lightweight review package; local document commits |
| M1 Small feasibility gate | 3-5 working days | Validate Go OP library + SQLite transactions + embedded UI on both targets; select a patched dependency graph |
| M2 Working core | 3-5 engineer-weeks | Local users, apps/scopes, code+PKCE, SSO/sessions/revoke, TOTP/recovery, simple admin UI/logs |
| M3 Validate and package | 2-3 engineer-weeks | Protocol/security regression evidence, Go integration example, native binaries/image smoke, backup/restore and rotation guide |

M0/M1/M2.1 were accepted. M2.2 minimal maintenance is implemented and locally
verified for parent review: consistent snapshots, isolated restricted restore,
persisted signing rotation, installation locking and bounded cleanup. The current
program remains one real Go authentication service with SQLite and embedded UI;
fixture code remains test-only. Final M2.2 evidence is 127 passing Windows race
tests/subtests across eight packages, final binary maintenance smoke, standard RP
regressions, browser client replacement, target builds and dependency scans.
[Exact commands and limitations](plan/2026-10-25-qoauth-overall.md#final-m22-evidence-2026-10-25).

One independent RP library is tested, including multiple app registrations. A second
implementation, local conformance profiles and native Linux/image runs remain open.
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

## Remaining bounded work

Product functionality is close to the agreed small-system boundary. The next
optional internal review slice should close only these concrete usability gaps:

| Item | Small implementation boundary | Acceptance / likely files |
|---|---|---|
| Owner-assisted password reset UI | Expose the existing authorized reset operation for a selected non-owner; preserve MFA and revoke prior state | Fresh admin MFA/CSRF, reset-to-old-login race and browser checks; `web/src/admin.tsx`, `internal/oidc/api.go` and focused tests |
| Disabled-client recovery visibility | Show disabled client IDs/configuration without secret verifiers and prefill the existing reviewed replacement form | No change to generation/assignment invalidation; fresh secret only on explicit replacement; `internal/store/identity.go`, `internal/oidc/api.go`, `web/src/admin.tsx` |
| Password blocklist and small operational readiness contract | Assess the existing small local list and hashing budget; define a minimal non-sensitive readiness check rather than a monitoring system | Bounded password rejection/cost tests and failure-state checks; `internal/identity`, `internal/oidc`, existing docs |

These are proposed follow-ups, not implemented by M2.2 or silently approved new
features. They need no new infrastructure. Work/tests can remain in the repository
with disposable credentials; actual accounts, deployment or host changes still need
specific authority. Use this same execution checklist for the parent's next decision.

Release gates remain finite and separate from product feature expansion:

- Native Linux amd64 and actual non-root Docker image build/run/restart, including
  file ownership, OS lock, embedded UI, restore/rotation and mount permissions. No
  usable permitted runtime was exercised here; do not start/install host components
  merely to satisfy the gate.
- A second independent standard RP and selected local OIDF profiles; pin a separate
  conformance harness and obtain any required runtime/setup authority. No public
  callback, SaaS trial or certification claim.
- Recorded small-team load/hash measurements and accessibility/keyboard review.
  Synthetic SQLite COMMIT/full-disk/locking failures already pass; this does not
  establish capacity or physical power-loss durability.
- Cross-version upgrade/rollback drill with a compatible snapshot binary, final
  NOTICE/license inventory, SBOM and reproducible artifact checksums. QoAuth's own
  source license remains a maintainer decision before distribution. No publishing
  or deployment is authorized by preparing these artifacts.

## Optional work and stop line

Passkeys, social/upstream OIDC, back-channel logout, machine clients and device
flow are demand-driven follow-ups. SCIM/LDAP/SAML, tenancy, approval flows and
cluster infrastructure are outside the current roadmap, not postponed obligations.

Use the existing overall-plan checklist for authorization history, file scopes,
verification and logical commits; no new per-stage plan ceremony. Keep main,
one logical change per commit, no branch/worktree, push, release or deployment.
