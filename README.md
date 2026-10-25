# QoAuth

QoAuth is a proposed self-hosted identity and access management system. In this
project, “OA” means unified identity authentication, not office automation.

**Status:** phase-one research and design. No authentication service, production
security claim, deployable binary, or container image exists yet.

The intended implementation uses Go with embedded frontend assets and targets
Windows amd64, Linux amd64, and a Linux amd64 Docker image. Implementation starts
only after the user reviews the design and authorizes the next stage.

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
conformance, security, build or deployment test. The proposal awaits user review;
no implementation or future automatic Git commit permission is implied.
