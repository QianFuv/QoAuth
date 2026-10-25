# Primary-source ledger and verification limits

All entries below were consulted on **2026-10-25 (Asia/Singapore)** by the primary
agent or a read-only Astra research delegate. Parent review cross-checked material
license, edition, packaging, standards and advisory findings. No runtime product
test was performed. A link is evidence for its stated subject, not all claims about
that product. Dynamic docs/default-branch licenses may change; release-pinned
dependency evaluation is separate in [dependencies](../dependencies.md).

<a id="r01"></a>
## R01 — Keycloak

- [License](https://github.com/keycloak/keycloak/blob/main/LICENSE.txt): Apache-2.0 upstream.
- [Server administration](https://www.keycloak.org/docs/latest/server_admin/index.html): federation, credentials, roles, sessions, organizations, consent and events.
- [26.8.0 release](https://www.keycloak.org/2026/10/keycloak-2680-released) and [feature status](https://www.keycloak.org/server/features): current native SCIM status; release dated 2026-10-01.
- [Import/export limits](https://www.keycloak.org/server/importExport): export excludes material recovery state.

<a id="r02"></a>
## R02 — authentik

- [Core license](https://github.com/goauthentik/authentik/blob/main/LICENSE), [enterprise license](https://github.com/goauthentik/authentik/blob/main/authentik/enterprise/LICENSE), [editions](https://docs.goauthentik.io/enterprise/): mixed license/production subscription boundary.
- [OAuth provider](https://docs.goauthentik.io/add-secure-apps/providers/oauth2), [SAML source](https://docs.goauthentik.io/users-sources/sources/protocols/saml/), [authenticators](https://docs.goauthentik.io/users-sources/user/user-interface): protocol and authentication behavior.
- [LDAP source](https://docs.goauthentik.io/users-sources/sources/protocols/ldap/), [LDAP provider](https://docs.goauthentik.io/add-secure-apps/providers/ldap/), [SCIM source](https://docs.goauthentik.io/users-sources/sources/protocols/scim), [SCIM provider](https://docs.goauthentik.io/add-secure-apps/providers/scim): direction matters.
- [Tenancy](https://docs.goauthentik.io/sys-mgmt/tenancy/): Enterprise alpha, PostgreSQL schemas and expression-policy limitation.
- [Architecture](https://docs.goauthentik.io/core/architecture/), [2025.10 changes](https://docs.goauthentik.io/releases/2025.10), [backup](https://docs.goauthentik.io/sys-mgmt/ops/backup-restore/), [events](https://docs.goauthentik.io/sys-mgmt/events/): operation and audit reference.

<a id="r03"></a>
## R03 — ZITADEL

- [Licensing policy](https://github.com/zitadel/zitadel/blob/main/LICENSING.md) and [license](https://github.com/zitadel/zitadel/blob/main/LICENSE): AGPL-3.0-only server and explicit directory exceptions; not the standalone OIDC library's license.
- [OIDC endpoints](https://zitadel.com/docs/apis/openidoauth/endpoints), [back-channel logout](https://zitadel.com/docs/guides/integrate/back-channel-logout), [SCIM limits](https://zitadel.com/docs/guides/manage/user/scim2): token/session behavior and User-only SCIM coverage.
- [Authentication settings](https://zitadel.com/docs/guides/manage/console/default-settings), [organizations](https://zitadel.com/docs/guides/manage/console/organizations-overview), [audit trail](https://zitadel.com/docs/concepts/features/audit-trail): policy/admin model.
- [API/UI deployment](https://zitadel.com/docs/apis/introduction), [production operations](https://zitadel.com/docs/self-hosting/manage/production): separate Login V2 and durable state.
- Unverified here: third-party per-scope consent parity, complete LDAP synchronization, outbound SCIM.

<a id="r04"></a>
## R04 — Ory

- [Hydra](https://github.com/ory/hydra), [Kratos](https://github.com/ory/kratos), [Keto](https://github.com/ory/keto), [Oathkeeper](https://github.com/ory/oathkeeper): responsibilities, current root licenses and edition statements; each inspected root LICENSE is Apache-2.0.
- [Kratos OSS](https://www.ory.com/docs/oss/kratos/intro), [Hydra logout](https://www.ory.com/docs/hydra/guides/logout): identity/protocol separation.
- [Management API protection](https://www.ory.com/docs/self-hosted/operations/api-access-control), [logging](https://www.ory.com/docs/self-hosted/operations/logging): operational integration work.
- [Polis](https://www.ory.com/docs/network/polis): separate SAML component; not counted as native capability of the four OSS components. Polis licensing was not audited.

<a id="r05"></a>
## R05 — Casdoor

- [Repository/license](https://github.com/casdoor/casdoor), [release packaging](https://github.com/casdoor/casdoor/blob/master/.goreleaser.yaml), [static serving](https://github.com/casdoor/casdoor/blob/master/routers/static_filter.go): Go/Apache and separate web files.
- [SAML SP](https://casdoor.ai/docs/provider/saml/overview/), [LDAP sync](https://casdoor.ai/docs/ldap/config/), [LDAP server](https://casdoor.ai/docs/ldap/ldapserver/), [SCIM server](https://casdoor.ai/docs/scim/overview/), [SCIM syncer](https://casdoor.ai/docs/syncer/SCIM/): connector direction and coverage.
- [RP logout](https://casdoor.ai/docs/session/rp-initiated-logout): documented logout surface; full revocation/recovery/restore properties not tested.

<a id="r06"></a>
## R06 — Authelia

- [Repository/license](https://github.com/authelia/authelia), [embedded assets](https://github.com/authelia/authelia/blob/master/internal/server/asset.go): Apache-2.0 and actual go:embed source.
- [OIDC roadmap](https://www.authelia.com/roadmap/active/openid-connect-1.0-provider/), [SAML roadmap](https://www.authelia.com/roadmap/planning/security-assertion-markup-language-saml-2.0-identity-provider/): capability status, not assumed completeness from a badge.
- [Security keys/passkeys](https://www.authelia.com/overview/authentication/security-key/), [state/HA](https://www.authelia.com/overview/authorization/statelessness/): authentication and deployment boundaries.

<a id="r07"></a>
## R07 — Auth0

- [Pricing](https://auth0.com/pricing), [Private Cloud](https://auth0.com/docs/deploy-monitor/deploy-private-cloud): free/paid/Enterprise and vendor-managed deployment; current cards/comparison have B2B/B2C entitlement ambiguity.
- [Protocols](https://auth0.com/docs/authenticate/protocols), [organizations](https://auth0.com/docs/manage-users/organizations), [organization token checks](https://auth0.com/docs/manage-users/organizations/using-tokens), [MFA](https://auth0.com/docs/secure/multi-factor-authentication): feature evidence.
- [Inbound SCIM](https://auth0.com/docs/authenticate/protocols/scim/configure-inbound-scim), [logout layers](https://auth0.com/docs/authenticate/login/logout), [refresh revoke](https://auth0.com/docs/secure/tokens/refresh-tokens/revoke-refresh-tokens): lifecycle semantics.
- [Log retention](https://auth0.com/docs/deploy-monitor/logs/log-data-retention), [export policy](https://auth0.com/docs/troubleshoot/customer-support/operational-policies/data-export-and-transfer-policy), [secret export](https://auth0.com/docs/manage-users/user-migration/export-password-hashes-and-mfa-secrets): portability limits.
- [Go SDK/license](https://github.com/auth0/go-auth0), [Go quickstart](https://auth0.com/docs/quickstart/webapp/golang): integration reference; quickstart and SDK majors require reconciliation before use.

<a id="r08"></a>
## R08 — Okta

- [Plans](https://www.okta.com/pricing/), [Integrator Free limits](https://developer.okta.com/docs/reference/org-defaults/), [authorization servers](https://developer.okta.com/docs/concepts/auth-servers): development vs production and API entitlement.
- [Organizations](https://developer.okta.com/docs/concepts/okta-organizations/), [SCIM](https://developer.okta.com/docs/concepts/scim), [consent](https://developer.okta.com/docs/guides/request-user-consent/main/), [WebAuthn](https://help.okta.com/en-us/content/topics/security/mfa-webauthn.htm): functional boundaries.
- [System Log](https://developer.okta.com/docs/api/openapi/okta-management/management/tags/systemlog), [session API](https://developer.okta.com/okta-sdk-java/25.0.0/apidocs/apidocs/com/okta/sdk/resource/api/UserSessionsApi.html), [Go SDK/license](https://github.com/okta/okta-sdk-golang): admin/integration evidence.
- General password-hash export eligibility remains unverified; a support article body could not be retrieved. No blanket claim of portability or impossibility.

<a id="r09"></a>
## R09 — Microsoft Entra

- [Workforce pricing](https://www.microsoft.com/en-us/security/business/microsoft-entra-pricing), [External ID pricing](https://www.microsoft.com/en-us/security/pricing/microsoft-entra-external-id), [security defaults](https://learn.microsoft.com/en-us/entra/fundamentals/security-defaults): separate offerings and premium boundaries.
- [Workforce/CIAM feature comparison](https://learn.microsoft.com/en-us/entra/external-id/customers/concept-supported-features-customers), [passkey prerequisites](https://learn.microsoft.com/en-us/entra/external-id/customers/how-to-sign-in-with-passkey): do not infer workforce method parity.
- [Password-reset guide](https://learn.microsoft.com/en-us/entra/external-id/customers/how-to-enable-password-reset-customers) vs [FAQ](https://learn.microsoft.com/en-us/entra/external-id/customers/faq-customers): SMS recovery discrepancy; customer-consent descriptions also need runtime confirmation.
- [Revoke sessions](https://learn.microsoft.com/en-us/graph/api/user-revokesigninsessions?view=graph-rest-1.0), [retention](https://learn.microsoft.com/en-us/entra/identity/monitoring-health/reference-reports-data-retention), [profile export](https://learn.microsoft.com/en-us/entra/identity/users/users-bulk-download), [MSAL Go/license](https://github.com/AzureAD/microsoft-authentication-library-for-go): session/export/SDK boundaries.

<a id="r10"></a>
## R10 — Clerk

- [Pricing](https://clerk.com/pricing), [architecture](https://clerk.com/docs/guides/how-clerk-works/overview): hosted service, retained-user metric, plan and log boundaries.
- [Enterprise connections](https://clerk.com/docs/guides/configure/auth-strategies/enterprise-connections/overview), [Directory Sync](https://clerk.com/docs/guides/configure/auth-strategies/enterprise-connections/directory-sync): production entitlements and future-dated billing notice.
- [Authentication/recovery](https://clerk.com/docs/guides/configure/auth-strategies/sign-up-sign-in-options), [OAuth behavior](https://clerk.com/docs/guides/configure/auth-strategies/oauth/how-clerk-implements-oauth): session tokens and OAuth tokens are different.
- [Migration/export](https://clerk.com/docs/guides/development/migrating/overview), [Go SDK/license](https://github.com/clerk/clerk-sdk-go): documented export and integration scope.

## Standards and local evidence

Protocol and security design cite original RFC/OpenID/NIST/W3C/OWASP material
directly in the [architecture](../architecture.md), [security](../security.md),
[dependencies](../dependencies.md) and [integration](../integration.md) documents.
Local packaging evidence is recorded in architecture.md with exact inspected paths.
No private source, credential or file content was sent to a research service.

This survey is deliberately not a complete license audit, vendor SLA comparison,
paid-entitlement guarantee, benchmark, penetration test or conformance report.

## Final packaging evidence sources

Checked 2026-10-25: official [SPDX 2.3.1 JSON schema](https://raw.githubusercontent.com/spdx/spdx-spec/v2.3.1/schemas/spdx-schema.json)
and [Vite build.license / modulepreload](https://vite.dev/config/build-options.html).
Installed Vite 8.3.5 source, final executable build metadata, original installed
license/NOTICE bytes and generated manifests supply artifact-specific evidence;
package-lock/go.sum alone do not establish linked runtime components. The final
[dependency inventory](../dependencies.md#artifact-inventories-and-preserved-notices)
records scope, supplements and remaining license/image boundaries.

## A6 image evidence sources

Checked 2026-10-25: the actual Docker image, extracted executable build information,
Linux frontend-stage inventory, BuildKit base digests and the supplying Alpine APK
metadata. [Pinned aports recipe](https://raw.githubusercontent.com/alpinelinux/aports/e78f5987166c6ffbe23f6472e2e8f040298fe2da/main/ca-certificates/APKBUILD)
identifies ca-certificates 20261020 and the source SHA-512. The exact
[Alpine source archive](https://gitlab.alpinelinux.org/alpine/ca-certificates/-/archive/20261020/ca-certificates-20261020.tar.bz2)
was verified against that digest; all 118 shipped certificate DER values occur in
its certdata.txt. Full [MPL-2.0 text](https://www.mozilla.org/media/MPL/2.0/index.txt)
and [SPDX MIT text](https://raw.githubusercontent.com/spdx/license-list-data/main/text/MIT.txt)
are preserved with original attribution and hashes. Source membership and SPDX
schema validity are evidence, not independent legal clearance or product licensing.
