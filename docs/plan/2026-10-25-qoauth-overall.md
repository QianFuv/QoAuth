# QoAuth lightweight overall review plan

Protocol Version: 2026-09-05
Protocol Revision: autonomous-execution-git-flow-v2

This is the requested overall design proposal and phase-one handoff record. It is
ready for human consideration, but deliberately **not an executable development
contract**. Review the product direction first; a later bounded task contract will
specify actual code changes, dependencies, commands and commit rows. No business
implementation starts from this document alone.

## Metadata

| Field | Value |
|---|---|
| Task ID | 2026-10-25-qoauth-overall |
| Status | Draft |
| Plan Revision | 2; lightweight scope replaces the initial broad proposal |
| Plan Approval | Pending |
| Approved Plan Revision / Plan Approved At | Pending |
| Plan Approval Reference | None; A0/A1 authorize research/documents, not this resulting design |
| Priority / Type / Area | P1 / docs / QoAuth product and technical design |
| Owner | User reviews; primary agent prepares and verifies |
| Created / Updated | 2026-10-25, Asia/Singapore |
| Branch / Git Repository | main / Present |
| Git Working Tree | Clean at pre-handoff baseline |
| Git Base Commit | a64bd1d103c8af1201625d9750c7d17fb32df45a |
| Git Base Status | No staged, unstaged or untracked work before this handoff document and README update |
| Git Flow Enabled / Git Flow Consent | No / Disabled; no implementation tasks in this review-only contract |
| Pre-Plan Cleanup Required / Decision | No / Not Required |
| Acceptance Mode / Status / Reference | Human Review / Pending / None |
| Related Links | [Product](../product.md), [roadmap](../roadmap.md), [architecture](../architecture.md), [charter](../project-charter.md) |

## Current Execution State

| Field | Current value |
|---|---|
| Approved contract | A0 research/local docs; A1 lightweight revision only |
| Current task | Review handoff; no business task selected |
| Last completed work | Competitor research, revised scope, architecture/integration, security plan |
| Last verified state | Six confirmed local commits through the baseline above; expanded handoff scan passed on 11 Markdown files |
| Pending commit gate | No future implementation gate opened; containing handoff commit is covered by A0/A1 and identified in Git/final response |
| Pending research operation | None; three research delegates and one design-critique delegate completed |
| Active blocker | None for phase-one delivery; design acceptance remains pending |
| Next action | Deliver the committed proposal for user review; do not start M1 |
| Resume references | This file; charter A0/A1; roadmap; actual git status/log |

## Workflow Resources

Both pinned network resources were fetched directly and validated with
SHA256-UTF8-LF-v1 before using their structure/status definitions. No local substitute
or persistent cache supplied their authority.

