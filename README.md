# QoAuth

QoAuth is a lightweight self-hosted identity and access management project. In this
project, “OA” means unified identity authentication, not office automation.

**Status:** M1 local feasibility implemented and tested; results are ready for
parent review. The application executable currently serves the embedded UI shell.
The working OIDC flow is an isolated synthetic test harness, not a login service
for real users. There is no production-readiness or all-platform-runtime claim.

The implementation uses Go with embedded frontend assets and targets Windows
amd64, Linux amd64, and a Linux amd64 Docker image. The first two targets compile;
the container configuration validates. Native Linux and container execution remain
unverified because no usable Linux container engine was available.

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

## Run the local feasibility checks

Use the existing Go 1.26.10 and Node 24.22.1/npm 12.2.0 toolchain; these commands
download locked project dependencies, not system tools. Build assets before Go:

```powershell
Push-Location web
npm ci --ignore-scripts
npm run build
Pop-Location
$env:GOTOOLCHAIN = 'local'
go test -timeout 60s ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
```

On a host with an existing C compiler, also run
`go test -timeout 60s -race ./internal/store ./internal/fixture` with `CGO_ENABLED=1`.
Use explicit package paths so Go does not traverse `web/node_modules`.
The [integration guide](docs/integration.md#m1-executable-fixture) identifies the
standard-client and independent-resource tests. They use temporary databases,
synthetic identities and memory-only keys, and close their local HTTP servers.

`scripts/build.ps1` builds the frontend and two standalone **UI shell** executables
under ignored `dist/`. No adjacent assets directory or Node runtime is required.
The shell defaults to `127.0.0.1:8080`; it does not create an administrator, database,
signing key or OAuth client. Docker configuration is preparatory, not a deployed
or runtime-validated image. The fixture is not linked into the application binary.

Sources and M1 evidence are dated 2026-10-25. See the single
[execution checklist](docs/plan/2026-10-25-qoauth-overall.md) for actual checks,
limitations and the next bounded slice. Real credentials and deployment remain
outside the actions performed.
