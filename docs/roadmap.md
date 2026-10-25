# Short roadmap and acceptance conditions

Review revision: 5, 2026-10-25. Parent A5 approved the final bounded product and
quality closure. The lightweight scope still supersedes enterprise-oriented plans.
The estimates below are historical planning ranges, not current delivery promises.

| Milestone | Effort estimate | Result |
|---|---|---|
| M0 Research and plan | Completed | Official-source comparisons and this lightweight review package; local document commits |
| M1 Small feasibility gate | 3-5 working days | Validate Go OP library + SQLite transactions + embedded UI on both targets; select a patched dependency graph |
| M2 Working core | 3-5 engineer-weeks | Local users, apps/scopes, code+PKCE, SSO/sessions/revoke, TOTP/recovery, simple admin UI/logs |
| M3 Validate and package | 2-3 engineer-weeks | Protocol/security regression evidence, Go integration example, native binaries/image smoke, backup/restore and rotation guide |

M0/M1/M2.1/M2.2 were accepted; A5's four usability/operations changes and available
local quality checks are now complete for parent acceptance. One real Go service,
SQLite and embedded UI remain the whole runtime. Final Windows race evidence is
135 passing tests/subtests in eight packages; six opt-in/helper entries skip in the
ordinary run. Browser/load/compatible-version restore and final Windows binary
smoke ran separately. Two RP implementations pass, with the shared Zitadel OP/RP
dependency limitation recorded. Native Linux/image execution remains open.
[Exact commands and limitations](plan/2026-10-25-qoauth-overall.md#final-bounded-delivery-evidence-2026-10-25).

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
| A3 | Coreos and Zitadel RP implementations pass code+PKCE; disclose their shared OAuth2/JOSE and OP/RP dependencies; wrong/missing PKCE, wrong client/redirect, replayed code and bad token claims fail |
| A4 | Application scopes/audiences/assignments are enforced server-side; consent cannot grant excess access; public clients need no embedded secret |
| A5 | Two applications reuse eligible SSO; prompt=login, prompt=none and max_age behave correctly; app sessions are distinguished from QoAuth sessions |
| A6 | Session revoke, user disable and logout reject subsequent refresh/introspection after the transaction commits; RP-initiated logout checks context and exact return URI |
| A7 | Concurrent refresh reuse yields at most one new token and then revokes the family on detected replay; clients serialize refresh and reauthenticate after ambiguity |
| A8 | Admin/self-service endpoints reject unauthorized access, CSRF and untrusted content; activity logs record security changes without secrets; accessible keyboard flows work |

## M3: release candidate evidence

| ID | Observable acceptance |
|---|---|
| R1 | A5 targets locally executable discovery/code/PKCE/JWT/UserInfo/refresh/logout checks with pinned clients; full OIDF suite was not available and was not installed; no certification claim |
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

The four approved product changes are complete: consistent new-password policy,
MFA/CSRF-protected administrator reset, secret-free disabled-application review
and explicit replacement, and minimal readiness. No additional product functionality
is required by this closure. Existing maintenance and security semantics remain.

| Quality closure | Actual evidence / limit |
|---|---|
| Second RP | Coreos plus pinned Zitadel RP production-adapter tests; partial implementation independence, not certification |
| Local protocol checks | Discovery, code+S256, exact redirects/client/scopes, JWT claims, UserInfo, refresh/replay and confirmed logout; no installed OIDF suite |
| Modest load | 100 synthetic users/10 clients, 10 non-password requests/s for 600 seconds, 6,000 successes; p95 2.855 ms on this Windows host |
| Password cost | 30 sequential Argon2id samples, p50 23.264 ms/p95 26.876 ms; four hash slots; no capacity claim |
| Browser/accessibility | Owner MFA, reset, disabled-client review/cancel and lossless comma URLs, standard RP/logout, labels/keyboard/390px layout; not WCAG certification |
| Restore drill | Prior M2.2 executable to current code, both schema 7, real ten-minute quarantine and temporary cleanup; no arbitrary old-schema/downgrade support |
| Packaging | Two clean revisioned binaries/ZIPs plus an independently versioned image bundle; exact source ZIPs, artifact-specific SPDX/notices/checksums and separate build inventory; local only |
| Native Linux | Existing Ubuntu WSL2, uid 1000, private ext4; 130 tests/subtests passed, six skips; final binary startup/restart/SIGTERM and maintenance |
| Final image | Existing Docker 29.9.2; uid 65532, read-only rootfs, private volume, loopback TLS/coreos RP, restart/logout and maintenance; final-image binary scan passed |

Remaining release gates and limits are separate from feature expansion:

- **Platform limits:** Linux tests ran on the existing WSL2 kernel with CGO=0, not
  Linux race detection or bare-metal hardware. Image tests used small synthetic
  SQLite state, not large-database temporary-file pressure on scratch. Physical
  power-loss durability remains unverified. Native/image acceptance is now supported
  by actual runs; these limits are not missing startup permission or feature work.
- **License decision:** QoAuth has no selected source license. Artifact NOASSERTION
  is not permission to distribute. Preserve the shipped notices and review the
  recorded build-only optional-package gaps before redistributing build tools.
- **Real installation:** trusted host directories, existing HTTPS proxy and issuer,
  accounts/keys and backup custody require a concrete installation review; none were
  configured for the user. The four-entry local password blocklist is intentionally
  limited. Its adequacy must be assessed for real exposure, without claiming breach-
  corpus coverage. Full OIDF, extensive fuzzing, a broad accessibility/security audit,
  2-vCPU/2-GiB capacity and physical power-loss behavior are not established here.

No SaaS trial, public callback, new conformance infrastructure, publication or
production deployment is needed to accept this bounded local implementation.

## Optional work and stop line

Passkeys, social/upstream OIDC, back-channel logout, machine clients and device
flow are demand-driven follow-ups. SCIM/LDAP/SAML, tenancy, approval flows and
cluster infrastructure are outside the current roadmap, not postponed obligations.

Use the existing overall-plan checklist for authorization history, file scopes,
verification and logical commits; no new per-stage plan ceremony. Keep main,
one logical change per commit, no branch/worktree, push, release or deployment.
