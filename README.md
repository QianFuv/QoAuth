# QoAuth

QoAuth is a lightweight self-hosted identity service for a person or small team.
Here “OA” means unified identity authentication, not office automation.
One Go process contains a React/TypeScript/Vite/Semi UI and uses one local SQLite
database and one issuer. No external database, Redis, Node runtime or enterprise
tenant/SCIM/LDAP/SAML infrastructure is required.

**Status, 2026-10-25:** M2.1 implements local accounts, password/TOTP/recovery,
sessions, applications, consent, management UI/API and standard OIDC code+S256.
The real service is `cmd/qoauth`; `internal/fixture` is test-only and is not imported
by it. All validation used temporary synthetic state. This is not a production
readiness or OpenID certification claim.

Windows amd64 and Linux amd64 binaries build. The Windows binary was initialized,
started and restarted with a temporary installation and unchanged JWKS. Linux
native execution and Docker image build/run remain unverified. Compose config
validates; no system service, deployment, real owner or external binding was created.

## Build and check

Use the existing Go 1.26.10 and Node 24.22.1/npm 12.2.0 toolchain. Build assets first:

```powershell
Push-Location web
npm ci --ignore-scripts
npm run build
Pop-Location
$env:GOTOOLCHAIN = 'local'
go test -timeout 180s ./cmd/... ./internal/... ./examples/...
go vet ./cmd/... ./internal/... ./examples/...
```

With an existing C compiler, run the same package set with `CGO_ENABLED=1` and
`go test -timeout 180s -race`. Explicit paths avoid traversing `web/node_modules`.
The measured run passed 79 tests/subtests across seven packages; three opt-in
helper/browser entrypoints were skipped in the ordinary suite. The browser harness
was separately executed. See [exact evidence and limits](docs/plan/2026-10-25-qoauth-overall.md#final-m21-evidence-2026-10-25).

`scripts/build.ps1` builds two single-file executables under ignored `dist/`.
No adjacent assets directory is required. Builds retain symbol tables for precise
binary vulnerability scans; debug information is omitted. The container has a
separate frontend build stage and runs as a non-root user, but has not been run here.

## Service entry points

Commands below describe the implemented interface, not actions performed against
real state. Choose and protect a local data directory before real use:

```text
qoauth init --data-dir <private-directory> --username <owner-name>
qoauth serve --data-dir <private-directory> --issuer https://auth.example.test
qoauth recover-owner --data-dir <private-directory> --confirm
```

`init` and `recover-owner` read the new password from redirected standard input;
there is no password argument or echoed terminal entry. Use an operator-controlled
secret source, never a password literal in shell history. Initialization is explicit,
one-use and has no anonymous web claim. Owner access stays restricted until TOTP
enrollment is confirmed. Recovery invalidates old access and requires a new factor.

`serve` listens on `127.0.0.1:8080` by default, behind an existing HTTPS reverse
proxy that preserves the canonical Host. When proxying, set `--trusted-proxy <IP>`
and have that one proxy overwrite `X-Real-IP` with exactly one client IP. No other
forwarded header selects the issuer or bypasses throttling. Explicit local testing
may use `--development-http` with numeric loopback issuer/listener addresses only.

`keys.json` holds the signing private key and AES root secret under host permissions;
TOTP seeds in SQLite are encrypted. It is sensitive plaintext key material, not a
password-protected key vault. Startup checks owner/permissions and database sidecars;
missing or mismatched keys fail closed. Do not expose the data directory or copy a
live database alone. Automated backup/restore and key rotation remain the next slice.

Docker listens on `0.0.0.0:8080` inside the container; Compose publishes only
`127.0.0.1:8080` on the host. The image contains no owner/keys; explicit initialization
and persistent private storage are still required. Proxy/TLS/network changes and
actual deployment need their own authorization.

## Review package

- [Single execution checklist, evidence and next scoped proposal](docs/plan/2026-10-25-qoauth-overall.md)
- [Product scope](docs/product.md), [roadmap](docs/roadmap.md), [architecture and operations](docs/architecture.md)
- [Threat model and remaining release gates](docs/security.md)
- [Dependencies, advisories and license obligations](docs/dependencies.md)
- [Runnable standard Go RP and API integration](docs/integration.md)
- [Competitor matrix](docs/research/comparison.md), [primary source ledger](docs/research/sources.md)
- [Authorization history](docs/project-charter.md)
