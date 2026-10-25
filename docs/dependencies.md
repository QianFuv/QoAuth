# Dependency choices and license risks

Verified: 2026-10-25. A5/A6 keep the tested Go module graph and npm lockfile. Dependencies
were downloaded into project-local caches/node_modules; no system toolchain was
installed or upgraded. The shortlist retains alternatives as research, not installed
requirements. Only the selected set below is part of the current implementation.

Current scope: personal/small-team single-instance service; SQLite is the primary
storage. Enterprise connectors and infrastructure are not prerequisites.

## Selected set and final local evidence

| Component | Exact selected version | License / scope |
|---|---|---|
| Go / Node / npm | 1.26.10 / 24.22.1 / 12.2.0 | Existing build tools; inventory tool/bundled licenses before distributing a build environment |
| ZITADEL OIDC / coreos RP | v3.51.16 / v3.21.0 | Apache-2.0; actual OP and independent runnable RP |
| go-jose/v4 / x/oauth2 | v4.1.6 / v0.37.0 | Apache-2.0 / BSD-3-Clause; no separate go-jose/v3 in resolved graph |
| modernc SQLite / libc | v1.61.0 / v1.79.0 | BSD-3-Clause and bundled notices; libc matches the driver's selected dependency |
| x/crypto / pquerna/otp | v0.58.0 / v1.5.0 | BSD-3-Clause / Apache-2.0 with NOTICE; Argon2id and TOTP |
| x/sys / x/text | v0.49.0 / v0.43.0 | BSD-3-Clause; Windows ACLs and password normalization |
| boombuler/barcode | v1.0.1-0.20190219062509-6c824513bacc | MIT; OTP transitive dependency |
| Embedded SQLite engine | 3.54.1 | Public domain engine; actual runtime checked on Windows and Linux |
| React / React DOM | 19.3.1 / 19.3.1 | MIT |
| Semi UI / Vite / TypeScript | 2.104.0 / 8.3.5 / 5.9.3 | MIT / MIT / Apache-2.0 |
| React / React DOM type packages | 19.2.2 / 19.2.2 | MIT |
| govulncheck | v1.9.0 | BSD-3-Clause; verification tool, not linked into product |

