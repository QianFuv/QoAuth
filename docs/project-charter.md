# Project charter and phase-one authority

Recorded: 2026-10-25 (Asia/Singapore). Product name: QoAuth.

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
- This stage is documentation and research only. Do not implement business code.
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

## Evidence and acceptance

Record sources and verification date, distinguish vendor claims from executed
tests, and distinguish open source from free plans, enterprise features, and
hosted-only services. Architecture decisions are recommendations pending user
review. Documentation validation, source inspection, and Git checks can pass in
this phase; protocol conformance, application security, cross-platform builds,
runtime integration, performance, and restore tests cannot pass before they run.

Acceptance is human review of the delivered proposal. Completion of research and
local commits is not user acceptance of the design or permission to implement it.
