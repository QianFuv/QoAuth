# QoAuth lightweight plan and execution checklist

Protocol Version: 2026-09-05
Protocol Revision: autonomous-execution-git-flow-v2

Revision 9, 2026-10-25. Status: A5 bounded implementation and available local
verification complete; submitted for parent acceptance. Native Linux/image runs
remain unverified. No end-user internal-plan approval wait or new feature phase is
introduced. This is the single checklist; earlier stage records below are historical.
The final evidence section owns current results and limits.

## Outcome and boundaries

One Go process, one SQLite database, one issuer, embedded React/TypeScript/Vite/Semi
UI. Build Windows/Linux amd64 binaries and validate Linux amd64 container config.
Use standard OIDC code with S256 PKCE, signed JWT ID Tokens, opaque access tokens
and authenticated introspection. A genuine standard Go RP must exercise the actual
OP handler and SQLite adapter; a private protocol cannot substitute for OIDC.

M1 used synthetic fixture callbacks. M2.1 implements the real authentication service,
but all validation still uses temporary synthetic credentials, keys and databases.
No real user directory, durable real credentials, SMTP, external OAuth bindings,
public deployment, firewall changes or system/runtime installation. No PostgreSQL,
Redis, queues, enterprise tenancy, SCIM/LDAP/SAML or HA. Stay on main, one logical
change per local commit; no worktree, branch, push, release or deployment.

HTTP behind the operator's existing trusted HTTPS reverse proxy is the primary
deployment design. Native listening defaults to loopback; a container listens on
0.0.0.0 internally while host publication defaults to 127.0.0.1. No proxy/TLS/network
configuration is changed. Production admin MFA remains required; fixture login is
explicitly test-only. Restore uses maintenance mode, session/token invalidation and
reconciliation of rolled-back user/client/key settings, without distributed epochs.

## Authorization and independent review

- A0: Original task authorized research, local Git initialization and document
  commits, not business implementation.
- A1: Later scope correction required lightweight personal/small-team self-hosting
  and removed enterprise requirements. Incorporated in revision-2 documents.
- A2: Parent's 2026-10-25 message beginning "Parent 已阅读总体计划全文及11项待决策和模块清单"
  accepted the direction, lifted the stage stop, authorized the listed M1 local
  implementation/tests and logical commits on main, and required no additional
  end-user approval of this internal plan. This records that actual message, not
  retrospective approval or a fabricated quote from the original end user. The
  task-specific instruction overrides redundant template/commit approval ceremonies
  within M1; protected external/system/real-credential actions remain excluded.
- Independent review: Astra/xhigh/default, all 11 Markdown files at clean HEAD
  `049400d545d3f2a6feff9152f45e625aa92be4ed`: approve with changes. Six gaps: complete
  invalidation rules; restore client/key checks; resource identity/audience mapping;
  first-run/container boundary; integrated feasibility slice; offline_access example.
  These are design findings, not runtime vulnerability findings or test passes.

## Historical M1 execution checklist

Baseline: clean main at `049400d545d3f2a6feff9152f45e625aa92be4ed`, no user edits.
Local commits use explicit A2 task authorization, not inferred blanket Git Flow.
Inspect diff/index, stage explicit paths, record parent/message/paths/fingerprint,
run normal hooks, and reconcile resulting HEAD for every logical commit.

| Task | Bounded files/result | Required checks | Intended commit | State |
|---|---|---|---|---|
| T1 | Existing docs: A2, proxy/container design, six review fixes | V1 | docs(plan): accept parent review and authorize local M1 | Done: 5de6f7c |
| T2 | go.mod/go.sum, web, asset handler, cmd, ignore/build/container files | V2; V6 consolidated in T5 | feat(scaffold): embed the lightweight application shell | Done: 5f74b31 |
| T3 | internal/store and SQL migrations | V3 atomic SQLite transitions | feat(store): enforce atomic sqlite credential transitions | Done: bbf7ad4 |
| T4 | Test-only fixture OP/RP/resource integration, resolved Go graph and store failure tests | V3, V4 actual standard endpoints | feat(oidc): validate the standard fixture flow with sqlite | Done: 76467f1 |
| T5 | Existing README/architecture/dependencies/integration/roadmap/checklist evidence | V1-V6 applicable checks | docs(validation): record M1 evidence and remaining gates | Done; commit identifiable by its exact message in Git |

Record a necessary separate logical correction's scope/message before committing
under A2. No unrelated changes or history rewriting. M2.1 was subsequently accepted
under A3 below; the M1 baseline is historical and must not be restored.

## Verification and acceptance

- V1: `python .git/validate-docs.py`, `git diff --check`, source/diff review.
- V2: `npm ci` and `npm run build` in web; embedded route tests: known UI routes
  work, unknown API/protocol paths stay 404, no runtime Node/assets directory.
- V3: `go test ./cmd/... ./internal/...`, `go vet ./cmd/... ./internal/...`, gofmt. SQLite version/settings checks,
  20 concurrent one-use attempts and injected transaction failures. OP integration
  must use the same adapter, not separate fake storage and SQL demonstrations.
- V4: Independent Go RP discovery/code+S256/signed ID Token/UserInfo; consent/refresh,
  authenticated audience-bound introspection, revoke and restart persistence.
  Reject wrong PKCE/redirect/client, code/recovery replay, refresh races, tampered
  opaque tokens, revoked users/grants and unauthorized resource introspection.
- V5: Pin actual dependency graph, review official advisories/licenses, run available
  source/binary vulnerability scans. GOTOOLCHAIN=local; task dependency downloads
  do not authorize system toolchain upgrades.
- V6: windows/amd64 and linux/amd64 builds, native Windows smoke, Docker Compose
  config validation. Native Linux/container execution requires an already usable
  runtime. Report missing runtime checks; do not start/install system components.
  Cross-compilation and config validation are not runtime test passes.

Security accompanies implementation. Complete admin/password/MFA product flows,
full local conformance-suite coverage and missing native environment runs remain
identified M2/M3 obligations, not fictional M1 passes. No production-readiness claim.
Reuse evidence only for equivalent code/tests/dependencies/environment. Final M1
acceptance is parent review of actual results and the minimum proposed next slice.

## Workflow resources

Refetched directly and validated on 2026-10-25 using SHA256-UTF8-LF-v1 and matching
protocol. The explicit A2 simplification overrides redundant template machinery.