Exact graph/checksums live in [go.mod](../go.mod), [go.sum](../go.sum) and
[package-lock.json](../web/package-lock.json). `go list -m all` was recorded locally.
The sources linked below and downloaded package license files govern actual terms;
this table is not complete distribution clearance. Password/TOTP are now selected;
passkeys remain uninstalled. OTP's [NOTICE](https://github.com/pquerna/otp/blob/v1.5.0/NOTICE)
and [license](https://github.com/pquerna/otp/blob/v1.5.0/LICENSE), plus
[x/crypto license](https://github.com/golang/crypto/blob/v0.58.0/LICENSE), must be
preserved in distribution records. x/crypto v0.58.0 requires Go 1.26.0; no toolchain
upgrade was needed. OTP v1.5.0 includes the upstream complete-random-read correction.

Final scoped Go tests/vet/race and both amd64 builds passed. `npm ci --ignore-scripts`
followed by the TypeScript/Vite build passed; the full npm audit reported zero
findings. govulncheck v1.9.0 reported zero affected symbols/imported packages in
source and final Windows/Linux binaries, with one non-imported module advisory:
[GO-2026-5932](https://pkg.go.dev/vuln/GO-2026-5932) affects unmaintained OpenPGP,
not the used Argon2id package. Do not call the whole module vulnerability-free.
Initial stripped binaries triggered a conservative module-level report. The
[tagged scanner](https://github.com/golang/vuln/blob/v1.9.0/internal/vulncheck/binary.go)
explains its no-symbol fallback. Build settings now retain symbols (`-w`, no `-s`);
both actual final artifacts pass the symbol scan. Details and runtime limitations
are in the [checklist](plan/2026-10-25-qoauth-overall.md#final-bounded-delivery-evidence-2026-10-25).
A5 added no dependency and reran source/both-final-binary scans plus npm audit on
2026-10-25. This is not a security verdict.

The selected OP version's actual source still defaults to a legacy AES-CFB
fallback. M1 explicitly supplies `op.NewAES256GCMCrypto` without a legacy decrypter;
tests reject both modified opaque tokens and tokens re-encoded with the legacy
format. The library's broad default discovery is also restricted to the enabled
code/refresh/S256 profile. Source: [tagged provider](https://github.com/zitadel/oidc/blob/v3.51.16/pkg/op/op.go),
[opaque-token advisory](https://github.com/zitadel/oidc/security/advisories/GHSA-j8gq-92xf-382c).
These are application integration obligations, not reasons to invent cryptography.

Semi's deep Button import and React 19 adapter build under the intended CSP without
the unused full-package animation dependency's eval warning. Final bundle: about
278.09 kB JavaScript and 52.99 kB CSS before gzip (85.41/9.07 kB compressed).
Chrome exercised the real temporary login/consent/Go RP/logout and administrator
paths, including reset and lossless disabled-client review/replacement. Full
accessibility audit remains outstanding. Native Linux and final-container synthetic
runs passed under A6; the actual Linux-built frontend inventory and embedded bytes
match the earlier verified assets.

## Artifact inventories and preserved notices

Final local version `0.1.0-dev.826200d48959` has separate Windows/Linux SPDX 2.3 JSON,
build metadata, original license files, notice index and SHA-256 manifests. Each
SBOM passes the official [SPDX 2.3.1 JSON schema](https://raw.githubusercontent.com/spdx/spdx-spec/v2.3.1/schemas/spdx-schema.json)
with PowerShell Test-Json. Schema validity is not legal clearance. SPDX metadata's
CC0-1.0 does not choose a license for QoAuth.

| Evidence scope | Windows amd64 | Linux amd64 |
|---|---|---|
| Modules actually recorded by `go version -m -json` | 30 | 29 |
| SPDX packages including QoAuth, Go and frontend supplements | 43 | 42 |
| Original-file/inline notice records | 60 | 59 |
| Frontend components including CSS/helper supplements | 11 | 11 |

The Vite [build.license](https://vite.dev/config/build-options.html#build-license)
JSON uses emitted chunk provenance. The packager adds Semi theme CSS and Vite's
generated modulepreload helper; it does not claim that full Vite runs in the browser.
It compares embedded HTML/JS/CSS bytes to the actual executable. The dot-directory
inventory is build evidence, not a publicly served UI route. No CSS URL assets were
present. Official schema/docs and installed Vite 8.3.5 source were checked 2026-10-25.

Preserved supplements include OTP/Zitadel NOTICE, JOSE json/LICENSE, SQLite/libc
third-party aggregates, SQLite public-domain text, three attributed libc inline
notices (Luke Champine, University of California, Poul-Henning Kamp), modernc memory
notices, all four Semi packages' complete LICENSE files, and Go LICENSE/PATENTS/vendor
notices. Aggregates can mention unused portions: sqlite-vec and memory logos are
not thereby claimed linked or licensed for logo redistribution. Go h1 values remain
module-integrity evidence, not mislabeled archive SHA-256 values.

The separate build/development inventory records 273 npm entries (248 installed,
25 optional/uninstalled) and 66 resolved Go dependency modules. Coreos belongs to
RP/example/test scope and is absent from the product binaries. Runtime sets differ:
Linux includes google/uuid; Windows includes go-isatty/go-strftime. Test-only libc
GPL material and build-only MPL components are not promoted to runtime dependencies.
Root build notices include Vite/Rolldown/TypeScript and relevant package copyright/
third-party files. The installed Rolldown Windows binding declares MIT but has no
root license file; that gap is explicit, and build executables are not redistributed.

The source ZIP is the exact clean commit; explicit dirty-preview mode omits it.
A6 separately packages the actual Linux image at source `bcee155a2fcf`, with 44 SPDX
packages and 62 notice records. The extracted image executable has the same 29 Go
modules and Go 1.26.10 toolchain; Linux-built Vite inventory and all embedded asset
bytes match. Original Go/frontend notices are reused only after those comparisons.
Its actual binary govulncheck scan reports zero affected symbols/imported packages,
with the same unused OpenPGP module-only advisory. The Docker build excludes .git;
its revision is bound by image label/BuildKit provenance, not invented binary VCS data.

The scratch image additionally ships only `/etc/ssl/certs/ca-certificates.crt` from
Alpine 3.24.3 package `ca-certificates-bundle 20261020-r0`, declared MPL-2.0 AND MIT.
The [pinned Alpine package recipe](https://raw.githubusercontent.com/alpinelinux/aports/e78f5987166c6ffbe23f6472e2e8f040298fe2da/main/ca-certificates/APKBUILD)
provides the source SHA-512. The verified source archive contains every shipped DER
certificate (118 in the bundle, 169 in certdata source). The image bundle preserves
certdata source, package attribution, [MPL-2.0](https://www.mozilla.org/media/MPL/2.0/index.txt)
and MIT text, source URL/hash and supplying base-image digests. No Alpine package
manager, CA-management utilities or source-only GPL build tool is claimed in runtime.

Image SPDX passes the official schema; archive/config-ID, all manifest/notice hashes
and outer ZIP integrity were checked. The notices accompany the docker-save archive;
they are not copied inside the image. Preserve the full companion bundle. QoAuth's
own license remains NOASSERTION pending a maintainer choice; no LICENSE file or
release permission has been invented.

## Protocol and storage shortlist

| Dependency | Snapshot / license source | Proposed use and limits |
|---|---|---|
| `github.com/zitadel/oidc/v3` | v3.51.16; [Apache-2.0](https://raw.githubusercontent.com/zitadel/oidc/v3.51.16/LICENSE) | First OP spike: protocol endpoint/metadata handling. Application supplies identity, consent, persistent sessions, transactional storage and key custody |
| `github.com/ory/fosite` | v0.49.0; [Apache-2.0](https://raw.githubusercontent.com/ory/fosite/v0.49.0/LICENSE) | Alternative composable OAuth/OIDC framework; greater session/prompt/max_age wiring burden and unresolved v3 JOSE dependency concern |
| `github.com/coreos/go-oidc/v3` | v3.21.0; [Apache-2.0](https://raw.githubusercontent.com/coreos/go-oidc/v3.21.0/LICENSE) | RP discovery/ID Token verification for the Go example; not an OP implementation |
| `golang.org/x/oauth2` | Go subrepository; [BSD-3-Clause](https://github.com/golang/oauth2/blob/master/LICENSE) | Client-side OAuth exchange and transport, coupled to OIDC verification; not a standalone login or server implementation |
| `github.com/go-jose/go-jose/v4` | v4.1.6; [Apache-2.0](https://raw.githubusercontent.com/go-jose/go-jose/v4.1.6/LICENSE) | JOSE primitive library; issuer/audience/purpose/algorithm policy remains application responsibility |
| `github.com/go-webauthn/webauthn` | v0.18.2; [BSD-3-Clause](https://raw.githubusercontent.com/go-webauthn/webauthn/v0.18.2/LICENSE) | Optional future passkeys only; do not add until demanded and tested |
| `golang.org/x/crypto/argon2` | module v0.58.0; [BSD-3-Clause](https://raw.githubusercontent.com/golang/crypto/v0.58.0/LICENSE) | Argon2id primitive; salt/format/parameter bounds/constant-time comparison/rehash policy still required |
| `github.com/pquerna/otp` | v1.5.0; [Apache-2.0](https://github.com/pquerna/otp/blob/v1.5.0/LICENSE) + NOTICE | Selected TOTP/HOTP primitives; actual matched counter, enrollment confirmation and replay state remain transactional application responsibilities |
| `modernc.org/sqlite` | v1.61.0 observed; [BSD-3-Clause](https://gitlab.com/cznic/sqlite/-/raw/v1.61.0/LICENSE) | Primary single-instance candidate; CGo-free Windows/Linux amd64 documented; verify actual SQLite engine, libc coupling and atomic transactions |
| `github.com/mattn/go-sqlite3` | v1.14.52; [MIT](https://raw.githubusercontent.com/mattn/go-sqlite3/v1.14.52/LICENSE) | Not preferred: CGo and compiler/cross-toolchain requirements complicate requested artifacts |

Tagged repository files were inspected for the main Go candidates. Release IDs
are an observation, not an assertion that no subsequent release exists.
SQLite's engine is [public domain](https://www.sqlite.org/copyright.html), while
drivers and bundled code have separate terms. Verify the runtime SQLite version
contains the WAL-reset fix (3.51.3 or applicable backport); do not infer that from
the Go driver version alone. [SQLite WAL notice](https://www.sqlite.org/wal.html).

## Material security findings before selection

The inspected ZITADEL OIDC release
[declares go-jose/v4 v4.1.4](https://github.com/zitadel/oidc/blob/v3.51.16/go.mod).
Its upstream JOSE advisories identify RSA JWK CPU exhaustion fixed in v4.1.6 and
deeply nested JSON crashes fixed in v4.1.5. The Fosite snapshot
[declares go-jose/v3 v3.0.3](https://github.com/ory/fosite/blob/v0.49.0/go.mod);
the same advisories list affected v3 releases with no patched v3 release.
Adding v4 does not replace a separate v3 module path.
Sources: [JWK advisory](https://github.com/go-jose/go-jose/security/advisories/GHSA-mgcc-cww5-fp2q),
[JSON advisory](https://github.com/go-jose/go-jose/security/advisories/GHSA-vvm8-gf78-28g4).

Therefore prefer the ZITADEL OP spike with a verified compatible patched dependency
graph. M1 must record `go list -m all`, scan source and binary with `govulncheck`,
test the actual parsing boundaries, and reject or remediate reachable affected
paths. Merely choosing a version with a fixed advisory is not a complete security
review. No reachability test was executed in M0.

ZITADEL OIDC also fixed an opaque-token forgery issue in v3.47.0. Preserve the
new secure token behavior; do not enable legacy unauthenticated-token acceptance
for a new product. Regression tests must cover modified token bytes and subject
substitution. [Maintainer advisory](https://github.com/zitadel/oidc/security/advisories/GHSA-j8gq-92xf-382c).

The ZITADEL library README identifies its **RP** Basic/Config certification;
that is not OP certification for QoAuth. Its OP storage and lifecycle contract
must be tested independently. Fosite also leaves important interactive OIDC
behavior to its integrator.
[ZITADEL README](https://github.com/zitadel/oidc/blob/v3.51.16/README.md),
[OP storage](https://github.com/zitadel/oidc/blob/v3.51.16/pkg/op/storage.go),
[Fosite README](https://github.com/ory/fosite/blob/v0.49.0/README.md).

## Frontend and build recommendation

Use React, TypeScript, Vite, and Semi Design consistently for QoAuth, with npm and
a committed lockfile to align with the inspected Exta browser application. Do not
reuse Yousage's Wails/mobile runtime or change either neighboring project.

| Component | Observed license | Selection consideration |
|---|---|---|
| React 19.3.1 | [MIT](https://raw.githubusercontent.com/facebook/react/v19.3.1/LICENSE) | Matches the local browser stack; no need for server-side React runtime |
| Vite 8.3.5 | [MIT](https://raw.githubusercontent.com/vitejs/vite/v8.3.5/LICENSE) | Static production build; embedding must be reproducible from locked inputs |
| TypeScript 7.0.2 upstream snapshot | [Apache-2.0](https://raw.githubusercontent.com/microsoft/TypeScript/v7.0.2/LICENSE.txt) | Local Exta is on 5.9.3; do not upgrade blindly or assume Semi/plugin/tooling compatibility |
| Semi Design | [MIT and bundled notices](https://github.com/DouyinFE/semi-design/blob/main/LICENSE) | Prefer for admin tables/forms; validate CSP, keyboard support and login bundle size; pin compatible version in M1 |
| Go standard library | [BSD-3-Clause](https://go.dev/LICENSE) | `net/http`, `embed`, `crypto/rand`, standard cryptography and TLS rather than bespoke implementations |

Choose versions as a tested set during M1; the newer TypeScript snapshot is not a
required upgrade. The inspected ZITADEL release requires Go 1.26.0; the WebAuthn
snapshot additionally declares a go1.27.1 toolchain. Inspect target `go.mod` files
before accepting a toolchain, and do not let auto-download silently change the
system environment. [ZITADEL module](https://github.com/zitadel/oidc/blob/v3.51.16/go.mod),
[WebAuthn module](https://github.com/go-webauthn/webauthn/blob/v0.18.2/go.mod).

TOTP uses pinned `pquerna/otp` v1.5.0; its actual matching counter is persisted
atomically with session elevation. No OTP algorithm was written from scratch. Mail transport
is optional; migrations are small ordered SQL files, not a database portability
framework. SAML/SCIM dependencies are outside scope. Prefer a small graph; evaluate
maintenance, advisories, supported Go/OS versions, transaction semantics, license
and negative tests, not only stars or download counts.

## Test tooling and distribution license boundary

The OpenID Foundation conformance suite is
[MIT licensed](https://raw.githubusercontent.com/openid-certification/conformance-suite/master/LICENSE.txt).
Its local stack includes other components, including MongoDB with
[SSPL terms](https://www.mongodb.com/legal/licensing/server-side-public-license).
Keep the harness separate from product artifacts and review the pinned stack's
licenses. Local testing is not certification, and hosted tests, account creation,
public callbacks or certification fees are not authorized by this research task.
[Official OP testing process](https://openid.net/certification/connect_op_testing/),
[local build/run guide](https://gitlab.com/openid/conformance-suite/-/wikis/Developers/Build-%26-Run).

Before distributing any artifact, inventory the resolved backend/frontend/build
and container graphs, record SPDX identifiers and exceptions, preserve copyright
and required notices, and generate an SBOM and license bundle. Treat fonts/icons,
copied snippets, container packages and test fixtures as dependencies too. Apache
NOTICE obligations and copyleft/network-copyleft terms need component-specific
review; an MIT top-level frontend does not clear its entire dependency graph.

No QoAuth source license is selected yet; no LICENSE file has been invented.
Recommend considering Apache-2.0 for QoAuth's own original code, subject to the
user's distribution decision and compatibility review. This is a licensing risk
assessment, not legal clearance. Comparing an AGPL product does not require importing
its code; a separately Apache-licensed SDK must be evaluated by its own license.
