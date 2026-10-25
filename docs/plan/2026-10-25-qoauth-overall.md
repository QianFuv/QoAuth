# QoAuth lightweight plan and M1 execution checklist

Protocol Version: 2026-09-05
Protocol Revision: autonomous-execution-git-flow-v2

Revision 4, 2026-10-25. Status: M1 accepted; M2.1 authorized and in progress.
Parent internal review accepted the
lightweight direction and explicitly authorized M1 local feasibility work (A2).
There is no outstanding end-user approval gate for this scope. This is the single
execution checklist; do not create another plan for each small step.

## Outcome and boundaries

One Go process, one SQLite database, one issuer, embedded React/TypeScript/Vite/Semi
UI. Build Windows/Linux amd64 binaries and validate Linux amd64 container config.
Use standard OIDC code with S256 PKCE, signed JWT ID Tokens, opaque access tokens
and authenticated introspection. A genuine standard Go RP must exercise the actual
OP handler and SQLite adapter; a private protocol cannot substitute for OIDC.

M1 uses synthetic fixtures and ephemeral test keys only. No production authentication
service, real user directory, durable real credentials, SMTP, external OAuth bindings,
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

## Single execution checklist

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
under A2. No unrelated changes or history rewriting. M2 remains a later parent decision.

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

## Proposed next slice for parent decision: M2.1 local authentication core

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
| U1 | This acceptance/execution record | docs(plan): record M2.1 implementation approval | Done; commit follows |
| U2 | Password/TOTP/key primitives, SQLite identity/session/client transitions, dependency locks and tests | feat(identity): persist credentials and enforce atomic session security | Pending |
| U3 | Production config/CLI/OP, account and management HTTP APIs, Go RP example and E2E/failure tests | feat(auth): connect local accounts to standard OIDC sessions | Pending |
| U4 | Embedded Semi login/MFA/account/admin screens and browser verification | feat(ui): add local authentication and management screens | Pending |
| U5 | Final security/build evidence, deployment limits and next bounded scope | docs(validation): record M2.1 authentication evidence | Pending |

Mandatory checks extend V1-V6: secure cookie attributes and origin/CSRF binding;
TOTP window replay and recovery-code contention; disable/password/factor recovery
invalidate sessions, pending codes and grants; exact redirects and authorized
audiences/scopes; persisted signing keys survive restart; credential canaries do
not appear in logs. Run actual production handlers with standard Go RP E2E,
negative tests, race, dependency scans and browser verification. Native Linux and
container execution remain separately unverified unless a permitted runtime exists.
No additional approval gate or enterprise framework is introduced.
