# Integration contract and simple examples

Updated 2026-10-25. M1 provides executable synthetic tests using standard libraries;
it does not provide a production login server or proprietary SDK. The configuration
examples below remain a target contract with reserved domains. No real users,
credentials or external bindings were created. Endpoint locations are discovered
from metadata in real integrations; fixture assertions may pin the tested library's
known paths to catch regressions.

## M1 executable fixture

After building the frontend as shown in [README](../README.md), run:

```sh
go test -timeout 60s -v ./internal/fixture
```

[integration_test.go](../internal/fixture/integration_test.go) uses coreos/go-oidc
v3.21.0 and x/oauth2 v0.37.0 against the actual ZITADEL OP v3.51.16 and SQLite
adapter. It visits the embedded login HTML, submits a synthetic login transaction,
checks state/nonce, verifies the signed RS256 ID Token, reads UserInfo, refreshes
with consent, introspects, revokes and checks RP-initiated logout. The OP uses a
canonical loopback test issuer; insecure HTTP is fixture-only.

[resource_test.go](../internal/fixture/resource_test.go) runs an independent HTTP
API. Its resource credential is distinct from the requesting RP. It checks active,
audience and scope; valid access succeeds, revoked/wrong-audience/missing-scope
access fails, and an introspection timeout denies access. There is no positive
cache in this example. It is an executable compatibility example, not a complete
middleware SDK or production HTTP hardening package.

Each run creates a temporary SQLite database and memory-only signing/encryption/
client keys, then closes servers and removes test state. The restart test closes
and reopens SQLite and rebuilds the OP while retaining those same keys in memory;
it does not demonstrate persistent key custody or a full process restart. The
separate storage crash test covers uncommitted transaction recovery only.

The browser smoke rendered the actual embedded bundle and checked keyboard focus;
the full protocol journey was driven by Go HTTP tests, not browser automation.
There is no password, MFA, real consent UI, persistent SSO cookie, account CRUD or
administrator in the fixture. The production `cmd/qoauth` does not import it. The
finished Go web application example, two-app SSO, second RP implementation and
local conformance suite remain later acceptance work.

## Start with an existing OIDC client

Prefer a server-side web application or backend-for-frontend (BFF): the backend
stores tokens, and the browser receives only a Secure/HttpOnly application-session
cookie. A pure SPA is a public client with code+PKCE, no secret, exact CORS origins
and short-lived tokens kept out of persistent browser storage. A browser cannot
keep a client secret. Desktop/native clients use the system browser and a separately
tested native redirect profile; native/device support is optional future scope.

The first example should use `coreos/go-oidc/v3` for discovery/ID Token verification
and `golang.org/x/oauth2` for the code exchange. Prefer an independent RP library
from the OP library to expose interoperability defects. Do not wrap crypto into a
new proprietary SDK. The admin API may use generated clients from its reviewed
OpenAPI contract; an admin API client is not a login SDK.
Sources: [go-oidc](https://github.com/coreos/go-oidc),
[OAuth2 client](https://pkg.go.dev/golang.org/x/oauth2),
[native apps](https://www.rfc-editor.org/rfc/rfc8252.html), verified 2026-10-25.

## Example registration contract

This is a design record, not a supported configuration-file or API schema:

```json
{
  "issuer": "https://auth.example.test",
  "client_id": "example-go-web",
  "client_type": "confidential",
  "redirect_uris": ["https://app.example.test/auth/callback"],
  "post_logout_redirect_uris": ["https://app.example.test/signed-out"],
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "require_pkce": true,
  "scopes": ["openid", "profile", "email", "offline_access"]
}
```

Client authentication is separately configured; no example secret is embedded.
Refresh tokens are issued only when offline access and consent/policy allow them;
adding a grant type alone is not consent. In M2, register through an authorized
administrator; there is no anonymous dynamic registration endpoint.

For future local testing, first inspect metadata with a command like the following,
after replacing the domain with an explicitly authorized synthetic test issuer:

```sh
curl --fail --silent --show-error \
  https://auth.example.test/.well-known/openid-configuration
```

Expect exact issuer agreement, authorization/token/UserInfo/JWKS locations and
implemented response/grant/authentication methods. Do not add `--insecure` to make
a TLS mismatch disappear; use the test harness's explicit trust configuration.

## Code-flow walkthrough

1. Server creates random `state`, `nonce` and PKCE verifier using the chosen
   library; stores them with a short-lived, one-use login transaction bound to the
   browser session, client and issuer. Send only the S256 challenge upstream.
2. Redirect to the discovered authorization endpoint with `response_type=code`,
   client ID, exact registered redirect URI, `scope=openid profile email`, state,
   nonce, challenge and `code_challenge_method=S256`. Add `offline_access` only
    when the application's use case and user consent require it; for initial offline
    consent use `prompt=consent`. The registration above is the refresh-enabled
    variant; ordinary login may omit offline_access and must not expect refresh.
3. Callback checks state and expected issuer/error context, then consumes the
   login transaction once. Backend exchanges code with verifier, redirect URI
   and the registered confidential-client authentication method.
4. Library verifies the ID Token signature and allowed algorithm; application
   enforces exact issuer, audience/authorized party, expiry, nonce and required
   authentication age/assurance. Associate the account using `(iss, sub)`.
5. Rotate the application session identifier. Store tokens server-side. API calls
   use an access token with the correct resource audience/scopes, never the ID Token.
   Serialize refresh; on reuse/invalid_grant, discard the family and reauthenticate.
6. On logout, clear the app session and, when requested, use the discovered OP
   end-session endpoint with validated ID Token hint/context and a pre-registered
   return URI. Validate returned logout state. This cannot clear an unrelated
   application's cookie. Back-channel logout is an optional future integration,
   not an initial server feature or universal logout guarantee.

The expected flow is library-driven; the executable fixture above demonstrates
its M1 subset. [OIDC Core](https://openid.net/specs/openid-connect-core-1_0.html),
[RFC 9700](https://www.rfc-editor.org/rfc/rfc9700.html),
[back-channel logout](https://openid.net/specs/openid-connect-backchannel-1_0.html).

## API/resource clients

For the initial opaque-token profile, an API authenticates itself to introspection,
checks `active`, intended audience, expiry and required scope, and applies its own
record-level permission rules. Introspection must not expose arbitrary other
clients' tokens. A successful introspection is not a blanket permission to every
API route. Fail closed if the required online validation is unavailable.
[RFC 7662](https://www.rfc-editor.org/rfc/rfc7662.html).

Machine-client `client_credentials` is optional future work for an actual use case;
it must not be advertised before implementation. Such a client has assigned
resource/scopes and is not a human identity. No machine-client framework is needed
for the first release.

## Integration acceptance pack

Ship a Go web RP and generic browser/BFF instructions with synthetic
fixtures, exact tested versions, minimal local setup, cleanup instructions and a
compatibility table. Verify login, two-app SSO, token expiry, wrong nonce/audience/
issuer, refresh replay, key rotation, user disable, consent withdrawal, local vs
QoAuth-wide logout, including a demonstration of surviving app-local cookies.
Include a clear revocation-delay statement for each app-session/token mode.
Real third-party SaaS tenants remain
unverified unless their test access is explicitly authorized.