- [PLAN_TEMPLATE.md](https://gist.githubusercontent.com/QianFuv/08d3789effa14cdcf1c23b5a7abe0ea7/raw/d69c065ce9a1aea57f207f62dadc71a4ddbea914/PLAN_TEMPLATE.md):
  `1ddfdce0058f9187b529634b09921385b6690fb02705cba39b0eb2eddbe03a59`.
- [PLAN_EXECUTION.md](https://gist.githubusercontent.com/QianFuv/63b4c5707ad011baaa80880e93fb310d/raw/3afc10647951bd36b51809197fdc9966511cccd7/PLAN_EXECUTION.md):
  `a2e0029e22cda2a37c0e4e9db06b57430ef5a48e3e13e38adac25f626b951d54`.

## Progress and actual evidence

- M0: 11 documents, seven local commits through the baseline; documentation checks
  passed. Historical plan remains in Git; no history was rewritten.
- M1 entry: Go 1.26.10 windows/amd64, Node 24.22.1, npm 12.2.0, Python 3.12.15.
  Docker CLI/Compose available but Linux engine pipe absent. No environment change
  or real credential created. T1-T5 are complete within the authorized M1 boundary;
  missing native Linux/container execution remains a later acceptance gap.

- T2: Locked frontend build and current embedded route tests passed; npm audit reported
  zero findings. Go test passed (two packages). V6 cross-build/config evidence is
  consolidated in T5 so a cold build does not block independent storage work.

- T3: Six storage tests and scoped vet passed, including two independent SQLite
  connections, 20 concurrent consumes, insertion failure rollback, recovery atomicity
  and subprocess exit before commit. T4 must still test these through actual OP calls.
- V6: Both amd64 shell binaries built and Compose config validated. No Docker engine
  or native Linux execution was performed.

- Verification scope correction: Go ./... traverses web/node_modules; a vet run
  overlapped npm ci and reported a disappearing package path. Use explicit cmd/internal
  Go package patterns and verify only after asset writes finish; this is not a code
  test pass or a waived check.

- T4: Actual ZITADEL OP, SQLite adapter, embedded login, coreos Go RP and separate
  HTTP resource consumer passed together. Final scoped tests report 28 passing
  tests/subtests; vet passes; race tests report 27 passes across store/fixture.
  Coverage includes PKCE/client/redirect binding, code replay, 20-way refresh,
  actual COMMIT failure rollback, expiry, user disable, logout redirect validation,
  resource audience/scope, revocation and introspection timeout denial. Strict
  AES-GCM rejects the library's legacy unauthenticated token format. No production
  login/admin/session/MFA implementation is represented by these fixture passes.

## Final M1 evidence (2026-10-25)

Code evidence corresponds to `76467f1b322553d2f9cfa7abb0e8128326423f3a`; T5 changes
documentation only. Generated assets and local binaries are ignored, not release
artifacts. Checks were run on Windows amd64 with the versions in
[dependencies](../dependencies.md). No benchmark or production capacity result exists.

| Check | Actual result | Limit |
|---|---|---|
| `npm ci --ignore-scripts --no-audit --no-fund`; `npm run build` | Locked install, TypeScript check and Vite build passed | No full browser login/admin test |
| `go test -timeout 60s ./cmd/... ./internal/...` | 28 passing tests/subtests; 4 packages considered | Synthetic fixture; command package has no tests |
| `go vet ./cmd/... ./internal/...`; gofmt check | Passed | Applicable authored Go only |
| `CGO_ENABLED=1 go test -timeout 60s -race ./internal/store ./internal/fixture` | 27 passing tests/subtests; 2 packages | Existing GCC; no compiler installed |
| SQLite live settings | 3.54.1; WAL; foreign_keys=1; synchronous=2 (FULL); busy_timeout=3000 | Windows runtime; single-instance design |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 -test ./cmd/... ./internal/...` | No known vulnerabilities found | Versioned advisory snapshot, not proof of absence |
| Same scanner, `-mode=binary` on both fixture test binaries | No known vulnerabilities found | Includes actual OP/SQLite graph; Linux binary was scanned, not executed |
| Full `npm audit --json` | Zero findings | Dependency advisory coverage only |
| `CGO_ENABLED=0 GOARCH=amd64 GOOS=<windows,linux> go build -trimpath -ldflags="-s -w" ... ./cmd/qoauth` | Both UI shell binaries built | Shell contains no production authentication |
| Same target pair, `go test -c ... ./internal/fixture` | Both OP/SQLite test executables built | Windows tests run; Linux execution missing |
| Windows shell/browser smoke | Embedded page rendered; synthetic form and keyboard focus worked under CSP | T3 snapshot timed out twice; Chrome fallback used; task tabs/process stopped |
| `docker compose config --quiet` | Passed; internal 0.0.0.0 with host 127.0.0.1 publication | Linux engine pipe absent; no image build/run or deployment |
| `python .git/validate-docs.py`; `git diff --check` | Passed | Local links/UTF-8/markers, not every remote URL re-fetched |

The UI smoke is reusable: the final locked rebuild produced the same asset names
and index SHA256 `1dfac9d5f87bb798b2a9c33fd78a50b8939631b3dedb9fb15b6f137d6a3354bf`.
The HTTP test harness owns its ephemeral loopback servers. It retained in-memory
keys while reopening SQLite/provider state; persistent real-key restart, key
rotation and backup restore were not tested. Recovery-code atomicity is a storage
test, not a completed password/MFA recovery product flow.

Local artifact SHA256 values:

| Ignored path | SHA256 |
|---|---|
| `dist/qoauth-windows-amd64.exe` | `0096b709cb1c547da758fd058a7848e55a88d48743c4be802b82b018b09cf4d6` |
| `dist/qoauth-linux-amd64` | `8f9bf7430217b51fe54cebe8fd5d8324767e8040513aa931948150d06f5a800d` |
| `dist/qoauth-fixture-windows-amd64.test.exe` | `e9107664823b7337757d583d888778ac400768a2a9c17bc84fce836c4d2f414c` |
| `dist/qoauth-fixture-linux-amd64.test` | `7b93634659637bb00937b6f4938e120767265b9f66ec89edc625f52eaca09f63` |

F1-F4 support the selected lightweight approach within M1's synthetic/local scope.
Formal OpenID conformance, a second independent RP, real SSO/session assurance,
password/TOTP/ownership/recovery, license bundle/SBOM, native Linux/container,
backup/restore and key rotation remain future gates. No user approval wait is a
technical blocker; the parent owns the next implementation decision.

## Historical M2.1 proposal, accepted by A3

Parent accepted this concrete proposal in A3 below.
Keep the same process/database/UI stack and existing single checklist.

1. Add production configuration and a host-invoked initialization command. A fresh
   database gets one owner with restricted enrollment state, never a default account
   or anonymous web claim. Re-running initialization cannot reopen ownership.
   Implement key-file handling and credential persistence; exercise it only with
   synthetic temporary test data until real credential creation is authorized.
2. Implement password login (mature Argon2id), bounded throttling, TOTP enrollment/
   verification and one-use recovery codes. Owner/admin APIs require completed MFA
   and fresh proof for sensitive changes; recovery never bypasses MFA. Provide one
   simple host-controlled owner recovery command, with session/grant invalidation.
3. Connect a production OIDC adapter to real user/client/session records, keeping
   the M1 atomic code/refresh invariants. Add exact app callbacks, allowed scopes,
   resource-introspection identity, consent and minimal user/app administration.
   Support two synthetic RPs sharing eligible SSO, prompt/max_age, local logout and
   session/grant revocation. Preserve the independent RP-cookie boundary.
4. Add only the login/MFA, account/session and user/application management screens
   required for that path. Keep activity records simple. Deliver one runnable Go RP
   example using standard libraries; no private OAuth SDK.

Bounded files: `cmd/qoauth`, new small `internal/config`, `internal/identity`,
`internal/session` and `internal/oidc` packages where actual boundaries require them;
existing `internal/store` plus the next SQL migration; `internal/webassets`,
`web/src`, `examples/go-rp`, lockfiles and the existing docs. Keep
`internal/fixture` as regression support, never an application login mode.

Acceptance: fresh synthetic install -> restricted owner -> confirmed TOTP -> add
one ordinary synthetic user and app -> standard RP login/consent -> protected API
-> refresh -> revoke/logout -> rejection -> restart without resurrecting access.
Negative checks cover unowned/repeated init, wrong passwords and throttling,
promotion/reset MFA bypass, TOTP/recovery replay, CSRF/session fixation, exact
client/redirect/scope/audience binding, pending-code invalidation and all M1 races.
Run scoped Go tests/vet/race, browser flow, dependency scans and cross-builds. Add
native Linux/container evidence only when an already authorized runtime is usable.

Excluded from M2.1: public signup, SMTP/email recovery, social login, passkeys,
enterprise protocols/tenancy, generic permission engines, HA and extra services.
Backup/rotation/conformance/release hardening remain M3, with security-critical
account/session regressions required during M2.1 rather than deferred.

Requested implementation authority is limited to repository code/docs/dependencies,
temporary synthetic data and loopback test servers, and logical main commits.
No additional OS/account privileges are expected for that work. Creating a real
owner, durable operational signing/encryption/client keys, production state,
reverse-proxy/TLS/network changes, starting a missing Docker/system service, real
OAuth/SMTP bindings or deployment are separate concrete actions requiring explicit
authorization; none is needed to write/test the proposed code. Parent can approve
or narrow this slice directly without transferring internal plan review to the user.

## M2.1 authorization and execution

A3: The parent message beginning "我已审阅 M1 最终证据和 M2.1 三部分计划，批准在原授权范围内实施 M2.1"
explicitly accepted M1 and authorized this implementation, tests and logical main
commits. Verified baseline is clean main at
`0a77f7bfec48baf65e9e498f8995109652dbc715`; the older 049400d handoff is historical,
not a rollback target. This authorization covers code for persistent keys/init,
but execution only with temporary isolated synthetic credentials/databases.

| Task | Bounded scope | Commit | State |
|---|---|---|---|
| U1 | This acceptance/execution record | docs(plan): record M2.1 implementation approval | Done: ef23df8 |
| U2 | Password/TOTP/key primitives, SQLite identity/session/client transitions, dependency locks and tests | feat(identity): persist credentials and enforce atomic session security | Done: 788a4f6 |
| U3 | Production config/CLI/OP, account/management APIs, Go RP, private-directory/store support and container command wiring | feat(auth): connect local accounts to standard OIDC sessions | Done: 0503b9d |
| U4 | Embedded Semi screens, UI response contract and bounded browser harness | feat(ui): add local authentication and management screens | Done: 20098bf |
| U5 | Final security/build evidence, deployment limits and next bounded scope | docs(validation): record M2.1 authentication evidence | Done; this evidence commit is identifiable by its exact message in Git |

Mandatory checks extend V1-V6: secure cookie attributes and origin/CSRF binding;
TOTP window replay and recovery-code contention; disable/password/factor recovery
invalidate sessions, pending codes and grants; exact redirects and authorized
audiences/scopes; persisted signing keys survive restart; credential canaries do
not appear in logs. Run actual production handlers with standard Go RP E2E,
negative tests, race, dependency scans and browser verification. Native Linux and
container execution remain separately unverified unless a permitted runtime exists.
No additional approval gate or enterprise framework is introduced.

U2 evidence: 35 tests/subtests across the existing and new packages passed; scoped
vet passed. Race checks passed 15 identity/storage tests. New checks cover private
Windows key-file ACLs and restart, factor context binding, Argon2 parameter bounds,
TOTP counter windows, 20-way MFA/recovery contention, real COMMIT rollback, stale
password verification, restricted owner recovery and pending-code/family invalidation.
The first tests exposed and corrected noncanonical Windows DACL comparison and
modernc multi-statement positional parameter reuse; mixed parameter sets now use
separate statements in the same transaction. No HTTP implementation pass is implied.

U3 evidence: the full scoped race command now passes 58 tests/subtests across seven
packages, including the real production OP/account handlers and Go RP example;
scoped vet and Compose config validation pass. The protocol/CLI tests cover initial
owner enrollment, two RP clients sharing SSO, cookie/CSRF restrictions, same-second
forced login, silent-login errors, denied/cross-browser consent, exact redirects,
scope/resource rejection, refresh races, security-change invalidation and confirmed
RP logout. A separate process loads the same temporary keys and decrypts saved
synthetic state; key loss never reopens initialization. Log/error/activity canaries
are absent. U3 also wires container data/issuer arguments required by the new CLI;
the image is still not built or run. Browser UI and final graph scans remain U4/U5.

U3 review correction (A3 scope): `fix(auth): enforce unambiguous authorization boundaries`
contains only production protocol/storage code, migration 003, regression tests and
this record. Independent reviews of 0503b9d found duplicate-parameter decoding,
callback method confusion and consent-withdrawal races. New tests reproduced the
first two failures (unexpected HTTP 302); the corrections now pass 41 scoped
OIDC/store tests. Automatic completion rechecks consent in its transaction; explicit
password-authentication events distinguish fresh login from cookie/MFA rotation.
Deferred foreign-key COMMIT failures prove consent/completion and recovery rollback.
The remaining review findings (resource limits and file ownership/privacy) are being
handled before final validation. No production readiness claim is made.

U3 storage correction (A3 scope): `fix(storage): validate private database ownership`
covers identity private-file checks, CLI startup/init, their tests and this record.
Windows validates DACL and trusted owner, Unix checks owner and mode, and production
startup rejects unsafe database/WAL/SHM/journal files and reparse/link targets. Only
new files receive private permissions. A missing database beside existing keys
requires restoration instead of creating a new owner. Ten Windows tests/subtests
pass; the Unix identity test binary cross-compiles, but has not run on Linux.

U3 availability correction (A3 scope): `fix(auth): bound anonymous state and trust proxy addresses`
covers the production address/throttle configuration, SQLite quotas/expiry indexes,
CLI flag and regression tests. An optional single trusted proxy IP must overwrite
X-Real-IP with exactly one parsed client address; other peers cannot spoof it.
Atomic limits cap anonymous sessions at 4096, login transactions at 2048 globally
and 16 per browser; creation prunes at most 64 expired rows of each kind.
Forty-four scoped tests pass, including two-connection contention and independent
client allowances behind one proxy. Real proxy/network configuration was not changed.

Second independent review, Astra/xhigh/default at 9cf4784, found one remaining
P1: denied admission rolled back bounded expiry cleanup, so an old oversized
backlog could never recover. `fix(storage): retain cleanup when admission is denied`
changes only store admission/pruning behavior, its tests and this record under A3.
Cleanup now commits before returning the quota error, while the admission check and
insert stay atomic. Regressions cover 4300 expired anonymous sessions, 2200 expired
requests and a per-browser expired quota behind another browser's cleanup batch.
The complete scoped race run passes 79 tests/subtests in seven packages (three
explicit helper/manual-harness skips); later UI/RP corrections are included in that
working-tree verification. Review reports are static evidence, not test execution.

Browser/client correction (A3 scope): `fix(oidc): preserve secure browser logout interoperability`
covers the Go RP, OP/static response headers, their regression tests and this record.
The RP now binds UserInfo sub to the verified ID-token subject, uses HTTPS __Host-
cookies, and tests successful callback/refresh/logout plus nonce/subject failures.
Chrome showed that no-referrer made native form Origin null. Strict-origin keeps
URLs/credentials out of Referer while preserving canonical Origin checks. Logout
CSP allows only self plus the validated redirect origin; static HTML preserves it.
The final Chrome flow reached the registered signed-out page and /profile then
returned sign-in-required. These changes are covered by the 79-case race run.

U4 evidence: TypeScript/Vite build passed (274.37 kB JS, 52.96 kB CSS; gzip
84.37/9.06 kB). The React review used the local react-best-practices skill for
component/hooks/form checks; no frontend framework or dependency was added.
T3 snapshot failed twice with `Preview automation snapshot timed out after 15000ms.`;
Chrome fallback exercised password login, consent, the independent Go RP callback,
UserInfo, confirmed RP/OP logout, rejection afterward, owner TOTP, user create/disable,
public application registration/assignment and account/session rendering.
Browser testing caught and fixed the protocol library's space-delimited scope
serialization (UI requires arrays) and native logout response-header conflict.
The final tested origin emitted no application-script console errors; installed
Chrome extensions emitted their own CSP errors, and their settings were not changed.
The harness is opt-in, test-only, bounded to ten minutes and cleans temporary state.
All test servers/RP child processes were stopped; the Chrome tab was closed.
Screenshot evidence is ignored `.cache/m21-admin.png`, containing synthetic state.

Artifact scan correction (A3 scope): `build: retain symbols for vulnerability verification`
changes only scripts/build.ps1, Dockerfile and this record. Both initial stripped
(-s -w) artifacts reported GO-2026-5932 because govulncheck v1.9.0 falls back to
module-level matching without symbols. The actual graph uses Argon2, not OpenPGP.
Retaining symbols (-w, no -s) lets the final Windows/Linux binaries both pass with
zero affected symbols/imported packages; one non-imported module advisory remains.
See the tagged scanner implementation and official advisory in dependencies.md.
Both targets rebuild; Compose config validates. Actual Windows binary initialization,
HTTP discovery/embedded UI, two separate serve processes and unchanged JWKS pass
using a temporary synthetic installation, removed after stopping both processes.
Linux and container execution remain unverified; no service was installed or started.

## Final M2.1 evidence (2026-10-25)

Implementation/build reference: `6144c9c12b8b15eb6a412321cbfc4bab08cf9ac5` on main.
U5 changes documentation only. Earlier 28/58-test results above are historical;
the final applicable run below includes the review corrections and UI/RP contracts.
The primary agent inspected results; independent reviews were static only.
All accounts, credentials, keys and databases used for verification were synthetic.

| Check | Actual result | Evidence/limit |
|---|---|---|
| `go test -count=1 -timeout 180s -race -json ./cmd/... ./internal/... ./examples/...` | 79 tests/subtests and seven packages passed, zero failures | `.cache/m21-race.json`; Windows, existing GCC, CGO_ENABLED=1; helper/browser entrypoints skipped in ordinary suite |
| `go vet ./cmd/... ./internal/... ./examples/...`; Go formatting | Passed | Scoped packages avoid frontend dependency sources |
| `npm run build` in web | TypeScript/Vite passed | 274.37 kB JS and 52.96 kB CSS; gzip 84.37/9.06 kB |
| `npm audit --json` | Zero findings at every severity | `.cache/m21-npm-audit.json`; current locked graph |
| govulncheck v1.9.0, source and `-mode=binary` on both final artifacts | Zero affected symbols/imported packages | `.cache/m21-vuln-source.txt`, `m21-vuln-windows-final.txt`, `m21-vuln-linux-final.txt`; one required-module advisory in unimported OpenPGP remains |
| `CGO_ENABLED=0 GOARCH=amd64 GOOS=<windows,linux> go build -trimpath -ldflags="-w" ... ./cmd/qoauth` | Both final executables built | Symbol tables retained for precise vulnerability scans; no release publication |
| Native Windows executable | Explicit init, discovery, embedded UI, JWKS, stop and separate-process restart passed | Same temporary database/key file and stable JWKS; task processes stopped and temporary installation removed |
| Browser + actual production adapter + independent Go RP | Login, consent, callback/UserInfo, confirmed logout and rejection afterward passed | Chrome fallback after two T3 snapshot timeouts; no external account/binding |
| Browser management | Owner TOTP, user create/disable, public client registration/assignment and account/session views passed | Ignored screenshot `.cache/m21-admin.png`; synthetic state; no full accessibility claim |
| `docker compose config --quiet` | Passed | No Docker image build/run; no native Linux runtime execution |
| `python .git/validate-docs.py`; `git diff --check` | Passed before U5 commit | UTF-8, local links/anchors, placeholders and whitespace; not all remote sources refetched |

The three ordinary-suite skips are `TestBrowserHarness`, `TestKeyProcessHelper`
and `TestCrashHelper`. The browser harness ran separately; child-process helper
entrypoints are invoked by their owning tests. They are not Linux/container passes.
No application-script console errors appeared in the final browser flow. Unchanged
user extensions produced their own CSP messages. T3 preview snapshot timed out
twice; Chrome completed the equivalent flow. Test servers and RP child processes
were stopped, the Chrome task tab closed, and disposable browser state removed.

The source scan and final binary scans retain the unimported-module warning rather
than suppress it. See [dependency evidence](../dependencies.md#selected-set-and-final-local-evidence)
for GO-2026-5932 and the stripped-binary scanner fallback. The module as a whole is
not described as vulnerability-free. The final binaries have the following actual
hashes; these supersede the historical M1 executable hashes, not its old evidence.

| Ignored artifact | Bytes | SHA256 |
|---|---|---|
| `dist/qoauth-windows-amd64.exe` | 17699328 | `89920d9bdd2e7d323d29f5c8f9aa39bcfb4baf527a6e923bbb4de6a4bc66f855` |
| `dist/qoauth-linux-amd64` | 17616966 | `56813b90e80ecc25e8ae3bb0ba3858d3dfe42745ac2ea06451661ed3cfc479ae` |

M2.1 local commits, in order after accepted 0a77f7b:

| Commit | Logical result |
|---|---|
| ef23df8 | Record A3 implementation authority |
| 788a4f6 | Identity primitives, persistent keys and atomic SQLite authentication |
| 0503b9d | Production OP/API/CLI and independent RP integration |
| 7d090db | Duplicate/callback/consent/freshness and transaction regressions |
| 761d456 | Private file ownership and SQLite sidecar validation |
| 9cf4784 | Trusted proxy addresses and bounded anonymous state |
| 0779231 | Commit bounded cleanup even when admission is denied |
| 9484edb | Secure standards-based browser logout and RP subject checks |
| 20098bf | Embedded authentication and management screens |
| 6144c9c | Retain binary symbols for vulnerability verification |

Remaining acceptance limits: native Linux/image runtime, second independent RP
implementation and local OIDF profiles, backup/restore and key rotation, bounded
authenticated-state/activity retention, disk pressure/password-cost measurements,
broader password-blocklist assessment, full accessibility and distribution notices/
SBOM. M2.1 does not implement public signup, SMTP, external IdPs, passkeys, enterprise
protocols or a general authorization framework. UI/API are intentionally small:
registration/assignment/disable are implemented; comprehensive client editing,
secret rotation and arbitrary admin-role management are not implied.

Actual key custody is a protected local plaintext `keys.json` containing PKCS#8 RSA
and the AES root secret, with encrypted TOTP records in SQLite. File ownership/ACL
checks are implemented; nested encryption under a co-located key was not added.
The host administrator is trusted. This implementation choice and outstanding
release gates are explicitly presented for parent review; no production-ready claim.

## Historical M2.2 proposal, accepted by A4

Parent accepted this proposal in A4 below; no final-user internal-plan approval
wait is introduced. Continue
the same single checklist, one process/issuer/SQLite database, embedded UI and main.

1. **Stopped-service backup and safe restore.** Add small local CLI commands and
   exclusive installation locking shared with serve/init/recovery. Snapshot only
   while the installation is exclusively held; validate SQLite integrity/schema
   and key binding. Restore into a separate private directory, never overwrite the
   source installation by default. Before serving restored state, invalidate all
   browser/restricted sessions, pending login/codes, grants/access/refresh tokens
   and one-use recovery authorizations/codes. Require a short local maintenance
   reconciliation of rolled-back users/passwords/MFA and client secrets/redirects/
   assignments/scopes/key state; uncertain clients remain disabled. The old snapshot
   cannot reveal changes made after it. No daemon, cloud backup or distributed epoch.
   **Files:** `cmd/qoauth`, `internal/identity`, `internal/store`, the next ordered
   migration only if persistent maintenance state is needed, focused tests and docs.
   **Acceptance:** temporary snapshot -> revoke/use recovery code -> restore cannot
   revive those credentials; unsafe paths, wrong/missing keys, corrupt/unsupported
   databases and a running installation fail closed while preserving source files.
   Reconciliation is required before ordinary serving can resume.
2. **Local signing-key rotation.** Keep the existing AES root stable. Introduce only
   the minimal persisted signing-key lifecycle needed to publish the new public key,
   switch signing and retain old public keys for the documented token/cache/skew
   window. Do not retain obsolete private signing material beyond its need or add
   KMS/HSM/HA infrastructure. Missing/corrupt/interrupted state must fail closed.
   **Files:** `internal/identity`, `internal/oidc`, `cmd/qoauth`, relevant storage
   metadata if needed, protocol/restart tests and the existing runbook.
   **Acceptance:** old/new ID tokens verify only within their valid periods, new
   tokens use the selected key, restart preserves the selection, interrupted writes
   preserve recoverable prior state, and retired keys disappear after the window.
3. **Bounded maintenance and failure evidence.** Add small in-process or explicit
   maintenance batches for expired authenticated sessions/tokens and local activity
   history. Preserve consumed-code/refresh lineage for the entire required replay
   detection period; no timer service or queue. Document SQLite disk-full/lock and
   migration/backup failure behavior. **Files:** `internal/store`, focused tests,
   CLI wiring where needed and existing operational/security docs.
   **Acceptance:** batch bounds, valid-session preservation, durable family-replay
   revocation, interrupted cleanup, disk/commit errors and restart recovery. Run
   scoped tests/vet/race, applicable scans and target builds after relevant changes.

Permissions needed for this proposal are repository edits/logical local main commits,
temporary isolated synthetic files/credentials, existing tools and bounded loopback
test processes. It does not require real user access, production backups, host ACL
repair, TLS/firewall/account changes, system services, external writes or deployment.
Testing against a real installation or creating a usable Linux/container runtime
would require a separate concrete action and authorization; do not silently install
or start system components. Until a permitted runtime exists, list those runtime
checks as unexecuted. Second-RP/OIDF and final packaging gates remain visible release
work, not a reason to add enterprise functions or claim M2.2 already validated them.

## M2.2 authorization and execution (A4)

The actual 2026-10-25 parent message beginning "已审阅 M2.1 交付及 M2.2 计划，批准这个最小运维闭环的代码实现与隔离合成测试"
approves this scope and logical local main commits. It accepts keys.json protected
by strict ownership/ACLs with encrypted TOTP in SQLite, requires consistent stopped
backups, new-directory restore without source overwrite, signed-token invalidation,
bounded signing overlap without AES changes, dependency-aware retention and explicit
Linux/image gaps. It grants no real credential/installation/system/network action.
Baseline is clean main `15e2f421c522113d4529e3872710fda512174d7c`.

| Task | Bounded scope | Intended commit | State |
|---|---|---|---|
| N1 | This accepted execution record | docs(plan): record M2.2 maintenance approval | Done; identifiable by exact message |
| N2 | Private installation lock, shared CLI lifecycle, isolated process tests | feat(storage): serialize local installation operations | Done; identifiable by exact message |
| N3 | Persistent signing rotation and OP verification window, restart/interruption tests | feat(keys): rotate signing keys with bounded overlap | Done; identifiable by exact message |
| N4 | Consistent backup, new-directory restore, restricted recovery and signed-token quarantine | feat(maintenance): add isolated backup and restore | Done; identifiable by exact message |
| N5 | Bounded dependency-aware state and activity cleanup, failure regressions | feat(storage): bound maintenance without losing replay evidence | Done; identifiable by exact message |
| N6 | Actual checks, operational instructions and bounded remaining product/release list | docs(validation): record M2.2 maintenance evidence | Done; evidence commit identifiable by exact message |

Verification reuses unchanged M2.1 UI evidence; no UI redesign is planned. Required
new tests cover lock contention/process exit, unsafe files, backup integrity/version/
binding, source preservation, restore revocation/owner restriction, fresh JWKS and
cached-ID-token expiry boundaries, rotation restart/interruption/concurrency, and
cleanup retention/commit failure. Run scoped Go tests/vet/race, source/binary scans,
both target builds and Compose configuration validation. Native Linux/image runs
remain unexecuted without an already usable permitted runtime. A4 permits necessary
in-scope corrections and their logical commits; record each correction before use.

N2 evidence: 12 targeted identity/CLI tests passed on Windows. A permanent private
lock file uses nonblocking OS locking; serving and all local mutations share it.
Same-process/cross-process contention rejects a second owner; process exit releases
the lock without unlinking the file. Tests use only temporary installations. Unix
locking is included in the later cross-build; no Linux runtime pass is implied.

N3 evidence: the full scoped Windows race run passed 89 tests/subtests across seven
packages (`.cache/m22-rotation-race.json`). Rotation stages one synced candidate,
commits its DB binding and atomically publishes; either interrupted commit boundary
is explicitly resumable with the same key/deadline. Startup rejects pending state.
Real COMMIT failure preserves the old binding. Concurrent CLI attempts yield one
success. A new process verifies the rotated binding/key and unchanged AES/TOTP.
The standard RP fetches the new kid; existing opaque access/refresh survive ordinary
rotation. Each JWKS call drops the previous public key at its persisted ten-minute
deadline (five-minute ID tokens plus supported five-minute clock-skew allowance).
Only public old material is retained; repeated rotation during overlap is rejected.
Native Linux and filesystem power-loss behavior are not runtime-verified here.

Two independent baseline design reviews confirmed no OP key-cache hook is needed,
but RP application caches do not have a reliable finite TTL. Restore must use fresh
RSA without overlap and enforce a persisted quarantine anchored to stopped issuance,
not backup time. Reviews also require waiting for HTTP shutdown before releasing
the lock, non-migrating exact snapshot inspection, incomplete-restore markers and
preserving session/request parents of live grants. These are integrated into N4/N5;
static review itself is not test evidence.

N4 evidence: 115 tests/subtests across eight packages pass the full scoped Windows
race run (`.cache/m22-restore-race.json`); frontend TypeScript/Vite build passes.
Snapshot validation is non-migrating and compares the exact supported ledger and
actual schema, integrity/FKs, owner invariant, file hashes and RSA/AES binding.
Locked VACUUM INTO includes committed WAL data. Fixed private snapshot members,
link/reparse rejection, synced output and an incomplete-operation marker protect
publication; restore always creates a separate new directory and preserves source.
HTTP shutdown drains requests before SQLite/lock release, including timeout handling.

Restore uses fresh RSA with no old verification keys and unchanged AES. It clears
all authentication/consent/assignment state, disables all accounts/clients, persists
a ten-minute signed-token quarantine and requires reviewed local owner recovery.
The old issuer must remain stopped: a destination lock cannot stop copies elsewhere.
Tests show a warm coreos RP still accepts an unexpired cached-key token, a cold RP
rejects it under new trust, and expiry rejects it after the supported window. This
is not immediate remote revocation or cancellation of independent RP cookies.

Recovery closes the practical re-enablement gap: `recover-user --username ...
--confirm` replaces a disabled ordinary user's password/MFA, drops old roles/access
and requires new enrollment. The existing application form can explicitly replace
a disabled client with complete reviewed settings, fresh confidential secret and
no old assignments/consent. It cannot change resource/client type. Resource APIs
require their dependent applications disabled first; resource permissions are
rechecked inside the registration transaction. No general policy/approval system
was added. Browser confirmation of this small form option is consolidated in N6;
production API tests already cover credential replacement and stale scope rejection.

N5 evidence: targeted store/maintenance/CLI tests and scoped vet pass. Cleanup is
an explicit stopped-service command, one transaction with 1-256 rows per table,
default 128. Tokens require an expired grant and no future token in that family;
then grants, requests and unreferenced sessions can be removed in dependency order.
Activity retains 30 days separately. Indexes support expiry/reference predicates.
An oversized family progresses across batches; an expired browser cookie supporting
a live refresh family survives. Deferred COMMIT and real SQLITE_FULL (temporary
max_page_count limit, no host disk pressure) leave no partial cleanup. Initial test
failures were a driver error-string assumption and a null synthetic payload; tests
now check SQLite error code 13 and a valid non-null payload.

Retention does not change baseline replay semantics: still-valid consumed credentials
revoke their family on reuse; expired ancestors are rejected before replay handling
and do not revoke live successors. Their retained rows are not claimed as evidence
of family-lifetime replay detection. No daemon or persistent schedule was created.

N4 correction under A4: `fix(maintenance): reject schema prefix lookalikes` covers
only snapshot schema filtering, its rehashed malicious-trigger regression and this
record. Independent review found SQL LIKE treated the underscore as a wildcard.
The new test first failed because a `sqliteXrestore` trigger bypassed validation;
literal-prefix GLOB matching now rejects it before destination creation.

N4 correction under A4: `fix(oidc): fence requests across client replacement`
covers a clients.generation migration, application lookup/replacement, transactional
login admission, OP handoff and affected regression fixtures. The deterministic
old-config-read -> replacement/reassignment -> insertion test failed before the
fix. Admission now compares the observed generation inside its insertion transaction;
replacement still deletes existing requests and invalidates codes/grants. A fresh
request using the new generation succeeds. No policy framework or infrastructure
is introduced; this is a local concurrency fence for existing client replacement.

N3 correction under A4: `fix(keys): publish only complete rotation candidates`
covers key serialization/staging, the CLI pre-staging binding check and subprocess
interruption tests. A forced child-process exit during the direct candidate write
first reproduced an unusable published candidate. The corrected path writes a
private synced keys.staging.json, then atomically publishes keys.next.json before
any DB update. Only explicit rotation retry reclaims a validated private unpublished
staging file. A completed candidate retains the same key/deadline; malformed published
candidates still fail closed. This does not claim physical power-loss validation.

N5 coverage follow-up under A4: `test(maintenance): cover cleanup after revocation`
adds cleanup after replay revocation, another reopen and retained-lineage/successor
denial assertions. CLI coverage now includes cleanup in lock/incomplete guards,
invalid batch limits, default 128-row progress and strictly count-only JSON. The
independent cleanup review found no P0/P1/P2; these were coverage recommendations,
not claimed runtime defects.


## Final M2.2 evidence (2026-10-25)

Implementation/tests baseline: `22d1b39ab274c69f7189e6260b454c62f676d4e3` on main.
Final changes after that baseline are this documentation closure only. Go/npm graphs
are unchanged from M2.1. All state/key/recovery operations used isolated synthetic
installations; no real accounts, external bindings, services or deployment were used.

| Check actually run | Result and limits |
|---|---|
| `go test -count=1 -timeout 180s -race -json ./cmd/... ./internal/... ./examples/...` | 127 tests/subtests passed across eight packages on Windows; `.cache/m22-reviewed-race.json`. Five opt-in helper/browser entrypoints skipped in the ordinary run; subprocess parents invoke helpers and browser was run separately |
| Scoped `go vet` and gofmt inspection | Pass; existing Go 1.26.10, project caches; race uses existing MinGW compiler with CGO=1 |
| SQLite settings and migrations | Runtime 3.54.1, WAL, synchronous=FULL (2), foreign_keys=1, busy_timeout=3000; migrations 001-007. Backup validation never silently migrates its source |
| `npm run build` | TypeScript/Vite pass; final JS 275.03 kB, CSS 52.96 kB (gzip 84.56/9.06). This unchanged UI build was reused for final binaries |
| `npm audit --json` | Zero info/low/moderate/high/critical findings at this check; `.cache/m22-npm-audit.json` |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 -test ./cmd/... ./internal/... ./examples/...` | Zero affected symbols or imported packages; one advisory in a required but unused module package, GO-2026-5932 OpenPGP; not a claim that all x/crypto packages are safe |
| Same scanner, `-mode=binary`, both final target files | Zero affected symbols/imported packages; same non-imported module advisory. Retained symbols avoid a misleading stripped-binary fallback |
| `CGO_ENABLED=0 GOARCH=amd64 go build -trimpath -ldflags='-w'`, GOOS windows/linux, `./cmd/qoauth` | Both final executables built; Linux identity test binary also cross-compiles. These are not native Linux execution |
| `docker compose config --quiet` | Pass. Docker image build/run and non-root volume behavior not executed; no daemon started |
| Final Windows binary isolated smoke | Init, embedded login/discovery, restart with same JWKS, locked backup rejection, snapshot, signing rotation/new JWKS and unchanged AES, fresh restore, reviewed owner recovery/quarantine, count-only cleanup, secret-free stdout/stderr all pass; `.cache/m22-binary-smoke.txt` |
| Browser via production adapter/embedded UI | Existing owner login+TOTP, fresh-MFA rejection/relogin, public client create/disable/re-register with narrower scopes, resulting application/activity visible. `.cache/m22-admin.png`; test harness stopped and temporary state removed |
| Document UTF-8, local links/anchors, conflict markers, Git diff whitespace | Validated with `.git/validate-docs.py` and `git diff --check`; 11 Markdown files remain the complete review package |

The native T3 preview failed twice at snapshot with `Preview automation snapshot
timed out after 15000ms.` Chrome was used as the permitted fallback. Its console
reported existing extension/userscript CSP and extension messaging failures, not a
QoAuth application exception. The captured viewport shows the replacement activity
sequence; full-page screenshot capture timed out, so no full-page image is claimed.
The browser harness completed in 586 seconds and cleaned up; later Go-only review
corrections did not change the UI bundle. Their actual OP/storage paths are covered
by the final race suite. M2.1's broader standard Go RP/browser flow remains prior
unchanged UI evidence, not newly rerun in this smaller browser slice.

Final artifact identity (local, ignored, not published):

| Artifact | Bytes | SHA-256 |
|---|---:|---|
| `dist/qoauth-windows-amd64.exe` | 17810432 | `6418e7e7516c912494165e52e42083abb438f469d8a095f20b646507a740ff1b` |
| `dist/qoauth-linux-amd64` | 17725692 | `b5f4812e5f1fcde5032d8c174317c1c7896586b6e226ef9aa7de6b02ffea0465` |

Review closure: independent cleanup review at 3a8366f found no P0/P1/P2 and suggested
coverage additions now committed. Maintenance review at f9952e7 found the two P1s
and P2 described above; before/after regressions reproduced all three. Independent
correction review at d77ac9b found no remaining P0/P1/P2 in those corrections and
immediate callers. Reviews were source-only; the implementation agent executed the
checks. Neither these reviews nor process termination tests establish exhaustive
security assurance or physical power-loss durability.

M2.2 logical local commits after accepted 15e2f42:

- `54f4757` records A4 and the one execution checklist.
- `6e15f78` serializes installation operations with OS locks.
- `d167dd8` implements bounded persistent signing rotation.
- `f9952e7` implements validated backup/new-directory restore and reconciliation.
- `3a8366f` implements dependency-aware bounded cleanup.
- `c6922ef` rejects schema-prefix lookalike objects.
- `842de0c` fences client replacement against in-flight old configuration.
- `d77ac9b` atomically publishes only complete signing candidates.
- `22d1b39` strengthens post-revocation cleanup and CLI coverage.
- This documentation evidence commit is identifiable by `docs(validation): record M2.2 maintenance evidence`.

No unreviewed enterprise/infrastructure expansion is scheduled. The remaining
bounded product items and finite release gates are in [roadmap](../roadmap.md#remaining-bounded-work):
password-reset UI, disabled-client recovery visibility, password/readiness assessment;
then second RP/OIDF, native Linux/image runtime, load/accessibility, upgrade drill and
license/SBOM packaging. These are proposals for the parent's next internal decision,
not an end-user approval wait or evidence that M2.2 already passed those gates.


## Final bounded closure authorization (A5)

The actual parent message beginning "已审阅 M2.2 证据和剩余工作，批准下一阶段限定收口"
approves the four bounded changes below plus in-scope quality closure, logical
main commits and isolated synthetic tests. It explicitly excludes SMTP/public
recovery, enterprise expansion, push/release/deploy, real credentials and system
service/security changes. Baseline is clean main 7ced5e1256fd82250ffe5c2e3fc3e0604a772472.
This supersedes the prior proposal wait. No further internal-plan approval is needed.

| Task | Bounded scope | Intended logical commit | State |
|---|---|---|---|
| C0 | A5 and this single checklist | docs(plan): record final closure authorization | Committed f0e2227 |
| C1 | Existing password-policy consistency, shared new-password field, precise safe errors | fix(identity): align password policy across entry points | Committed d50d81d |
| C2 | Existing administrator reset API/UI with fresh MFA and CSRF, preserved factor/invalidation | feat(admin): expose controlled password reset | Committed 6f4499e |
| C3 | Disabled-client list/safe settings and explicit review-prefill, no restored grants or secrets | feat(admin): review disabled application settings | Committed 54e34a5 |
| C4 | Minimal bounded readiness with generic public status and store/key/install checks | feat(health): expose bounded readiness checks | Committed b240fc3 |
| C5 | Second pinned RP implementation and focused local protocol checks | test(oidc): verify a second standard relying party | Committed 9c4855e |
| C6 | Existing phase-version restore/upgrade drill, bounded load, browser/accessibility evidence | test(validation): exercise release closure scenarios | Committed 5312400 |
| C7 | Versioned local binaries, graph/NOTICE/license/SBOM inventory and packaging script | build(distribution): assemble versioned local artifacts | Committed 5d56cd5; preview correction below |
| C8 | Actual evidence, runbook and finite remaining gates; no next feature expansion | docs(validation): close bounded delivery checks | Done; commit containing this evidence is identified by its exact message |

A5 authorizes necessary in-scope fixes and their separate logical commits; record
any such correction before committing. Preserve explicit paths, index diff and
commit recovery evidence. No branches/worktrees, history rewriting or external writes.

Acceptance: new-password rules agree for CLI/API/UI including normalized Unicode
and UTF-8 byte limits; administrator reset rejects stale/MFA-incomplete/CSRF calls,
preserves enrolled MFA and invalidates sessions/pending codes/refresh families;
disabled configuration reads never disclose secret/verifier and re-registration
never revives access. Readiness is read-only, time-bounded, secret-free, fails closed
on unavailable/mismatched/maintenance state, and creates no sessions. Run scoped
Go test/vet/race, frontend build, dependency scans, browser and final-binary smoke.
Use already installed/pinned RP/tools; local focused protocol checks are not OIDF
certification. Cross-version testing operates only on copied temporary synthetic
state and never checks out another Git branch. License inventory is not a source
license decision. Record Linux/image execution honestly; do not start/install system
services or change security configuration to obtain a pass.

C1: centralized creation validation and public policy metadata; shared frontend field
counts NFC code points/UTF-8 bytes, retaining existing login semantics. Identity/CLI/OP
suites passed (57 top-level tests); added API policy/error consistency regression passed.
TypeScript/Vite build passed. Windows runtime inspection found no running WSL distro
and no Docker Desktop Linux engine pipe; neither was started. Browser acceptance is C6.

C2: existing reset endpoint is now available in a labeled administrator form,
with explicit revocation/MFA text and password-field clearing after submission.
Production API tests cover non-admin, CSRF, stale/partial MFA, persisted invalidation
and successful new-password plus preserved-factor login; six OP tests/subtests passed.
A storage interleaving proves stale admin freshness and old verified credential
versions cannot authorize reset/login; targeted test and TypeScript/Vite build pass.

C3: disabled applications are listed through an explicit credential-free settings
projection. Review fills the existing form without submitting; assignment choices
exclude disabled clients. Replacement still generates new confidential credentials,
clears consent/assignments and revokes old codes/grants. Targeted API/client-generation
tests and frontend build passed. The initial test assumed 400 for missing assignment;
the existing policy correctly returns 403, now asserted. Browser acceptance remains C6.

C4: GET /readyz makes one read-only query with a one-second context deadline.
It checks available DB, active key binding, exact persisted issuer, maintenance and
restore quarantine. It creates no cookie/session, returns only ready true/false,
and rejects other methods. Two targeted tests passed, including exhausted-pool
timeout, cancellation, wrong binding/issuer, restore blocking and closed storage.
This is readiness, not an external TLS/proxy probe or exhaustive integrity scan.

C5: pinned Zitadel RP discovery, S256 code exchange, nonce/signature/at_hash,
subject-bound UserInfo, refresh and discovered browser-confirmed logout pass against
the production SQLite adapter. Negative PKCE, nonce, access substitution, UserInfo
subject and missing-logout-CSRF checks pass. Coreos and targeted authorization/prompt
regressions were rerun. This second implementation shares the OP module and OAuth2/JOSE
dependencies; it is not fully independent. The RP EndSession helper lacks browser
CSRF/origin confirmation support and is not claimed interoperable. No OIDF suite
was installed or certification claimed; these are targeted local protocol checks.

C3 review correction under A5: `fix(admin): preserve reviewed callback configuration`
covers distinct create/review React keys, newline-separated callback/logout textareas,
matching field styling and a synthetic browser fixture for disabled client ID new.
Independent review found two P2s: key collision retained an edited draft, and comma
splitting could expand a literal URL into additional redirects. Browser execution
confirmed correct review/cancel/remount, no mutation during review, replacement action,
and exact comma-containing callback/logout values after submit -> disable -> reread.
TypeScript/Vite build passed. The correction does not widen application permissions.

C6: 600 seconds/6,000 requests at 10 requests per second, one in-flight request,
100 synthetic users/10 applications: zero failures, p50 1.026 ms, p95 2.855 ms,
max 16.363 ms. Mix is readiness/discovery/authenticated introspection with periodic
refresh, not password capacity. database_bytes records only the main file, excluding WAL.
Browser owner MFA, corrected review/cancel/lossless URL round trip, Unicode rejection,
admin reset/input clearing, new-password coreos login/UserInfo/confirmed logout passed.
Basic labels/landmarks/keyboard focus and 390px layout passed, not a WCAG audit.
Chrome extensions modify colors and produce unrelated CSP errors; none were reconfigured.

M2.2 binary SHA 6418e7e7 -> current verification binary d34dadbe (production Go code
through 9c4855e) passed enrollment/snapshot/new-directory restore, unchanged-source,
old recovery/session invalidation, actual ten-minute quarantine and recovered-owner
MFA restriction. Both schemas are 7; no arbitrary old-schema migration is claimed.
The first process exited 1 only during temporary cleanup: Python's SQLite context
manager did not close its handle. Explicit closing and post-cleanup result publication
correct that script; a focused Windows connection/open/read/close/delete check passed,
and the exact task-owned leftover was removed. Flow assertions above are reused; a
full rerun exited 0 and passed every flow assertion and temporary cleanup. Evidence is
.cache/final-upgrade-rerun.json. No product recovery bypass.

Final Go race suite currently passes 135 tests/subtests in eight packages with six
opt-in/helper skips; go vet passed. Source govulncheck found zero affected symbols/imported
packages, with the previously documented unused OpenPGP module advisory. npm audit zero.
Evidence: .cache/final-{load,race,browser-validation,upgrade}.json and final-admin.png.
C7: Vite now emits the actual chunk-component license inventory. The standard-library
Python packager binds current revision/toolchain/module versions and embedded asset
bytes to each executable. It preserves original root notices, JOSE JSON and three libc
inline supplements, Go standard-library notices, CSS/helper provenance, and a separate
build/development graph. Final packaging requires clean committed inputs; dirty preview
packaging is explicitly labeled. No source license is selected on the user's behalf.

Local dirty-build verification passed both target builds, TypeScript/Vite, official
SPDX 2.3.1 JSON-schema validation through PowerShell Test-Json, every notice/manifest
hash, ZIP integrity/explicit contents/Linux executable bits, dirty-final rejection and
missing-required-notice rejection. Windows has 30 Go dependency modules, 43 SPDX packages
and 60 notice records; Linux has 29, 42 and 59 respectively. Both include eleven frontend
components counting the CSS theme and generated Vite helper supplements. These are
artifact inventories, not a claim that whole dependency modules are linked. Clean final
artifacts and binary scans are collected after this commit; validation archives under
.cache/package-validation-531 are not final deliverables.

C7 correction under A5: `fix(distribution): label every dirty preview explicitly`
changes only preview provenance handling and this record. Independent static review
found that --allow-dirty plus a clean checkout could package an older modified binary
with a clean version and source archive. A local metadata-interposition regression
reproduced that mismatch without changing checkout/binary bytes. Explicit preview mode
now always adds .dirty and excludes the source archive; the same regression passes.
Default final mode still requires clean checkout and unmodified artifact build settings.
The first clean build was rejected because the local validation import created an
untracked Python bytecode file; only that generated file/directory was removed, then
the clean build passed. No provenance check was bypassed. A corrected clean revision
will supersede the initial 5d56cd5 local package; no artifact is published.

## Final bounded delivery evidence, 2026-10-25

A5's four product changes and available quality closure are complete for parent
acceptance. No additional feature plan is proposed. The real program remains
cmd/qoauth; internal/fixture is test-only. All keys/users/clients/state used for
validation were isolated synthetic fixtures, and every task-owned service stopped.
No real account, persistent real access, external binding, service/security change,
push, release or deployment occurred. The documented Linux/image gap is unchanged.

### Actual final checks

| Check | Executed result and evidence |
|---|---|
| `go test -count=1 -timeout 180s -race -json ./cmd/... ./internal/... ./examples/...` | 135 tests/subtests pass, six helper/opt-in skips, eight packages, zero failures; .cache/final-race.json. Process parents invoke helpers; browser/load ran separately |
| `go vet ./cmd/... ./internal/... ./examples/...` | Passed on final Go sources; packaging/docs do not alter those inputs |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 -test ./cmd/... ./internal/... ./examples/...` | Zero affected symbols/imported packages; one non-imported OpenPGP module advisory; .cache/final-govulncheck.txt |
| `npm audit --json` and `npm run build` | Zero audit findings; TypeScript and Vite pass; JS 278.09 kB, CSS 52.99 kB, unchanged asset hashes after inventory emission |
| Two RP implementations | Production SQLite OP plus coreos and Zitadel client stacks; code/S256, JWT/nonce/at_hash/UserInfo subject, refresh, negative boundaries and browser-confirmed logout pass; shared-dependency and EndSession-helper limits in integration.md |
| Browser | Owner password/TOTP, reset/MFA semantics, cleared password inputs, disabled `new` review/cancel/no-write, exact comma URL round trip, standard RP login/consent/UserInfo/logout; .cache/final-browser-validation.json and .cache/final-admin.png |
| Basic accessibility | One main/h1, labeled app fields, visible keyboard focus, 390x844 layout without horizontal overflow; not full WCAG. Existing extensions altered colors and emitted unrelated CSP errors; none changed |
| Local load | `QOAUTH_LOAD_SECONDS=600`, selected `TestModestLocalLoad` with Go JSON output and bounded timeout; 6,000 successes at 10/s, p50 1.026 ms/p95 2.855 ms/max 16.363 ms; .cache/final-load.json |
| Password cost | .cache/password-cost.go calls real PasswordHash 30 times sequentially; p50 23.264 ms/p95 26.876 ms/max 26.896 ms; .cache/final-password-cost.json |
| Compatible-version restore | `python scripts/verify-upgrade.py --previous .cache/prior-m22/qoauth-windows-amd64.exe --current .cache/final-verify.exe`; rerun exit 0, every assertion and temporary cleanup pass; .cache/final-upgrade-rerun.json |
| Clean final build | `./scripts/build.ps1 -UseInstalledFrontend` at clean 826200d; Windows/Linux amd64 CGO=0, embedded assets, symbol tables retained; .cache/final-build.txt |
| Final binary scans | govulncheck v1.9.0 `-mode=binary` against both actual final dist binaries; zero affected symbols/imported packages, same unused-module advisory; .cache/final-windows-amd64.exe-vuln.txt and .cache/final-linux-amd64-vuln.txt |
| Final Windows binary | .cache/final-binary-smoke.py passed init, embedded login/discovery/readiness, restart/JWKS, busy-lock backup denial, consistent snapshot, RSA rotation/AES continuity, fresh restore/owner recovery/quarantine, count-only cleanup and secret-free output; temporary directories/processes removed |
| Artifact checks | Both SPDX documents pass official 2.3.1 JSON schema with Test-Json; all notice/manifest/archive hashes and ZIP integrity checked, Linux ZIP executable bit preserved, no runtime fixture executable or private state included |
| Packaging failures | Default dirty checkout and missing required notice rejected; modified-binary/clean-checkout metadata regression failed before and passed after 826200d; explicit preview always .dirty without source ZIP |
| Container configuration | `docker compose config --quiet` passed; no daemon required, no image build/run |
| Documentation | Eleven tracked Markdown documents checked for UTF-8/local links/anchors and unresolved markers; git diff --check passed; current evidence supersedes historical stage status |

The load host was Windows amd64, i9-12900H (20 logical CPUs), 31.7 GiB memory.
One request was in flight; the mix was readiness/discovery/authenticated introspection
with periodic refresh, not concurrent password login. Main-file database_bytes omits
WAL; memory growth and 2-vCPU/2-GiB capacity were not established. Four Argon2 slots
bound hashing; measured sequential cost is not a throughput guarantee.

The restore drill used earlier M2.2 Windows artifact SHA
6418e7e7516c912494165e52e42083abb438f469d8a095f20b646507a740ff1b
and current-code verification artifact d34dadbe1bbf4160d924c55c989cf81a14c310a94b18482c9ddec9690b5cc5ce.
Both schemas were 7. It exercised a real ten-minute quarantine, retained source bytes,
disabled restored clients, invalid old recovery/session state and restricted recovered
owner. Production Go code is unchanged between that verification build and final
artifacts; final binary maintenance smoke separately confirms packaging/runtime wiring.
There is no claim of arbitrary historical-schema migration or destructive downgrade.

Independent final product review found two P2 UI cases, fixed in af185b8 and verified
in the browser. Independent packaging review found one P2 preview-label case, safely
reproduced and fixed in 826200d. Reviews were static; actual runs above belong to the
primary agent. No unresolved reported blocker remains in these bounded findings.

### Versioned local artifacts

Final code/build revision: **826200d48959bef3e23f554fc39f73ea1635a149**.
Version: **0.1.0-dev.826200d48959**. Source and binary provenance use this clean
revision; this later documentation commit does not relabel them. Earlier 5d56cd5
packages and .cache dirty archives are superseded validation artifacts, not this handoff.

| Artifact under dist/0.1.0-dev.826200d48959 | SHA-256 |
|---|---|
| qoauth-0.1.0-dev.826200d48959-windows-amd64.zip | efe854a9e7458142d4fe360f78f710a955a73631953d8ad96a66b1ed6122e57a |
| qoauth-0.1.0-dev.826200d48959-linux-amd64.zip | a151b5f391d62870ab96dc74dc360a66c04e570b50e3064089f269a10f080ee5 |
| qoauth-0.1.0-dev.826200d48959-source.zip | ecc08bfd13f11544594960a2b8a458675d1ff832d03ce9ec42c87daa391477a5 |
| windows-amd64/qoauth-windows-amd64.exe | f7c3793d6aaab7b8bc677a21c8847d14113374eb43b9dbc5fdc126aa3f94e5e3 |
| linux-amd64/qoauth-linux-amd64 | 5da3b258fab86e6328ee40c8c89e788cd64c027968ae818e844cae13d3a151d3 |

The [root manifest](../../dist/0.1.0-dev.826200d48959/manifest.json) records archive
sizes and hashes. Each platform directory/ZIP has executable, sbom.spdx.json,
THIRD_PARTY_NOTICES.txt, LICENSES, notice-index.json, build-info.json,
frontend-inventory.json, manifest.json and RELEASE-GATES.txt. Windows/Linux record
30/29 actual Go dependency modules, 43/42 SPDX packages and 60/59 notice records;
both include eleven frontend/helper/CSS components. build-development/inventory.json
separately records 273 npm entries (248 installed) and 66 resolved Go modules.
The exact source ZIP contains source/tests, Docker/Compose recipes, docs and the RP
example, not node_modules, cache, live databases or credentials. No image exists.

### Finite remaining gates and handoff

- Native Linux execution and Docker image build/run/non-root persistence/maintenance
  remain unverified. Existing Ubuntu WSL and Docker Desktop are installed, but no
  distribution was running and docker info failed on the absent Linux-engine pipe.
  Concrete future actions would be starting Ubuntu WSL or Docker Desktop/Linux
  engine; those host runtime/service actions were not authorized or executed.
- Actual image provenance, CA-certificate notices and image SBOM require the image.
  QoAuth's own source-license choice remains unresolved; NOASSERTION/CC0 metadata is
  not a product license. No publication is authorized or performed.
- Real host/proxy/account/key setup is outside synthetic tests. Full OIDF testing,
  extensive fuzzing, broad accessibility/security audit, small-host capacity and
  physical power-loss durability are not claimed. The four-entry local password
  blocklist needs real-exposure assessment; no breached-password corpus is implied.
- The approved product slice needs no new enterprise feature or infrastructure.
  Parent acceptance is the next decision; no internal plan is delegated back to an
  end-user approval wait.

A5 logical commits: f0e2227 authorization, d50d81d policy, 6f4499e reset UI,
54e34a5 disabled-client review, b240fc3 readiness, 9c4855e second RP, af185b8 lossless
review correction, 5312400 quality harnesses, 5d56cd5 packaging, 826200d preview
provenance correction, and the current docs(validation): close bounded delivery
checks evidence commit. Main only; no branch/worktree, push, release or deployment.

## Platform validation clarification and execution (A6)

The actual parent message beginning "已审核当前交付，功能范围收口，不再加功能"
accepts the product boundary and authorizes ordinary startup of installed Docker
Desktop or existing Ubuntu WSL for local Linux/image building and isolated tests.
It explicitly distinguishes that from installing components, enabling Windows
features, elevation, changing network/security settings or service startup types.
Those additional changes remain excluded, as do real credentials, image pushes,
deployment and cleanup of unrelated images/containers/WSL data. Existing main-only
logical commit instructions persist. The source license stays undecided.

This supersedes the earlier blanket startup-permission interpretation. Inspection
found clean main 6d9f87f; about 8.9 GiB free host RAM and an active Python process
using about one CPU core. Work will run serially with low parallelism and bounded
resources, not compete with research verification. Ubuntu started normally as its
existing uid 1000 user through wsl.exe; its existing Docker engine was available,
with no running containers and only an existing hello-world image. Docker Desktop
need not be started. No sudo, installation or host configuration change occurred.

| Task | Bounded scope / checks | Intended commit | State |
|---|---|---|---|
| P0 | Record A6, actual resource/runtime inspection and this checklist | docs(plan): record authorized platform validation | Committed 9094fa6 |
| P1 | Execute Linux-native tests and final binary smoke on private ext4 temporary state; only necessary portability-test corrections | test(platform): validate isolated Linux execution | Committed 2fcac4b |
| P2 | Build existing image with CPU/memory limits; non-root/read-only/loopback TLS synthetic lifecycle, restart and maintenance; necessary recipe fixes only | test(container): verify isolated image lifecycle | Done; this record's commit |
| P3 | Actual image provenance/notices/SBOM and updated finite evidence; retain all unexecuted limits | docs(validation): record Linux and container evidence | Pending |

No additional product feature is authorized. Existing Windows race/browser/security
evidence is reusable for unchanged inputs; native checks must actually run before
changing platform claims. Test resources get task-specific names and only those
resources may be removed. Never prune or terminate unrelated host/container/WSL work.

P1 executed all eight cross-compiled CGO=0 Linux test binaries sequentially on
Ubuntu WSL2 kernel 6.18.54.1 as existing uid 1000, with GOMAXPROCS=2/nice=10 and
private native /tmp state. Results: 130 tests/subtests passed, six helper/opt-in
skips, zero failures. Unix ownership/mode, hard-link, flock/process-exit, SQLite,
OP/RP and maintenance tests needed no portability changes. The actual final Linux
binary SHA 5da3b258fab86e6328ee40c8c89e788cd64c027968ae818e844cae13d3a151d3
passed private modes, UI/discovery/readiness, real CLI SIGTERM, restart/JWKS,
locked backup, rotation/AES continuity, new-directory restore/recovery/quarantine,
cleanup bounds and redacted output. scripts/verify-linux.py reproduces the run;
.cache/platform-a6/linux-results-r1/report.json and package logs hold evidence.
All temporary state/processes were removed. This is native Linux execution on WSL,
not Linux race detection, physical power-loss or a real foreign-owned-inode test.
The earlier Windows race evidence remains separate. Host CPU afterward was 2%
and free RAM about 9.4 GiB; no research task was stopped or reconfigured.

P2 built the existing Dockerfile natively with Docker 29.9.2 in Ubuntu using
--resource memory=1536m --resource cpu-period=100000 --resource cpu-quota=100000.
Build metadata confirms these limits. No Dockerfile or product source correction
was required. scripts/verify-container.py uses one immutable local image, unique
labeled containers/volume, uid/gid 65532, read-only rootfs, dropped capabilities,
no-new-privileges, 1 CPU/512 MiB runtime limits, and offline maintenance containers.
Only serve publishes a random 127.0.0.1 backend port; a temporary local HTTPS proxy
uses a synthetic certificate trusted only by the test clients. The actual proxy
peer 172.17.0.1 was confirmed by missing-X-Real-IP rejection, not simply assumed.

The final harness run passed volume ownership, one-time init, secure cookie/Host,
owner MFA, the real coreos Go RP code/PKCE/JWT/UserInfo flow, restart/JWKS/session
persistence, confirmed logout, SIGTERM, lock exclusion, backup, signing rotation/AES
continuity, restricted restore/recovery/quarantine and cleanup. Evidence:
.cache/platform-a6/container-results-r3/report.json. Its first two runs exposed
harness assumptions (relative redirect URL and transient RP startup 503), corrected
before the passing rerun; no product failure was concealed. All synthetic volume,
proxy, RP and test-container resources were removed. Full quarantine expiry remains
the earlier real ten-minute drill; this run verifies startup denial, not clock bypass.

The initial validation image's explicit revision label was stale (6d9f87f), while
BuildKit correctly recorded source context 2fcac4b. It is not the final handoff image.
A clean post-harness build with the exact current revision label and retained
BuildKit provenance will replace it before export, with final-image checks recorded
below. Base image digests and actual CA/binary evidence are being collected locally.
