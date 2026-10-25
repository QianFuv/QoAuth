# IAM landscape and lessons for lightweight QoAuth

Verified: **2026-10-25, Asia/Singapore**. Official documentation, repositories,
licenses and standards were read; none of these products was installed or tested.
See the [source ledger](sources.md) for evidence and limitations. This is a bounded
representative survey, not a security ranking or exhaustive procurement study.

The user's latest scope is personal/small-team self-hosting. Enterprise features
below explain the landscape; they are **not QoAuth requirements or mandatory future
milestones**. The product contract is [the lightweight scope](../product.md).

## How to read the matrices

“Documented” means a vendor/project describes the capability, not a successful
interoperability test here. **Paid** denotes a known commercial boundary; **U**
means not established in this research, not proven absent; **Roadmap** is not a
shipping promise. OP/IdP issues authentication; RP/SP consumes it. SCIM provisioning
is separate from login, and its direction/resources matter. OAuth is not OIDC login.

## License, deployment and edition matrix

| Product | Backend/source terms | Free / paid / hosted distinction | Fit to QoAuth's delivery shape |
|---|---|---|---|
| Keycloak | Apache-2.0 upstream | Community self-hosting; commercial vendor distributions/support are separate | Integrated IAM but JVM, not Go; useful behavior reference [R01](sources.md#r01) |
| authentik | MIT core with explicit exceptions; enterprise directory has proprietary terms | Self-hosted core; enterprise features need production subscription; Enterprise Plus differs in support | Server/worker/PostgreSQL and optional outposts, not a Go executable [R02](sources.md#r02) |
| ZITADEL | Current server AGPL-3.0-only; directory-specific Apache/MIT exceptions; commercial alternative | Self-hosted OSS and hosted Cloud are distinct | Go server, PostgreSQL; current Login V2 separate from server; independent OIDC Go library has its own Apache license [R03](sources.md#r03) |
| Ory | Hydra/Kratos/Keto/Oathkeeper root licenses Apache-2.0 | OSS components vs Enterprise builds/features/support vs hosted Network | Go components, but assembled identity product needs multiple responsibilities and UI integration [R04](sources.md#r04) |
| Casdoor | Apache-2.0 | Self-hosted and managed offerings; no complete paid-feature entitlement audit here | Go + web UI, but inspected packaging ships web/build beside binary [R05](sources.md#r05) |
| Authelia | Apache-2.0 | OSS self-hosted project; no claim of enterprise SLA | Go directly embeds web assets; closest packaging reference, narrower product scope [R06](sources.md#r06) |
| Auth0 | Proprietary hosted service; Go SDK MIT | Free/paid/Enterprise plans; Private Cloud is vendor-managed, not a self-hostable OSS distribution | Strong integration reference, fails local binary ownership goal [R07](sources.md#r07) |
| Okta | Proprietary hosted service; Go management SDK Apache-2.0 | Integrator Free is development/testing; production suites and API Access Management entitlements separate | Workforce reference, not an embeddable backend [R08](sources.md#r08) |
| Microsoft Entra | Proprietary hosted service; MSAL Go MIT | Workforce Free/P1/P2 and External ID CIAM pricing are different; premium security/governance varies | Cloud directory/identity service, not local single-file IAM [R09](sources.md#r09) |
| Clerk | Proprietary hosted service; Go SDK MIT | Free and paid plans; production MFA/passkeys/enterprise SSO and administrative features have paid boundaries | Excellent application-UX reference; hosted dependency [R10](sources.md#r10) |

Open-source code, zero-price usage, paid production features and hosted operation
are four different properties. A permissive SDK license does not license a vendor's
service backend. No trial, account, purchase or private credential was used.

## Protocol, identity and lifecycle matrix

| Product | Protocol / identity direction | MFA and recovery | SSO, logout and revoke caveat |
|---|---|---|---|
| Keycloak | OAuth/OIDC OP, SAML IdP; OIDC/SAML upstream broker; LDAP/AD federation; native inbound SCIM Users/Groups currently supported | OTP, WebAuthn/passkeys, recovery flows/codes documented | Central/offline sessions and logout mechanisms; offline token consumers still need revocation policy [R01](sources.md#r01) |
| authentik | OIDC OP/SAML IdP; upstream sources; LDAP source vs LDAP-serving outpost; SCIM source=inbound, provider=outbound | Flow-based TOTP/WebAuthn/static recovery codes | OIDC end-session/front/back-channel and token rotation/introspection documented; configured flows matter [R02](sources.md#r02) |
| ZITADEL | OIDC OP/SAML IdP and upstream federation; inbound SCIM User schema, Groups not supported in reviewed SCIM docs | Passkeys/TOTP/OTP and recovery APIs | Introspection can observe revocation; offline JWT verification differs [R03](sources.md#r03) |
| Ory | Hydra=OAuth/OIDC; Kratos=identity/login; Keto=permissions; Oathkeeper=request enforcement | Kratos credentials, MFA/passkeys and recovery | Kratos identity sessions, Hydra grants and app sessions must be coordinated [R04](sources.md#r04) |
| Casdoor | OIDC/OAuth, SAML IdP/SP; LDAP sync/server; inbound SCIM Users/Groups; SCIM syncer imports rather than outbound provision | WebAuthn/MFA documented; complete recovery semantics U | RP-initiated logout documented; full revoke/restore semantics U [R05](sources.md#r05) |
| Authelia | Reverse-proxy authentication plus OIDC OP; LDAP/file users; SAML IdP Roadmap | TOTP/WebAuthn/passkeys documented | OIDC roadmap still shows session/logout items incomplete; do not assume all SSO logout profiles [R06](sources.md#r06) |
| Auth0 | OIDC/OAuth/SAML and enterprise connections; inbound connection-scoped SCIM | Passkeys and plan-dependent MFA/recovery | App/Auth0/upstream sessions separate; refresh-grant revocation configurable [R07](sources.md#r07) |
| Okta | Workforce SSO OIDC/SAML; directory lifecycle/SCIM; custom API AS has production entitlement boundary | Policy-driven MFA/WebAuthn/recovery | Session revoke does not erase app-created cookies; OAuth token revocation is a separate option [R08](sources.md#r08) |
| Entra | Separate workforce and external CIAM tenants; OIDC/OAuth and documented SAML support; provisioning direction varies | Workforce and External ID method combinations differ; not feature parity | revokeSignInSessions may be delayed; external home-tenant and app sessions remain separate [R09](sources.md#r09) |
| Clerk | App CIAM, enterprise SAML/OIDC connections, OAuth/OIDC provider capabilities; Directory Sync | MFA/passkeys and recovery availability vary by plan/flow | Short browser-session JWTs differ from OAuth token lifetimes; opaque OAuth tokens allow online revoke checks [R10](sources.md#r10) |

## Administration, permissions and operations matrix

| Product | Directory / organization and permissions | Admin / logs / operations | Portability or operational concern |
|---|---|---|---|
| Keycloak | Realms vs organizations; roles/groups/client scopes/consent | Integrated console/API and user/admin events; DB/HA planning | Realm export explicitly omits some sessions/events/revoked state; not a full backup [R01](sources.md#r01) |
| authentik | Brands are presentation; real tenancy is Enterprise alpha with expression-policy cross-tenant caveat | Admin UI, delegated permissions, flow consent, events; PostgreSQL plus files backup | Flow flexibility increases configuration/testing burden; Redis removed in documented 2025.10 architecture [R02](sources.md#r02) |
| ZITADEL | Instances, organizations, projects and delegated project roles | Console/REST/gRPC; event-oriented audit model | Current login runtime and AGPL matter; per-scope interactive consent parity U [R03](sources.md#r03) |
| Ory | Component-specific identity/relations; advanced tenancy/organization/SAML/SCIM claims require edition check | OSS APIs/CLI; complete combined console/audit needs integration | Protect management APIs; a collection of Go services is not one small product [R04](sources.md#r04) |
| Casdoor | Organizations/roles/permissions and integrated UI | Broad integrated administration | Hard isolation, audit/restore guarantees not established; do not equate organization label with isolation [R05](sources.md#r05) |
| Authelia | Configuration-oriented users/groups/access policy | Lightweight proxy-focused operation | HA adds shared state dependencies; not a general organization/provisioning suite [R06](sources.md#r06) |
| Auth0 | Organizations inside tenant, org roles, API RBAC and consent | Dashboard/API; log retention/streaming and admin rights vary by plan | Credential export is gated, not ordinary profile export [R07](sources.md#r07) |
| Okta | Directory sourcing, app assignments, consent and scoped admin roles | System Log API and lifecycle tooling | Production entitlement and migration/export eligibility need exact verification [R08](sources.md#r08) |
| Entra | Workforce directory/guest governance differs from External ID app roles/groups | Portal/Graph; finite log retention, Azure export integrations | Profile CSV is not a credential backup; some CIAM docs disagree [R09](sources.md#r09) |
| Clerk | Organizations, invitations and role/permission UI | Prebuilt components/backend API; higher-tier admin logs/SIEM | Self-service password-hash export documented; not proof all authenticators/sessions migrate [R10](sources.md#r10) |

## Representative deeper comparisons

### Keycloak and authentik: breadth is not the target scope

Keycloak is the integrated protocol/lifecycle benchmark. Its current 26.8 release
promoted native SCIM; old “SCIM only via plugins” comparisons are stale. More useful
to QoAuth are precise sessions, client/consent separation and the distinction
between configuration export and recoverable state. Its JVM and broad feature set
are not the packaging or feature target. [R01](sources.md#r01)

authentik demonstrates clear connector direction and configurable authentication
flows. Its tenancy documentation explicitly marks Enterprise alpha and warns that
expression policies can reach other tenants. This is a reason to avoid introducing
a tenant product casually, not a task for QoAuth to replicate. A fixed, comprehensible
login/recovery flow is more appropriate here. [R02](sources.md#r02)

### Ory and ZITADEL: component boundaries and licenses

Ory makes the distinction between identity, protocol issuance, relationship
authorization and request enforcement concrete. Hydra alone does not supply a user
directory or account-recovery UI; Kratos alone is not an OIDC OP. QoAuth can retain
those conceptual boundaries inside a small monolith without adopting multiple
services. Native SAML/SCIM/advanced tenancy in commercial documentation must not be
silently attributed to every OSS component. [R04](sources.md#r04)

ZITADEL is useful for observing the difference between a Go backend and a fully
self-contained product: Login V2 is separate. Its current server AGPL terms differ
from its separate Apache-licensed OIDC library. This proposal evaluates the library,
not a fork of the server. Organization/project delegation remains outside QoAuth's
current scope. [R03](sources.md#r03), [library review](../dependencies.md)

### Casdoor and Authelia: nearest packaging references

Casdoor offers a broad Go-based integrated system, but the inspected release layout
includes a separate web/build directory; that is not the requested embedded-resource
artifact. Its broad connector and organization catalog is not a reason to add them
to QoAuth. Authelia directly embeds its UI and is the stronger inspected mechanism
reference, while its published OIDC/logout roadmap requires profile-by-profile
checking. Neither “written in Go” nor “OIDC supported” establishes the exact product
contract. [R05](sources.md#r05), [R06](sources.md#r06)

### Auth0 and Clerk: integration completeness and exit behavior

Auth0 separates application, provider and upstream sessions. Its ordinary export
does not expose password hashes/private keys; sensitive exports require support
and eligibility review. Current Free/paid B2B/B2C comparison text has entitlement
inconsistencies, so do not generalize a Free-card allowance to all paid plans.
QoAuth should learn from clear integration guides and document its own logout and
backup limits. [R07](sources.md#r07)

Clerk shows the value of ready-to-use login/account UI and a coherent backend API.
Its free usage unit is monthly retained users, which is not interchangeable with
MAU. Production MFA and enterprise SSO have paid boundaries; announced Directory
Sync charges from 2027-01-01 are future, not effective on the verification date.
Short session JWTs do not describe all OAuth tokens. Documented password-hash export
is useful portability evidence but not a full-service backup guarantee. [R10](sources.md#r10)

### Entra and Okta: appendix reference, not an adoption roadmap

Entra workforce and External ID customer tenants have different methods, roles,
provisioning and governance. External ID documentation contains unresolved recovery
and consent inconsistencies; runtime checks are needed if a real integration is
later requested. Okta's free integrator offering is not a general free production
license, and its custom API authorization server entitlement is separate from
basic organizational SSO. These products illustrate enterprise concerns which the
user has explicitly removed from QoAuth's current commitment. [R08](sources.md#r08),
[R09](sources.md#r09)

## Resulting recommendation

Build only the lightweight contract: local users, applications/scopes, standards-
based OIDC, safe sessions/revoke, TOTP/recovery, small admin UI and simple operations.
Use SQLite first and an existing OP library after the security/storage spike.
Borrow integration clarity and lifecycle correctness from the wider market;
do not copy its organizational structures, infrastructure or connector catalog.
