# Milestones and testable acceptance contract

Review status: proposed roadmap, 2026-10-25. Only M0 is authorized now. M1-M5
require a subsequent task contract approved by the user; this document is not
automatic implementation or commit consent.

## Milestones

Estimates are engineering effort for one experienced Go/full-stack engineer with
security review support, not calendar promises. Parallel work, upstream defects,
connector scope, and review availability can change them materially.

| Milestone | Estimate | Deliverables | Exit evidence |
|---|---|---|---|
| M0 Research and review | Current stage | Source-backed comparisons, architecture, risk/security plan, integration design, local documentation commits | D1-D4; user review pending |
| M1 Feasibility and contract | 2-3 engineer-weeks | OIDC library/storage spike, tenant/issuer contract, embedded-UI/CSP spike, exact dependency and license inventory, supported profiles | F1-F5; go/no-go review |
| M2 Secure core alpha | 6-10 engineer-weeks | Single-tenant users, passwords/TOTP/recovery, OIDC, clients/consent, sessions/revocation, admin/API, audit, packaging | A1-A12; no production label |
| M3 Team/B2B beta | 4-7 engineer-weeks | Tenant isolation, organizations, passkeys, upstream OIDC, delegated admin, integration examples | B1-B7; scoped pilot review |
| M4 Enterprise interoperability | 5-9 engineer-weeks | SAML SP/IdP, inbound SCIM, LDAP/AD federation; optional protocols selected individually | C1-C6; compatibility matrix |
| M5 Operational hardening and release candidate | 4-7 engineer-weeks plus independent review | HA/fault tests, recovery drills, security review, performance baseline, SBOM/licenses, signed artifact process | R1-R9; production approval remains explicit |

The full target is roughly 21-36 engineer-weeks after M0 before accounting for
external review delays and optional advanced protocols. M2 alone is not a secure
production release. Apply security and operational testing throughout, rather than
postponing it until M5. Release candidate validation does not authorize publication.

## M0 documentation checks

| ID | Acceptance condition | Verification |
|---|---|---|
| D1 | All ten named candidates covered; representative depth and edition/license caveats | Read the comparison against official sources and source ledger |
| D2 | Every required area has a proposed behavior and milestone; no implemented/secure claim | Coverage review across product, architecture, security, integration, and this file |
| D3 | Documents are readable UTF-8, local links resolve, no template placeholders or conflict markers | Local Python document scan and `git diff --check` |
| D4 | Work remains docs-only on main with separate logical commits; no remotes or residual changes | `git log`, `git ls-files`, `git remote -v`, `git status --porcelain=v1` |

## M1 feasibility gates

| ID | Observable pass condition | Why it can stop the design |
|---|---|---|
| F1 | Selected Go OP library completes code+PKCE, discovery, UserInfo, revocation/introspection and logout profile in an isolated synthetic harness | A library API checklist is insufficient; failure may require switching libraries or reducing the profile |
| F2 | One-use code and refresh rotation remain atomic under 50 concurrent redemption attempts and injected transaction failures | Duplicate valid grants or partially committed state blocks M2 |
| F3 | Same frozen frontend bundle embeds into Windows/Linux amd64 binaries; no runtime Node or web asset directory needed; API 404s never return SPA HTML | Proves the packaging constraint without building business features |
| F4 | Candidate UI works with intended CSP, keyboard navigation, and local assets; dependency graph licenses and advisories recorded at pinned versions | UI compatibility, licensing, or unsupported libraries require a recorded alternative |
| F5 | Tenant issuer, subject, cookie, account-linking, storage, recovery, and authorization decisions have approved ADRs; conformance harness can run locally with synthetic identities | Freeze external contracts before real clients depend on them |

## M2 core alpha acceptance

| ID | Observable pass condition |
|---|---|
| A1 | Unknown/inactive users cannot authenticate; password policy, hashing version upgrade, throttling and enumeration tests pass |
| A2 | TOTP enrollment requires proof and fresh authentication; OTP reuse fails; recovery codes are single-use; reset cannot bypass required MFA; last owner cannot be removed |
| A3 | Two independent RP libraries complete code+S256 login; missing/wrong PKCE, wrong client, replayed code, unsafe redirect and issuer/audience/nonce failures are rejected |
| A4 | Discovery advertises only implemented capabilities; JWKS rotation preserves valid unexpired tokens during planned overlap; none/algorithm confusion tests fail closed |
| A5 | Two applications reuse one eligible SSO session; prompt=login, prompt=none, max_age and consent requirements give the expected reauthentication/error behavior |
| A6 | Logout, user disable, credential compromise and admin revoke deny subsequent online validation/refresh; registered back-channel clients clear their app sessions; unreachable clients create observable bounded retries |
| A7 | Two simultaneous refresh uses yield at most one rotated token, detect reuse, revoke the family, and force reauthentication; client documentation explains serialization |
| A8 | Consent/scopes/audiences/client assignments cannot be escalated by request parameters; client credentials never produce a human identity |
| A9 | Every privileged mutation is authorized server-side and atomically audit-recorded; audit has actor/tenant/action/target/outcome/correlation/time but no credentials |
| A10 | Admin and self-service UI/API flows have browser tests, CSRF/CSP/cookie checks, keyboard checks, and generic enumeration-resistant errors |
| A11 | Both requested binary targets pass native smoke tests and local Linux amd64 image runs non-root with read-only root filesystem and explicit writable state |
| A12 | Fresh installation, migration, key backup, isolated restore, and failure-to-connect-to-DB behavior pass; no default admin password or embedded secret exists |

