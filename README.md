# QoAuth

QoAuth is a proposed self-hosted identity and access management system. In this
project, “OA” means unified identity authentication, not office automation.

**Status:** lightweight direction accepted by parent internal review; local M1
feasibility work authorized. No production authentication or security claim.

The intended implementation uses Go with embedded frontend assets and targets
Windows amd64, Linux amd64, and a Linux amd64 Docker image. Implementation starts
under parent decision A2, recorded in the single execution checklist below.

The current scope is **personal/small-team self-hosting: one Go process, one SQLite
database, one issuer**. Enterprise tenancy, SCIM/LDAP/SAML and cluster infrastructure
are excluded from the initial commitment.

Start with [the overall review plan](docs/plan/2026-10-25-qoauth-overall.md), then:

- [Required, optional and excluded features](docs/product.md)
- [Three short milestones and acceptance conditions](docs/roadmap.md)
- [Architecture, SQLite, packaging and minimal operations](docs/architecture.md)
- [Threat model, MFA/recovery tests and risks](docs/security.md)
- [Dependencies, current advisories and licenses](docs/dependencies.md)
- [Simple OIDC integration examples](docs/integration.md)
- [Ten-product comparison](docs/research/comparison.md) and [primary sources](docs/research/sources.md)
- [Project charter and authorization record](docs/project-charter.md)

Sources were verified on 2026-10-25. Documentation research is not a runtime,
conformance, security, build or deployment test. M1 uses isolated synthetic fixtures;
real accounts, credentials, network deployment and system changes are excluded.
