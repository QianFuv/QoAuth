# Product goals, boundaries, and feature tiers

Review status: proposal, 2026-10-25. No feature below is implemented or verified.

## Product position

QoAuth should be a self-hosted identity platform for small and medium engineering
teams and B2B applications that need a clear administrative interface, standard
SSO, predictable revocation, and a small operational footprint. The differentiator
is a coherent Go application with embedded UI and documented integration contracts,
not the largest connector catalog or an unqualified promise of enterprise parity.

Success means an application developer can integrate an existing OIDC library,
an administrator can explain who can access which application, and an operator can
revoke access, rotate keys, restore a backup, and upgrade without silent loss of
security state. A novice should not need to implement JWT verification or a login
protocol to use QoAuth.

## Responsibilities

| Responsibility | QoAuth owns | Application/operator owns |
|---|---|---|
| Authentication | Local/federated identity verification, assurance, authenticators, login sessions, OIDC ID Tokens | Correct OIDC client callback and its own session |
| OAuth authorization | Registered clients, consent, audience-bound delegated access, token lifecycles | Resource authorization and enforcement of granted scopes |
| User directory | Local users, immutable identifiers, tenant memberships, groups, external identity links | Authoritative HR/business records and connector credentials |
| Administration | Tenant-scoped administration, service principals, auditable configuration | Delegation decisions and operational approval |
| Business permissions | Coarse app roles and membership claims | Record ownership, payments, workflow, and domain-specific ABAC/ReBAC |
| Operations | Health/metrics, migrations, backup/restore procedure, secure defaults | Host/network/TLS/DB availability and custody of encryption roots |

OAuth alone is not a login protocol. ID Tokens describe authentication for their
audience; access tokens authorize calls to a resource. Do not use an access token
as an unvalidated user identity or an ID Token as an API bearer token.
Sources: [OIDC Core](https://openid.net/specs/openid-connect-core-1_0.html),
[OAuth security BCP](https://www.rfc-editor.org/rfc/rfc9700.html), verified 2026-10-25.

## Primary scenarios and measurable outcomes

| ID | Scenario | Observable outcome |
|---|---|---|
| U1 | Employee signs into two internal web applications | The second application reuses an eligible QoAuth session without collecting the password; each app receives only its own authorized claims |
| U2 | Administrator disables a departing member | New sign-in, refresh, and online token validation fail after the disable transaction; app sessions receive logout notification where integrated |
| U3 | B2B customer has multiple organizations | Membership and active organization are explicit; organization roles cannot administer another tenant or silently grant global rights |
| U4 | Existing OIDC/SAML identity provider supplies users | Upstream subjects are mapped by issuer/provider and subject; matching email alone cannot take over an existing account |
| U5 | User loses an authenticator | Documented recovery restores access at an explicit assurance level, alerts the user, and revokes compromised methods/sessions |
| U6 | Service calls an internal API | Client credentials grant only assigned audience/scopes, no end-user identity; rotation and disable are observable |
| U7 | Operator upgrades or loses the host | Versioned data and separate key backup restore into an isolated environment, with tested revocation and issuer continuity |
| U8 | Developer adds login to a Go web app or SPA | A library-based guide passes login, expiry, logout, and negative-token tests without custom crypto |

## Full-featured target, by layer

“Full-featured” means the accepted layers below, not every feature offered by every
vendor. Protocol direction is explicit: OP/IdP issues identity; RP/SP consumes it.

| Layer | Included target | First milestone | Excluded or deferred |
|---|---|---|---|
| Core identity | Invite/disable local users; verified email; password change/recovery; TOTP; recovery codes; administrator step-up | M2 | Open public signup, SMS login, security questions |
| OIDC/OAuth core | Discovery, JWKS, authorization code with S256 PKCE, ID Token, UserInfo, client credentials, introspection, revocation | M2 | Implicit, password grant, wildcard redirect URIs |
| Session lifecycle | Tenant SSO, session inventory, local and all-session logout, RP-initiated logout, refresh families, back-channel logout | M2 | Guaranteed logout of nonparticipating applications |
| Tenant/product model | One tenant initially on a tenant-scoped schema; organizations, invitations, group/app-role mapping, delegated admin | M2/M3 | Shared identities or automatic SSO across independent issuers |
| Modern authentication | WebAuthn/passkeys; enrollment/removal/step-up; recovery resistant to downgrades | M3 | Claiming every passkey meets every assurance level |
| Federation | Upstream OIDC first, then SAML SP; explicit linking and claim mapping | M3/M4 | Automatic email-based linking |
| Enterprise interoperability | SAML IdP, SCIM 2.0 inbound users/groups, LDAP/AD read/auth federation, optional outbound provisioning | M4 | General-purpose LDAP server; unlimited connector parity |
| Authorization and consent | Audience/scope allowlists, client assignment, tenant app roles, versioned consent/re-consent | M2/M3 | Embedded universal policy engine at initial release |
| Administration | Responsive admin UI and versioned API; user/client/key/session/audit management; accessible login screens | M2 onward | Arbitrary plugin execution or user-supplied login JavaScript |
| Operations | Structured redacted logs, audit export, metrics, rate limits, backups, upgrades, rotation, HA verification | M2/M5 | Unmeasured capacity/availability guarantees |
| Advanced protocols | Device authorization for CLI, PAR, private_key_jwt, optional JWT access tokens and sender constraints | M4+ separately prioritized | FAPI certification, DPoP/mTLS/token exchange claimed without profile tests |
| Ecosystem | Go RP example, browser/BFF guidance, generic OIDC instructions, admin OpenAPI | M2/M3 | Maintaining bespoke SDKs for every language |

## Administrative model

Separate deployment operator, tenant owner, tenant administrator, application
administrator, auditor, and end user. Tenant owners do not inherit deployment
operator powers. Application administrators manage assigned clients but cannot
grant themselves arbitrary tenant roles. Auditor is read-only and redacted.
Every role-changing action requires an existing authorized principal and an audit
event; deleting the last usable owner is rejected. Service accounts have explicit
scopes and cannot use an interactive user's session as their own credential.

UI sections: Overview; Users; Organizations and Groups; Applications; Identity
Providers; Authentication and Recovery; Sessions; Audit; Keys; Settings. Hide or
label unimplemented sections; a placeholder page must not imply a working feature.
Login/account screens must support keyboard use, clear errors without enumeration,
password managers, reduced motion, and English/Chinese localization-ready strings.

## Proposed limits, pending measurement

Start with a single-region service and a reference test dataset of 10,000 users,
20 tenants and 100 clients. This is a test fixture, not a supported-capacity claim.
Initial service objectives for the future benchmark: 50 non-password auth requests
per second for 30 minutes, p95 below 300 ms at the server, error rate below 0.1%,
excluding deliberate rejections. Measure password hashing separately with bounded
concurrency on a recorded 4-vCPU/8-GiB reference host; tune before promising latency.

Proposed defaults: 60-second one-use authorization codes; 5-minute access tokens;
8-hour idle and 24-hour absolute browser sessions; refresh families with 7-day idle
and 30-day absolute limits. Administrative sessions: 15-minute idle, 8-hour absolute,
and reauthentication within 5 minutes for keys/roles/recovery changes. Values are
product policy candidates, not standard requirements. Changing them needs documented
risk review and boundary tests, not silent configuration drift.

## Not in the initial product promise

No office workflows, HR/payroll, device management, full identity governance/PAM,
fraud scoring, government identity proofing, multi-region active-active service,
or formal FIPS/AAL/FAPI certification. An application portal and enterprise lifecycle
connectors can grow later; they do not displace correctness of the identity core.