## M3 beta acceptance

| ID | Observable pass condition |
|---|---|
| B1 | Two-tenant negative suite covers every resource, search/list/export, relationship, token/cache, job, logout, audit and admin path; foreign-tenant IDs never disclose or mutate data |
| B2 | Organization invitations expire and are single-use; ownership transfers are auditable; membership removal revokes affected grants without promoting another organization |
| B3 | WebAuthn verifies challenge/origin/RP ID/signature/UV requirements; replay and foreign-tenant credential fail; synchronized passkey counter behavior is handled without false safety claims |
| B4 | Losing all authenticators follows an explicit reviewed recovery path; recovery does not silently restore privileged sessions or reduce administrator assurance |
| B5 | Upstream OIDC login validates issuer/audience/nonce; account linking requires existing-account proof; malicious metadata/redirect destinations cannot trigger SSRF |
| B6 | Delegated application administrators cannot grant tenant-owner roles or edit unrelated applications; service scopes have equivalent negative tests |
| B7 | Go RP and browser/BFF walkthroughs pass on clean synthetic clients, including expiry, refresh failure, logout, session revoke, wrong audience, and migration of application configuration |

## M4 enterprise compatibility acceptance

| ID | Observable pass condition |
|---|---|
| C1 | SAML SP and IdP tested separately against two compatible implementations, with signed metadata/assertion rules, request correlation, replay/time/audience/recipient checks, and XML signature-wrapping/XXE regressions |
| C2 | SCIM inbound users/groups handles pagination/filtering/PATCH/idempotency and deprovisioning; cross-tenant requests and overprivileged provisioning tokens fail |
| C3 | LDAP/AD connector verifies TLS, handles search escaping and outage/disable semantics, and does not persist directory passwords; no insecure certificate bypass |
| C4 | Provisioning ownership and conflict rules prevent connector sync from overwriting local security state; partial sync failures are auditable/retryable |
| C5 | Selected device authorization/PAR/private_key_jwt extensions each have standard-specific negative tests and capability metadata; unselected extensions remain unadvertised |
| C6 | Published compatibility report identifies exact product/library versions, direction and tested profile; commercial products without authorized test access are explicitly unverified |

## M5 readiness evidence

| ID | Observable pass condition |
|---|---|
| R1 | Selected local OpenID conformance profiles pass with retained synthetic reports; certification is separately scoped and never inferred from a local pass |
| R2 | Independent security review covers the threat model; no unresolved critical/high exploitable findings; accepted lower risks have named owner and rationale |
| R3 | Two replicas preserve single-use/rotation/disable behavior across crash, DB failover, job retry, cache loss and clock skew; no sticky session dependence |
| R4 | Restore drill meets proposed single-site RPO <=24 h / RTO <=4 h on recorded dataset; smaller HA targets require separately measured WAL/PITR design |
| R5 | Both scheduled and emergency signing/encryption-key rotation drills pass, including retention of old verification/decryption keys and compromise revocation |
| R6 | Reference load test meets product objectives or explicitly revises the supported envelope before release; password hashing and abuse load measured separately |
| R7 | Native Windows/Linux smoke and Linux Docker smoke verify embedded UI, TLS/CA behavior, graceful shutdown, state permissions, health checks and offline asset loading |
| R8 | Frozen dependency locks, SBOM, notices, advisories, artifact hashes and build provenance exist; security update/incident/rollback runbooks are rehearsed |
| R9 | Operator and developer documentation matches observed behavior; release, deployment, real credentials, infrastructure and public access have separate user authorization |

## Ordering and change control

M1 precedes business implementation. Within M2: storage/invariants, protocol core,
credentials/recovery, sessions/consent, administration/audit, then integration and
packaging; security tests accompany each slice. M3 depends on stable M2 contracts.
M4 integrations are separable and may be reprioritized without claiming complete
enterprise coverage. M5 collects continuous hardening evidence into a release gate.

At each stage create a bounded implementation plan with exact files, commands,
dependencies and commit rows after inspecting the then-current repository. Keep
`main`; do not create branches/worktrees. One logical code change per future
authorized commit. No future commit message or broad directory in this roadmap
grants commit consent. Return to review when an issuer, schema, trust boundary,
license, protocol profile, or intended platform changes materially.
