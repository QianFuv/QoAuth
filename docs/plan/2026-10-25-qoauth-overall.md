# QoAuth lightweight plan and M1 execution checklist

Protocol Version: 2026-09-05
Protocol Revision: autonomous-execution-git-flow-v2

Revision 3, 2026-10-25. Status: In Progress. Parent internal review accepted the
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
| T2 | go.mod/go.sum, web, asset handler, cmd, ignore/build/container files | V2; V6 consolidated in T5 | feat(scaffold): embed the lightweight application shell | Done; commit follows |
| T3 | internal/store and SQL migrations | V3 atomic SQLite transitions | feat(store): enforce atomic sqlite credential transitions | Pending |
| T4 | Test-only fixture OP adapter and independent Go RP integration | V3, V4 actual standard endpoints | feat(oidc): validate the standard fixture flow with sqlite | Pending |
| T5 | Evidence and bounded support corrections | V1-V6 applicable checks | docs(validation): record M1 evidence and remaining gates | Pending |

Record a necessary separate logical correction's scope/message before committing
under A2. No unrelated changes or history rewriting. M2 remains a later parent decision.

## Verification and acceptance

- V1: `python .git/validate-docs.py`, `git diff --check`, source/diff review.
- V2: `npm ci` and `npm run build` in web; embedded route tests: known UI routes
  work, unknown API/protocol paths stay 404, no runtime Node/assets directory.
- V3: `go test ./...`, `go vet ./...`, gofmt. SQLite version/settings checks,
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
  or real credential created. Current task: T1.

- T2: Locked frontend build and current embedded route tests passed; npm audit reported
  zero findings. Go test passed (two packages). V6 cross-build/config evidence is
  consolidated in T5 so a cold build does not block independent storage work.
