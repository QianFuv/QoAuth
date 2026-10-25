# Standard OIDC integration and local examples

Updated/verified: 2026-10-25. M2.1 provides a real OP/account adapter and a runnable
Go RP using coreos/go-oidc v3.21.0 with x/oauth2 v0.37.0. The production server never
imports `internal/fixture`. All executed flows used temporary synthetic state;
no SaaS account, SMTP or third-party authorization was connected.

## Runnable Go web RP

After explicit owner initialization and MFA enrollment, register a server-side client
in Administration. Example settings use reserved domains, not a live installation:

| Setting | Value |
|---|---|
| Client ID / type | example-go-web / server-side web client |
| Exact callback | https://app.example.test/callback |
| Exact signed-out return | https://app.example.test/signed-out |
| Allowed scopes | openid, profile, offline_access |
| Resource audiences | Empty for login-only; assign explicitly for an API |

Copy the generated secret once into private server-side configuration. Assign the
user to the client. The interface does not support wildcard callbacks, anonymous
registration, email claims or enterprise connectors. The current library example
uses these exact callback paths, not /auth/callback.

```text
go run ./examples/go-rp --issuer https://auth.example.test   --client-id example-go-web --client-secret-file <private-secret-file>   --callback https://app.example.test/callback --listen 127.0.0.1:8081
```

This is command syntax for a configured environment, not authorization to create
real credentials or change a proxy. Explicit synthetic loopback testing adds
`--development-http` and numeric loopback HTTP issuer/callback/listener addresses.
The example uses in-memory app sessions and is a teaching integration, not a
production session store or private authentication SDK.

1. Discover the canonical issuer and standard endpoint metadata. Backend creates
   browser-bound, expiring, one-use state/nonce/PKCE values and sends S256.
2. Request code with openid/profile/offline_access and explicit consent. Ordinary
   login may omit offline_access and must not expect a refresh token. Registration
   must allow every requested scope; consent cannot expand the client's permissions.
3. The callback consumes state once, exchanges code with its verifier and client
   authentication, verifies the RS256 ID-token signature/issuer/audience/expiry,
   and compares nonce. `(iss, sub)` identifies the user; an ID Token is not API access.
4. App tokens remain server-side. HTTPS cookies are __Host-/Secure/HttpOnly/Lax.
   /profile serializes refresh, checks UserInfo sub against the ID-token subject,
   and discards the app session when token validation or the subject check fails.
5. POST /logout requires app CSRF and canonical Origin, clears the app session and
   initiates standard OP logout. QoAuth presents a CSRF-protected confirmation, then
   revokes that browser session/grants and returns only to its registered URI.
   The example does not send logout state; clients that send one must verify it.

The example's app session expires after 15 minutes. Independent application cookies
can survive QoAuth logout unless the app cooperates; /profile demonstrates online
revalidation. Browser-cookie expiry does not itself revoke an offline refresh grant;
explicit logout/security changes do. Refresh is serialized; an ambiguous committed
response can require signing in again. No replay grace window is promised.

## API resource identity and scopes

Register a Resource API validator named `notes-api`, with audience `notes-api` and
scope `notes.read`. The small implemented model uses the resource client's ID as
its audience; it has its own confidential credential and no browser callback.
Register web-rp separately, allow notes.read and the notes-api audience, assign the
user and obtain consent. ID-token audience remains web-rp; access-token introspection
audience is notes-api. The RP credential is not allowed to inspect other resources.

An API authenticates with client_secret_basic to the discovered introspection
endpoint, then checks active, intended audience, expiry and required scope. Its
own record-level business rules still apply. No positive introspection cache is
provided: an outage, disabled user, revoked grant, wrong audience or missing scope
must deny access. Opaque access tokens are not parsed as JWTs. Client credentials
grant and request-time `resource` selectors are not implemented or advertised.

[resource_test.go](../internal/fixture/resource_test.go) is an independent HTTP API
example testing those checks and timeout denial. It remains synthetic test support;
production introspection/audience boundaries also run in
[server_test.go](../internal/oidc/server_test.go).

## Executed production and RP tests

Build embedded assets before Go, then run:

```sh
go test -timeout 180s ./cmd/... ./internal/... ./examples/...
go test -timeout 180s -race ./cmd/... ./internal/... ./examples/...
```

The production adapter tests owner TOTP, ordinary users/apps, two RP registrations
sharing SSO/auth_time, consent, code/refresh/revoke, exact client/redirect/scope
boundaries, prompt/age behavior, browser callback isolation, security-change
invalidation and database/key reopen. CLI tests include a genuinely separate key
process; a manual native Windows executable restart also preserved JWKS.

The runnable RP tests successful callback, refresh, UserInfo and logout, wrong
state/nonce/subject and CSRF denial. Chrome separately exercised the actual production
adapter, embedded UI and Go RP in one temporary local flow. Backend HTTP tests cover
MFA enrollment/recovery/failure races more extensively than the browser smoke.
Two app registrations use the same independent RP library; this is not evidence of
two different RP implementations or OpenID conformance certification.

The optional [browser harness](../internal/oidc/browser_test.go) runs only when both
QOAUTH_BROWSER_STATE and QOAUTH_BROWSER_RP are set. Point the former to a temporary
state JSON and the latter to the built example executable; run only TestBrowserHarness.
It creates synthetic accounts/keys/clients, starts loopback OP/RP processes, and
stops when the state path plus `.stop` exists or after ten minutes. Its state JSON
contains disposable test credentials and must not be reused for real data. Cleanup
stops the child and removes its private temporary installation.

## M1 executable fixture

The retained [M1 tests](../internal/fixture/integration_test.go) exercise the actual
OP library with synthetic identity callbacks, not a real password/consent service.
Run `go test -timeout 60s ./internal/fixture` after the frontend build. They remain
useful protocol/transaction regressions but do not substitute for M2.1 production
adapter tests or browser-session semantics.

## Remaining compatibility gates

Use an existing OIDC client library in each application. The primary path is a
server-side app/BFF. A pure SPA cannot keep a client secret; its CORS/browser-token
profile needs separate integration validation. Native/device flows, external IdPs,
machine clients and back-channel logout are optional later work, not promised here.
The second independent RP implementation, local OIDF profiles and native Linux/image
runs remain documented release gates. M2.2 tests ordinary signing rotation with the
standard RP: new kid triggers key retrieval; old public keys retire by persisted
deadline; opaque access/refresh remain usable because AES is unchanged. A restore
uses fresh signing trust and invalidates all token DB state. Cached old keys may
still validate an unexpired ID Token, so the stopped-issuer/quarantine and app-session
requirements in the [runbook](architecture.md#maintenance-and-recovery-runbook) apply.
No RP may use an ID Token as an API bearer credential.

Primary protocol sources, verified 2026-10-25:
[OIDC Core](https://openid.net/specs/openid-connect-core-1_0.html),
[OAuth security BCP](https://www.rfc-editor.org/rfc/rfc9700.html),
[RP logout](https://openid.net/specs/openid-connect-rpinitiated-1_0.html),
[introspection](https://www.rfc-editor.org/rfc/rfc7662.html),
[coreos RP](https://github.com/coreos/go-oidc/tree/v3.21.0).
