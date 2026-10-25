# Focused threat model, tests and risks

Review revision: 4, 2026-10-25. Single installation, local users and clients.
A4 authorized M2.2; 127 scoped Windows race tests/subtests, temporary browser flows
and final-binary maintenance/restart have executed. The [checklist](plan/2026-10-25-qoauth-overall.md#final-m22-evidence-2026-10-25)
records exact evidence. The full matrix below remains a release contract, not a
claim that every fuzz/conformance/platform gate has passed.

## Assets and boundaries

Protect credentials/MFA/recovery data, signing keys, sessions/tokens, client
permissions, user information, activity records and backups. Threats include an
unauthenticated attacker, malicious registered client, compromised browser/app,
leaked backup, and dependency/parser abuse. The host administrator controls this
installation; no hostile-tenant or privileged-host isolation product is promised.

Trust boundaries: browser-to-HTTPS; proxy-to-canonical-issuer; untrusted protocol
input-to-library; regular user-to-admin API; client A-to-client B; process-to-SQLite/
key files; backup-to-restore; dependencies-to-artifact. Optional future federation
adds its own SSRF and account-linking boundary, outside current implementation.

## Required regression matrix

| ID | Threat | Observable test / control | Acceptance |
|---|---|---|---|
| S01 | Intercepted code, login CSRF, PKCE downgrade | Reject wrong/missing verifier, stripped challenge, wrong state/nonce and replay; require S256 | F1, A3 |
| S02 | Redirect takeover / issuer substitution | Exact registered redirects; reject forged Host/Forwarded, callback tampering and invalid-client redirects | A3 |
| S03 | Cross-client token or scope escalation | Client B cannot redeem A's code/refresh token; resource audience/scope required; scope strings never create admin rights | A3, A4 |
| S04 | Token forgery / confusion | Reject bad signature/algorithm/iss/aud/exp and wrong token purpose; modified opaque token cannot change identity | A3, R2 |
| S05 | Concurrent one-use token abuse | Atomic code/refresh/recovery consumption; injected transaction failure leaves no partial grant; reuse revocation persists on error | F2, A7 |
| S06 | Logout/revocation overpromise | Disable/recovery invalidate sessions and pending codes as well as tokens; assignment/consent removal invalidates matching grants/codes; code/refresh/UserInfo/introspection recheck current state; independent RP cookies may survive | A5, A6 |
| S07 | Password guessing / enumeration / resource exhaustion | Account+network throttles, bounded Argon2 budget, consistent errors and timing review, no attacker-triggered permanent lockout | A1, R6 |
| S08 | MFA/reset bypass and privileged bootstrap | Unenrolled/promoted admin cannot use privileged APIs; reset preserves MFA; fresh auth for factor changes; one-use codes/TOTP replay protection | A2 |
| S09 | XSS/CSRF/clickjacking and leaked secrets | Escape user content, CSP, CSRF/origin checks, secure cookies, no framing/localStorage tokens; planted canaries absent from logs/errors | A8 |
| S10 | Library/parser denial of service | Patched graph, bounded bodies/nesting/key sizes/timeouts, fuzzed JOSE and known advisory regression cases | F4, R2 |
| S11 | Snapshot rollback revives access | Invalidate grants/one-time credentials; reconcile user/password/MFA and client-secret/callback/assignment/scope/key state before serving; disable uncertain clients/keys | R4 |
| S12 | Key loss/rotation, disk full/locked DB, migration failure | Preserve keys across restart; verify old/new key overlap, missing-key failure, safe disk/transaction errors and isolated restore | R3, R4, R6 |

Tests must target the business/security invariant, not only reproduce a library
call. Demonstrate a failing case using isolated task-owned fixtures or mutation
tests when practical; never discard user work to manufacture a before-state.

## Password, MFA and recovery decisions

Salted Argon2id with encoded/versioned parameters; OWASP's current minimum
19 MiB/two iterations/parallelism one is a floor to benchmark, not a universal
latency setting. Bound request and hashing concurrency. Support password managers,
paste and long passwords; propose 15-character minimum for password-only login,
at least 64-character support, a local common-password blocklist, no forced
composition or routine expiry without compromise.
[OWASP storage](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html),
[NIST SP 800-63B-4](https://pages.nist.gov/800-63-4/sp800-63b.html), verified 2026-10-25.

Admin MFA must be enforced server-side even for an existing session after promotion.
Initial owner setup allows only enrollment until MFA succeeds. No shipped/default
credential. TOTP setup requires confirmation; stored seeds are encrypted; accepted
time-window reuse is rejected. Removing/changing a factor requires recent proof.
Recovery codes are random, hashed, single-use and transactionally consumed.

Password reset must not remove MFA, auto-login with full assurance, or let someone
replace a factor using only a reset email. Without SMTP, provide an owner-assisted
procedure and recovery codes. A sole-admin break-glass procedure must require
local host control, explicit confirmation, session/grant invalidation and an activity
record, then force MFA re-enrollment before privileged use. It is not a public API
backdoor. Implemented command: `recover-owner --data-dir <private-directory> --confirm`,
reading a replacement password from redirected stdin. All execution used synthetic
temporary state. Initialization requires local host control, cannot reopen an
initialized database, and refuses a missing database beside existing keys.
[OWASP recovery](https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html).

## Verification strategy

Unit/integration tests cover S01-S12, SQLite concurrency and failure injection.
Go checks include scoped tests/vet/format, race tests where shared state exists,
and govulncheck on source and binaries. Browser tests cover real login/admin/
recovery with CSRF/CSP/cookie/keyboard checks. Run local OpenID Basic OP/Config OP
and supported RP-initiated logout profiles using synthetic users. Test against two
independent RP implementations; a library certification badge is not QoAuth
certification. Retain exact versions, commands, results and unresolved failures.
[OpenID OP testing](https://openid.net/certification/connect_op_testing/),
[Go security guidance](https://go.dev/doc/security/best-practices).

Use a separate local conformance harness after approval, not real SaaS accounts,
public callbacks or production credentials. The suite has separate runtime and
license needs; do not install Docker/Java/databases or alter the host environment
under a documentation authorization. Native OS/image smoke and backup/rotation
tests remain mandatory for their deliverable, not replaced by cross-compilation.

## Small risk register

| Risk | Priority | Minimal response / owner |
|---|---|---|
| OIDC dependency advisories/storage callback gaps | P0 | Engineering: F1/F2/F4 before accepting library; no claim that latest implies safe |
| Lost or replayed refresh response | P0 | Client guide: serialize refresh, reauthenticate after ambiguity; server: atomic lineage and durable family revoke |
| Weak recovery defeats MFA | P0 | Security review: bootstrap, reset and sole-owner recovery tested together |
| SQLite snapshot or durability error restores access | P0 | Engineering/operator: verified settings/version, consistent backup, maintenance-mode recovery and reconciliation |
| RP cookies survive provider logout | P1 | Integration: explicit boundaries and negative demonstration; backchannel remains optional |
| UI/toolchain incompatibility | P1 | Frontend: compatible frozen set, CSP/keyboard spike; no automatic major upgrade |
| Distribution/license obligations | P1 | Maintainer: pinned graph, notices/SBOM; choose QoAuth's license before distribution |
| Missing Linux/container test environment | P1 | Build owner: report missing native checks; request only concrete necessary setup later |
| Scope creep | P1 | Product: required/optional/excluded table controls; no enterprise backlog by default |

Before a real-use candidate, perform a focused independent review of protocol,
recovery, client isolation and storage invariants; resolve critical/high exploitable
findings and record other residual risks. No blanket “secure production ready”
statement follows from this planning document.

## M2.1 review closure and remaining gates

Two independent reviews at 0503b9d found seven P1 items. Corrections reject duplicate
protocol values and POST callback confusion, recheck silent consent atomically,
separate proxy client rate budgets, bound/prune anonymous state, validate trusted
file ownership, and validate existing SQLite sidecars. A second review at 9cf4784
found cleanup rollback on denied admission; regressions and a correction now commit
cleanup while retaining atomic quotas. These were static reviews; the implementation agent ran
the tests. They do not constitute certification or a claim of exhaustive audit.

Passwords: strict Argon2id v19 m=19456 KiB/t=2/p=1, 16-byte salt/32-byte verifier,
15-rune minimum, 1024-byte maximum and four concurrent hash slots. The local common-
password blocklist is intentionally small and needs coverage/performance assessment
before real use. TOTP uses six digits/SHA-1/30 seconds, current counter +/-1, with
monotonic replay prevention including enrollment. Recovery codes are 256-bit random
handles stored as hashes. Recovery retires the whole old factor/code set and gives
only replacement-enrollment authority; password reset preserves enrolled MFA.

Cookie handles rotate on password/MFA authentication. HTTPS uses host-only
__Host-/Secure/HttpOnly/Lax cookies; the explicit HTTP test mode is loopback-only.
Browser mutations require a session CSRF proof and exact configured Origin.
Native logout forms use strict-origin Referrer-Policy and only validated CSP redirect
origins; null/foreign origins remain rejected. OAuth token endpoints use their own
client/PKCE checks rather than browser cookies. Tokens and full request bodies are
not logged; protocol logs redact arbitrary attributes and messages.

Remaining real-use gates: second independent RP implementation and selected local
OIDF conformance profiles; native Linux/image execution; cross-version restore/upgrade,
reference load/password-cost measurements, comprehensive accessibility and distribution
notices/SBOM. M2.2 covers same-version backup/restore, rotation, retention and synthetic
disk-full/transaction failures on Windows.
Private key material is held in a protected local file, not hardware-backed storage;
host administrators remain trusted. No blanket production-ready claim is warranted.

## M2.2 maintenance review closure

Independent code review found two P1 issues and one P2 interruption problem: SQL
LIKE hid reserved-prefix-lookalike triggers from exact schema inspection; in-flight
requests could carry client settings removed during replacement; partially written
rotation candidates blocked an intact installation. All three were reproduced in
isolated tests and corrected. A follow-up static review at d77ac9b found no remaining
P0/P1/P2 in those corrections and immediate lifecycle callers. A separate cleanup
review found no P0/P1/P2 and recommended additional post-revocation/CLI coverage,
which is now included. Reviews did not independently run tests or certify the system.

Snapshot integrity includes exact real schema and ledger, FK/integrity checks,
private fixed members, RSA/AES binding and owner invariants. Restore creates a new
directory behind an incomplete marker, discards authentication/recovery state,
disables accounts/clients, replaces signing trust and persists maintenance/quarantine.
Owner and ordinary-account recovery replace credentials/factors; clients require
reviewed re-registration and new confidential credentials. No unseen post-snapshot
account/permission change is presumed known. See the [runbook](architecture.md#maintenance-and-recovery-runbook).

The plaintext key file and complete snapshots must remain private. SHA manifests
are not authenticity against a backup custodian who can replace every member.
Private-file checks reject unsafe owners/permissions, links/reparse traversal and
hard-link aliases; trusted host-controlled parents remain required. They do not
isolate the process from a privileged host attacker or prove race-free hostile-host
filesystem access. Only task-owned temporary files were restricted during tests.

Old signed ID Tokens are not remotely revoked by deleting JWKS. Tests demonstrate
warm-RP cached-key acceptance before expiry, cold-RP rejection after restore and
expiry rejection afterward. Restore therefore requires old issuance stopped and
kept stopped, fresh signing trust, and ten minutes of quarantine (five-minute token
life plus supported skew up to five minutes). Independent RP cookies require their
own reconciliation. Ordinary rotation retains only the preceding public key until
its persisted deadline; AES is unchanged. Process-exit tests are not power-cut tests.

Cleanup preserves logical session/request parents and replay lineage through all
family/token deadlines, in bounded child-first transactions. Still-valid replay
revokes the family even across cleanup/restart; expired ancestors are rejected
without revoking a live successor. Neither retained rows nor passing tests imply
family-lifetime detection of every expired stolen credential.
