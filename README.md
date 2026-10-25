# QoAuth

QoAuth is a lightweight self-hosted identity service for a person or small team.
Here “OA” means unified identity authentication, not office automation.
One Go process contains a React/TypeScript/Vite/Semi UI and uses one local SQLite
database and one issuer. No external database, Redis, Node runtime or enterprise
tenant/SCIM/LDAP/SAML infrastructure is required.

**Status, 2026-10-25:** The approved lightweight product slice is implemented:
local users, password/TOTP/recovery, sessions, applications/scopes/consent, OIDC
code+S256, management UI/API and redacted activity. Maintenance includes installation
locking, consistent backup, isolated restore/reconciliation, signing rotation and
bounded cleanup. Final closure adds MFA-protected administrator password reset,
disabled-application review, consistent password rules and bounded readiness.
The real service is `cmd/qoauth`; `internal/fixture` is test-only and is not imported
by it. All validation used temporary synthetic state. This is not a production
readiness or OpenID certification claim.

Windows amd64 and Linux amd64 binaries build. The Windows binary was initialized,
started and restarted with a temporary installation and unchanged JWKS. Linux
native execution and Docker image build/run remain unverified. Compose config
validates; no system service, deployment, real owner or external binding was created.

## Build and check

Use Go 1.26.10 and Node 24.22.1/npm 12.2.0; local packaging also uses Python 3.12.15.
These tools were already available. Build assets first:

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
The final Windows race run passed 135 tests/subtests across eight packages, with
six opt-in/helper skips. Process helpers run through their parent tests; browser and
600-second load harnesses were also executed separately. Coreos and Zitadel RP flows
pass against the production adapter. See [final evidence and limits](docs/plan/2026-10-25-qoauth-overall.md#final-bounded-delivery-evidence-2026-10-25).

`scripts/build.ps1` builds both single-file executables and versioned local ZIPs,
SPDX 2.3 SBOMs, original license/NOTICE files, hashes and an exact-commit source ZIP
under ignored `dist/`. Use `-UseInstalledFrontend` to reuse already installed locked
packages. Final packaging requires a clean checkout and matching unmodified binary
build metadata; output directories must be new. Explicit `--allow-dirty` packaging
is always labeled `.dirty` and never includes a clean source archive.

No adjacent UI directory is required. Symbol tables remain for precise binary scans;
debug information is omitted. The final local preview is
[0.1.0-dev.826200d48959](dist/0.1.0-dev.826200d48959/manifest.json), built at
`826200d48959bef3e23f554fc39f73ea1635a149`; subsequent evidence documentation does
not relabel its source revision. Each platform directory contains its executable,
SBOM, notices and manifest. `build-development/inventory.json` separates the wider
build/test graph. The source ZIP includes Docker/Compose recipes and the Go RP
example. No image was built; these are local previews, not published releases.

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

GET `/readyz` returns only `{"ready":true}` (200) or `{"ready":false}` (503), with
no cookie and a one-second database deadline. It checks persisted issuer/key binding
and maintenance/quarantine state. Use the canonical Host even for a proxy-local
probe. It is not an external TLS check or full database integrity scan.

Administration can reset an ordinary account password after fresh administrator
MFA; it preserves enrolled MFA and revokes previous sessions/codes/grants. Disabled
applications have a read-only review action; replacement is explicit and never
restores old grants or reveals a stored secret. Enter one callback/logout URL per
line; literal commas are preserved. New passwords use the same NFC-normalized
15-character/1024-UTF-8-byte rules in CLI/API/UI.

`keys.json` holds the signing private key and AES root secret under host permissions;
TOTP seeds in SQLite are encrypted. It is sensitive plaintext key material, not a
password-protected key vault. Startup checks owner/permissions and database sidecars;
missing or mismatched keys fail closed. Do not expose the data directory or copy a
live database alone. All commands share an OS installation lock. Stop the instance
before maintenance; no background backup scheduler or remote service is installed.

Docker listens on `0.0.0.0:8080` inside the container; Compose publishes only
`127.0.0.1:8080` on the host. The image contains no owner/keys; explicit initialization
and persistent private storage are still required. Proxy/TLS/network changes and
actual deployment need their own authorization.

## Minimal maintenance

```text
qoauth backup --data-dir <stopped-installation> --snapshot <new-private-directory>
qoauth rotate-signing --data-dir <stopped-installation> --confirm
qoauth cleanup --data-dir <stopped-installation> --batch-size 128
qoauth restore --snapshot <verified-snapshot> --data-dir <new-private-directory> --confirm
qoauth recover-owner --data-dir <restored-directory> --confirm --restore-reviewed
qoauth recover-user --data-dir <stopped-installation> --username <disabled-user> --confirm
```

Backup uses locked SQLite VACUUM INTO and publishes the DB/key manifest last.
Restore never overwrites an existing destination. It checks the exact supported
schema, integrity, hashes and key binding, discards old authentication state and
signing trust, and disables users/clients. It requires local owner recovery, new MFA
and a persisted ten-minute signed-token expiry quarantine. Keep the old issuer
stopped throughout cutover; independent RP sessions need separate invalidation.
Do not run `init` against a restored or incomplete directory.

The snapshot includes plaintext operational keys: possession of the whole snapshot
can compromise identity and encrypted TOTP data. SHA-256 detects changed bytes; it
does not authenticate a maliciously replaced whole backup. Protect the full directory.
See the [operating and recovery procedure](docs/architecture.md#maintenance-and-recovery-runbook)
for client/account reconciliation, interrupted operations and version restrictions.

[Remaining release gates](docs/roadmap.md#remaining-bounded-work) are finite:
native Linux/image execution and image-specific provenance/notices, QoAuth's own
source-license choice, and real installation/proxy review. Basic accessibility,
local load and a compatible M2.2-to-current restore drill passed within recorded
limits. Full OIDF certification, broad capacity claims and a comprehensive
accessibility/security audit are not claimed. No further feature stage is scheduled.

## Review package

- [Single execution checklist and final evidence](docs/plan/2026-10-25-qoauth-overall.md)
- [Product scope](docs/product.md), [roadmap](docs/roadmap.md), [architecture and operations](docs/architecture.md)
- [Threat model and remaining release gates](docs/security.md)
- [Dependencies, advisories and license obligations](docs/dependencies.md)
- [Runnable standard Go RP and API integration](docs/integration.md)
- [Competitor matrix](docs/research/comparison.md), [primary source ledger](docs/research/sources.md)
- [Authorization history](docs/project-charter.md)
