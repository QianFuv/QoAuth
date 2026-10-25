# QoAuth

QoAuth is a lightweight self-hosted identity service for a person or small team.
Here “OA” means unified identity authentication, not office automation.
One Go process contains a React/TypeScript/Vite/Semi UI and uses one local SQLite
database and one issuer. No external database, Redis, Node runtime or enterprise
tenant/SCIM/LDAP/SAML infrastructure is required.

Version `0.1.0` includes local users, password/TOTP/recovery, sessions,
applications/scopes/consent, OIDC code+S256 and a simplified-Chinese Semi UI.
Maintenance includes installation locking, consistent backup, isolated restore,
signing rotation and bounded cleanup. This is not an OpenID certification or a
production deployment claim.

## Release and checks

[GitHub Releases](https://github.com/QianFuv/QoAuth/releases) provides
`qoauth_0.1.0_windows_amd64.zip`, `qoauth_0.1.0_linux_amd64.tar.gz` and
`SHA256SUMS`. Verify the downloaded archive before use. Each platform archive
contains the executable, SPDX inventory, original third-party/font notices and
source/build/file manifests. GitHub supplies source archives from the release tag.
QoAuth has no selected source license; publication by its maintainer does not
supply an additional source-license grant. Preserve all dependency notices.

Use Go 1.26.10, Node 24.22.1 and the locked npm dependencies; local packaging uses
Python 3.12. Run these checks from the repository:

```powershell
Push-Location web
npm ci --ignore-scripts
npm test
npm run build
Pop-Location
$env:GOTOOLCHAIN = 'local'
go test -timeout 180s ./cmd/... ./internal/... ./examples/...
go vet ./cmd/... ./internal/... ./examples/...
node --test scripts/release.test.mjs
python scripts/package-release.test.py
```

With an existing C compiler, also run the same Go package set with
`CGO_ENABLED=1` and `go test -timeout 180s -race`. Explicit package paths avoid
traversing node_modules. `node scripts/browser-check.mjs <new-output-directory>`
uses installed Chromium and fresh synthetic state to verify real authentication,
360/390/768/1440 layouts, network fonts with local fonts disabled, and blocked-font
fallback. It removes its temporary credentials and profiles and reports no tokens.

From a clean committed checkout, `scripts/build.ps1 -UseInstalledFrontend -Release`
builds both amd64 executables and formal platform archives under ignored `dist/`.
The output directory must be new; use `-Output <new-path>` when repeating a build.
Formal packaging rejects dirty source. Local preview packaging keeps an explicit
development version; `--allow-dirty` never produces a formal release.

The [CI workflow](.github/workflows/ci.yaml) follows four stages: detect-changes,
quality, build-acceptance, publish. Reusable frontend/backend/release workflows
retain native Go/race, real-browser and Linux container gates. The frontend is
built and tested once, then transferred by immutable artifact ID and a complete
source-bound hash inventory. Windows acceptance runs the same packaged bytes.
The Linux image is an acceptance fixture, not a published registry artifact.

A push to main builds and checks only. After reviewing its successful CI run and
all accepted archives, explicitly dispatch CI with the exact `version` and its
`source_run` ID. Publication requires that run's complete same-SHA acceptance,
rechecks all bytes and creates a new tag/Release without rebuilding. Existing
tags/releases are never overwritten; an interrupted publication requires inspection
before any retry. To rebuild accepted artifacts, start a new build-only dispatch
rather than overwriting immutable artifacts in an existing run. Public downloads are rechecked after publication.

Only code and formal platform artifacts are published. Installation state, keys,
configuration, caches and test credentials are not release contents. No service
is deployed. See [font provenance](docs/font-licensing.md),
[dependency notices](docs/dependencies.md) and [remaining limits](docs/roadmap.md).

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

The embedded Semi UI serves a static product homepage at `/` (no API calls or
installation data), sign-in, MFA and consent at `/login`, self-service at `/account`
and administration at `/admin`. Pages are code-split; the interface uses simplified Chinese.

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
and persistent private storage are still required. When overriding the serve command,
retain `--data-dir /data/state` and configure `--trusted-proxy` with the actual
container-visible proxy peer. Compose does not guess that address; it may differ
from host loopback. The proxy must overwrite `X-Real-IP` and preserve canonical Host.
Real proxy/TLS/network changes and deployment need their own authorization.

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

[Remaining limits](docs/roadmap.md#remaining-bounded-work) include the unselected
QoAuth source license and a separate real installation/proxy review. Native Linux
and final-image synthetic execution plus image-specific provenance/notices now pass.
Basic accessibility, local load and a compatible M2.2-to-current restore drill passed within recorded
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
