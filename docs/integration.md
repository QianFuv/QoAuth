# Integration contract and simple examples

Design examples only, 2026-10-25. QoAuth has no running endpoints or SDK yet.
All domains below are reserved examples; no clients, secrets, users, bindings or
network services have been created. Endpoint names other than standardized
discovery are resolved through metadata, not hard-coded guesses.

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

The expected flow is library-driven; this is intentionally not copy-paste server
code before the library/storage feasibility gate. [OIDC Core](https://openid.net/specs/openid-connect-core-1_0.html),
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