| Resource | Exact accepted source | Protocol / verified digest |
|---|---|---|
| AGENTS.md | User-pasted repository policy in this thread; no physical policy file found | 2026-09-05 / autonomous-execution-git-flow-v2; source identity is the supplied user message |
| PLAN_TEMPLATE.md | [Pinned raw revision](https://gist.githubusercontent.com/QianFuv/08d3789effa14cdcf1c23b5a7abe0ea7/raw/d69c065ce9a1aea57f207f62dadc71a4ddbea914/PLAN_TEMPLATE.md) | Same protocol; 1ddfdce0058f9187b529634b09921385b6690fb02705cba39b0eb2eddbe03a59 |
| PLAN_EXECUTION.md | [Pinned raw revision](https://gist.githubusercontent.com/QianFuv/63b4c5707ad011baaa80880e93fb310d/raw/3afc10647951bd36b51809197fdc9966511cccd7/PLAN_EXECUTION.md) | Same protocol; a2e0029e22cda2a37c0e4e9db06b57430ef5a48e3e13e38adac25f626b951d54 |

## Summary and context

Build a lightweight self-hosted identity service for personal/small-team apps:
Go monolith, embedded frontend, SQLite first, Windows/Linux amd64 and Linux amd64
Docker. Keep safe OIDC/OAuth, local login/users/clients, scopes, session revoke,
TOTP/recovery, clear UI/logs and small operating instructions. OAuth authorization,
OIDC authentication and application business permissions remain separate.

The directory began empty and without Git. No local/ancestor AGENTS.md or .agents
skills were found; default Git identity was already configured. Git was initialized
on main under explicit user authority. Existing Go projects were read only for
packaging patterns. No user data needed cleanup, no credentials were created,
no system dependency was installed, and no remote/push/release/deployment occurred.

## Git readiness and preservation

| Check | Observed result / decision |
|---|---|
| Repository / branch / full HEAD | Present / main / baseline in Metadata; actual commands succeeded |
| Index and working tree | Clean before handoff authoring; all prior changes attributable to this task |
| Pre-existing changes | None; initial directory was empty including hidden entries |
| Cleanup | N/A; no reset, restore, stash, cleanup commit or user-file deletion |
| Plan storage | Tracked as an explicitly requested research/design deliverable under A0; no ignore/config change |
| Automatic Git Flow | Not activated for future code; current local document commits use explicit A0/A1 task authorization |

## Scope

Deliver the comparison/source ledger, product boundary, short roadmap, architecture,
dependency/license review, integration example, threat/test plan and this review
entry point. These are documentation deliverables only.

Required functionality is listed in [product.md](../product.md). Passkeys, upstream
OIDC/social login, machine credentials, back-channel logout and device flow are
optional demand-driven additions. Complex organizations/tenancy, enterprise
SCIM/LDAP/SAML, approval flows, PostgreSQL/Redis prerequisites, microservices,
queues, Kubernetes and HA are outside the current commitment.

## Decisions, assumptions and review points

| Item | Recommendation | Review status |
|---|---|---|
| Target | Personal/small-team self-hosting, not enterprise parity | Confirmed by A1 |
| Storage | Local SQLite and one instance; modernc candidate | Proposed; F2/F4 must validate |
| Protocol | Single issuer, code+S256, opaque access tokens/introspection; existing OP library | Proposed; library advisory and transaction gates remain |
| UI | React/TypeScript/Vite/Semi, embedded at build time | Proposed; coherent versions/CSP checked in M1 |
| Recovery | Admin TOTP, single-use codes, no MFA bypass through password reset; local-control sole-admin procedure | Proposed; explicit negative tests before real use |
| Logout | Revoke QoAuth sessions/grants; no universal RP-cookie erasure promise | Proposed documented contract |
| Distribution license | Consider Apache-2.0 for original code; not selected yet | Deferred before distribution; no LICENSE file created |
| Next stage | M1 feasibility only after review and bounded implementation contract | Pending; no automatic start |

## Plan and file map

| File | Purpose |
|---|---|
| [README](../../README.md) | Short navigation and actual project status |
| [Charter](../project-charter.md) | User constraints, A0/A1 and initial inspection |
| [Comparison](../research/comparison.md), [sources](../research/sources.md) | Ten-product matrices, representative depth, date/edition/license evidence |
| [Product](../product.md), [roadmap](../roadmap.md) | Required/optional/excluded scope; M1-M3 and measurable F/A/R checks |
| [Architecture](../architecture.md) | Process/data/protocol design, SQLite atomicity, deployment, backup/rotation |
| [Dependencies](../dependencies.md) | OP/crypto/storage/UI shortlist, actual advisories, licenses and acceptance gates |
| [Integration](../integration.md) | Design-only registration and library-based login/revoke/logout walkthrough |
| [Security](../security.md) | Twelve targeted threats, verification strategy and small risk register |
| This plan | Review contract, authority and handoff record |

Interfaces/data flow: browser or RP → HTTPS Go handlers → identity/client/session
logic → local SQLite and protected keys; embedded UI calls the same authorized
admin API. No external database or messaging dependency is planned.

Key edge cases: cross-client redemption, code/refresh/recovery replay, lost refresh
response, provider logout with surviving app cookie, full/locked DB, old backup
restoring revoked state, missing keys and administrator MFA bootstrap. Acceptance
maps to F2, A2-A7, R3-R4 and S01-S12 in the linked documents.

Parallel read-only work used Astra with inherited xhigh settings for OSS products,
commercial products, standards/dependencies, and a later lightweight-design
critique. Delegates did not edit repository files or grant approval. Parent review
validated material findings; a delegate assertion is not a runtime test pass.

## Implementation Task Order, Git Flow and Tasks

There are **no executable business tasks or future commit rows in this review
contract**. M1 → M2 → M3 in roadmap.md is the proposed order, not implementation
authorization. After review, instantiate the selected bounded task contract with
exact source scope, dependency/environment needs, tests and per-change commit rows.
Do not mark those milestones In Progress now.

| Task | Commit Required | Commit Status | Message / scope / consent |
|---|---|---|---|
| None; business execution deferred | No | Not Required | N/A; no automatic implementation consent |

This avoids a misleading one-commit-per-milestone promise: later code commits must
still each contain one logical change. The initial docs-only task's actual commits
are historical facts below, not future permissions.

## Verification Plan and Results

| Check | Requirement / actual evidence | Reuse rule |
|---|---|---|
| D1 | Required: official-source comparison and consequential license/edition/advisory cross-check; completed for the 2026-10-25 snapshot | Recheck when source/version/entitlement affects implementation |
| D2 | Required: review every current design file against A1; enterprise prerequisites removed; logout/backup/MFA critique incorporated | Rerun if scope changes |
| D3 | Required: Python UTF-8/local-link/anchor/fence/conflict-placeholder scan; 10-file baseline and expanded 11-file handoff candidate passed; git diff --check passed | Final changed document inputs need a fresh pass |
| D4 | Required: git status/log/ls-files/remote checks; main, six commits and clean baseline verified; final handoff checked again after commit | Always fresh at final reporting |
| Product runtime checks | Not Applicable in M0: no app, build, credentials, DB or service exists; F1-F4/A1-A8/R1-R7 are future obligations | Never report a future test as passed |

Local verification uses existing Python/Git only. Helper scripts and sanitized
commit-attempt fingerprints live inside .git for this session, not as application
source. Checks cover staged/unstaged/untracked task documents, not merely HEAD.
No runtime/security exception has been accepted or required check bypassed.

## Approval Log

| ID | Provenance | Actual decision and boundary |
|---|---|---|
| A0 | Current user's original phase-one task; recorded 2026-10-25, original message timestamp not separately available | Research, local Git initialization and incremental document commits explicitly authorized; no business implementation or external actions |
| A1 | Later user scope correction; recorded 2026-10-25 | Revise immediately for lightweight personal/small-team use, SQLite first; remove enterprise roadmap; deliver proposal for review |

Neither entry accepts the resulting design or authorizes M1. Human acceptance is
pending; no silence, generated plan field or internal critique is treated as approval.

## Progress, commit attempts and review feedback

| Commit | Actual logical change |
|---|---|
| d86967ad2b7f8f8cf4fadd8b30c76bfcea20088f | Establish research scope and README |
| 6a8bd77f76defa33bdd5aa024d875453fcba5581 | Initial product/acceptance proposal; later superseded in scope |
| b7861e971c756894bf126711288c312ab2b2948a | Apply A1 lightweight correction; preserve earlier history |
| fbaeda723d74a200861ec04e597e2fdd659c6035 | Current platform/license research and source ledger |
| c9a6d6c678f3d67b586d3fac22e52e18d3d9a14b | SQLite monolith, dependency selection and integration design |
| a64bd1d103c8af1201625d9750c7d17fb32df45a | Focused threat/recovery tests and risks |

Each commit used explicit paths, inspected content, default identity, normal hooks
and a single-line Conventional Commit message. Pre-attempt parent/message/scope/
staged fingerprints are in .git/qoauth-commit-attempts.jsonl; actual Git history is
authoritative. The handoff commit adds this plan and updates README; its confirmed
hash is reported after creation rather than fabricated inside its own content.

Review feedback: A1 was a material scope correction and is implemented throughout
the current design. Independent critique identified logout propagation limits,
atomic refresh-family revocation, SQLite durability/version checks, restore rollback
and admin MFA bootstrap; those were incorporated. No internal review equals user
acceptance. No business implementation notes exist.

## Final Outcome

Phase-one research and lightweight design artifacts are delivered for review.
The handoff's final documentation/Git checks and commit identity are reported in
the accompanying final response. Current design acceptance remains Pending.
No application, binary, image, real credential, network deployment or production
security claim has been created. Next decision: accept or revise the lightweight
scope, SQLite/library direction and M1 boundary before a separate execution contract.
