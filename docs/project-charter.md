# Project charter and authorization history

Recorded: 2026-10-25 (Asia/Singapore). Product name: QoAuth.

## Current authorization A5

The actual parent message beginning "已审阅 M2.2 证据和剩余工作，批准下一阶段限定收口"
authorized administrator password reset UI, disabled-application review/prefill,
consistent existing password policy and minimal readiness, followed by available
second-RP, load/accessibility, compatible-version recovery and license/SBOM checks.
It permits isolated synthetic tests and logical main commits, including necessary
in-scope review corrections. It does not permit enterprise expansion, real accounts,
system service/security changes, push, release or deployment. This is the parent's
actual decision, not invented original end-user wording or a new approval wait.

The bounded work is complete for parent acceptance. Final binaries are tied to
826200d; later evidence-only documentation does not change their provenance. Native
Linux/image verification and the source-license decision remain explicitly open.
No further feature phase is automatically authorized or scheduled.

## Historical authorization A4

The parent message beginning "已审阅 M2.1 交付及 M2.2 计划，批准这个最小运维闭环的代码实现与隔离合成测试"
authorized minimal local maintenance and logical main commits from 15e2f42. It
accepts protected plaintext keys.json, stopped locked consistent DB+key backup,
new-directory restore/reconciliation and old-token invalidation, bounded signing
rotation without AES changes, and dependency-aware cleanup. Implementation and
correction evidence are in the single execution checklist. This records the actual
parent decision, not a quote attributed to the original end user.

All tests/commands use isolated temporary synthetic credentials and databases.
No real installation, persistent real access, system permission/network change,
deployment, push or release was authorized. M2.2 was subsequently accepted by A5;
its bounded follow-up is recorded above.

## Historical authorization A3

The parent message beginning "我已审阅 M1 最终证据和 M2.1 三部分计划，批准在原授权范围内实施 M2.1"
authorized the bounded core, verification and logical main commits from 0a77f7b.
Code may implement durable keys and owner initialization, but validation uses only
temporary isolated synthetic state. This is the actual parent decision, not invented
end-user wording. Real credentials, deployment, system services and external bindings
remain outside scope. The earlier research-only stop below is historical.

## Historical authorization A2

Parent internal review on 2026-10-25 accepted the lightweight direction and lifted
the M0 stop for M1 local feasibility implementation, tests and logical local commits.
The message beginning "Parent 已阅读总体计划全文及11项待决策和模块清单" is the actual
authorization source; it is not attributed retrospectively to the original end user.
Use one concise execution checklist. No additional user approval of internal plans
is required. The historical M0-only limits below are superseded only for this M1
scope; protected real-credential, external, deployment and system boundaries remain.

## Current scope correction A1

The user's later instruction on 2026-10-25 supersedes enterprise-oriented design
assumptions: personal/small-team self-hosting, lightweight Go monolith, embedded UI,
SQLite first, and the same Windows/Linux/Docker targets. Do not default to complex
organizations, multi-tenancy, SCIM/LDAP/SAML, PostgreSQL/Redis, queues, microservices
or Kubernetes. Keep core login/users/clients, safe OIDC/OAuth, scopes, sessions and
revocation, necessary MFA/recovery, simple UI/logs and minimal operations. Security
and standards tests remain required. Enterprise functions may appear in competitor
research only, not in the delivery commitment. The revised product and roadmap
documents are authoritative; the earlier broad proposal remains only in Git history.

The exact steering includes: “不是为了做那种企业，或者是非常非常复杂的业务流程”
and “千万不要过度工程化”. It authorizes in-scope document corrections under A0,
not business implementation. Deliver the revised proposal for review first.

## Original requested outcome (constrained by A1 above)

Research contemporary open-source and proprietary IAM, SSO, OAuth, and OIDC
systems using official sources. Produce a reviewable overall product and technical
plan for a full-featured, secure, easy-to-integrate identity platform. Define
“full-featured” in explicit tiers with testable acceptance criteria; never equate a
feature list, a library choice, or a standards claim with production readiness.

Authentication establishes who a principal is. OAuth authorization delegates
limited access to resources. OIDC adds interoperable identity authentication on
top of OAuth. A directory stores identities and memberships; application business
authorization remains the application's responsibility unless explicitly delegated
to a future QoAuth policy service. These boundaries must be reflected in the design.

## User-confirmed constraints

- Repository: `D:\QianFuv\Go\QoAuth`, already registered as a project in T3 Code.
- Go backend; frontend static assets embedded in the executable.
- Windows amd64 and Linux amd64 executables; Linux amd64 Docker image.
- Work directly on `main`; one logical change per commit. No development branch
  or worktree, remote push, release, or deployment.
- Original M0 stop: documentation/research only; subsequently superseded by A2/A3/A4/A5 within their bounded implementation scopes.
- Compare Keycloak, authentik, ZITADEL, Ory, Casdoor, Authelia, Auth0, Okta,
  Microsoft Entra, and Clerk; select representative systems for deeper analysis.
- Cover product scenarios, feature tiers, architecture, storage, deployment,
  integration/SDKs, threat model, security verification, dependencies/licenses,
  milestones, and risks. Prefer mature protocol and cryptography libraries.
- Other local Go projects may be inspected read-only for packaging conventions.
  Their code, configuration, dependencies, and environments must remain unchanged.

## Authorization record A0

The current user instruction explicitly authorizes read-only research, local Git
initialization, and incremental research/design document commits in this repository:
“只读调研和本仓库本地初始化/文档提交已授权”. It also explicitly requests `main`
and no repeated internal-design questions before handing back the concrete proposal.

This is a current-task override of the supplied AGENTS.md defaults requiring
separate exact-message commit consent for ad-hoc work. It covers only phase-one
documentation and the requested local initialization, with messages and scopes
reported in the execution record. It is not approval of the resulting architecture,
business implementation, future code commits, or any external operation.

The overall plan is a requested research/design deliverable and is intentionally
tracked with the other documents under this explicit document-commit instruction.
No ignore or Git configuration change is needed to enable automatic Git Flow.
Automatic plan-approved implementation Git Flow is not activated in this stage.

## Protected boundaries

Do not register trials, use real customer credentials, create persistent real
credentials or authorization bindings, change firewall/public-access/account
permissions, upload private files, permanently delete user data, alter other
projects, or change research/training or system environments. A later stage needing
one of these actions must present the exact action, reason, data scope, and rollback
before obtaining specific authorization. Internal design proposals need review,
not speculative permission prompts during research.

## Initial inspection

At initial inspection on 2026-10-25 the project directory was empty, including
hidden entries. `git rev-parse --show-toplevel` reported that no Git repository
existed. No `AGENTS.md` or `.agents` directory was found at the repository or its
ancestors `D:\`, `D:\QianFuv`, and `D:\QianFuv\Go`. There were no pre-existing user
files or staged changes to preserve. The policy pasted by the user in this thread
therefore governs; no repository policy file has been invented or copied in.

Git 2.56.0.windows.3 and Python 3.12.15 were already available. Existing default
Git identity was present; it was not changed. No dependency installation is needed
for this documentation stage.

## Historical M0 evidence and acceptance

Record sources and verification date, distinguish vendor claims from executed
tests, and distinguish open source from free plans, enterprise features, and
hosted-only services. Architecture decisions are recommendations pending user
review. Documentation validation, source inspection, and Git checks can pass in
this phase; protocol conformance, application security, cross-platform builds,
runtime integration, performance, and restore tests cannot pass before they run.

Acceptance is human review of the delivered proposal. Completion of research and
local commits is not user acceptance of the design or permission to implement it.
