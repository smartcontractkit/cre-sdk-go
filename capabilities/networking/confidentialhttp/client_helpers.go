package confidentialhttp

import (
	"github.com/smartcontractkit/cre-sdk-go/cre"
)

// appendSecret appends id to the collector and returns its Key for use as
// the proto template name.
func appendSecret(id *SecretIdentifier, secrets *[]*SecretIdentifier) string {
	*secrets = append(*secrets, id)
	return id.Key
}

// ---------------------------------------------------------------------------
// Send + RequestOption
// ---------------------------------------------------------------------------

// Send is the recommended entry point for sending a confidential HTTP
// request. It assembles a *ConfidentialHTTPRequest from the supplied
// *HTTPRequest and a variadic list of RequestOptions.
//
// Typical usage:
//
//	apiKey := &confhttp.SecretIdentifier{Key: "cg_key", Namespace: "my-ns"}
//	client.Send(runtime,
//	    &confhttp.HTTPRequest{Url: "https://example.com", Method: "GET"},
//	    confhttp.WithApiKey("x-api-key", apiKey),
//	)
func (c *Client) Send(runtime cre.Runtime, req *HTTPRequest, opts ...RequestOption) cre.Promise[*HTTPResponse] {
	cr := &ConfidentialHTTPRequest{Request: req}
	for _, o := range opts {
		o(cr)
	}
	return c.SendRequest(runtime, cr)
}

// RequestOption is a functional option applied to a ConfidentialHTTPRequest
// before it is marshaled and sent.
type RequestOption func(*ConfidentialHTTPRequest)

// WithSecrets declares additional Vault-DON secrets that the capability must
// fetch. Secrets passed to auth helpers are registered automatically; use
// this only for extra secrets not covered by the auth config.
func WithSecrets(ids ...*SecretIdentifier) RequestOption {
	return func(r *ConfidentialHTTPRequest) {
		r.VaultDonSecrets = append(r.VaultDonSecrets, ids...)
	}
}

// WithAuth attaches a pre-built AuthConfig to the request. Prefer the typed
// helpers (WithApiKey, WithBasicAuth, …) which set auth and register secrets
// in one step. This is available for callers who construct *AuthConfig
// manually.
func WithAuth(a *AuthConfig) RequestOption {
	return func(r *ConfidentialHTTPRequest) {
		r.Auth = a
	}
}

// ---------------------------------------------------------------------------
// API Key
// ---------------------------------------------------------------------------

// WithApiKey attaches a secret value to the named header.
//
//	WithApiKey("x-api-key", secret)
//	WithApiKey("Authorization", secret, "ApiKey ")  // with prefix
func WithApiKey(headerName string, secret *SecretIdentifier, valuePrefix ...string) RequestOption {
	return func(r *ConfidentialHTTPRequest) {
		var secrets []*SecretIdentifier
		name := appendSecret(secret, &secrets)
		prefix := ""
		if len(valuePrefix) > 0 {
			prefix = valuePrefix[0]
		}
		r.Auth = &AuthConfig{Method: &AuthConfig_ApiKey{ApiKey: &ApiKeyAuth{
			HeaderName:  headerName,
			SecretName:  name,
			ValuePrefix: prefix,
		}}}
		r.VaultDonSecrets = append(r.VaultDonSecrets, secrets...)
	}
}

// ---------------------------------------------------------------------------
// Basic
// ---------------------------------------------------------------------------

// WithBasicAuth sends `Authorization: Basic base64(username:password)`.
func WithBasicAuth(username, password *SecretIdentifier) RequestOption {
	return func(r *ConfidentialHTTPRequest) {
		var secrets []*SecretIdentifier
		uName := appendSecret(username, &secrets)
		pName := appendSecret(password, &secrets)
		r.Auth = &AuthConfig{Method: &AuthConfig_Basic{Basic: &BasicAuth{
			UsernameSecretName: uName,
			PasswordSecretName: pName,
		}}}
		r.VaultDonSecrets = append(r.VaultDonSecrets, secrets...)
	}
}

// ---------------------------------------------------------------------------
// Bearer
// ---------------------------------------------------------------------------

// BearerOption customizes a BearerToken auth config.
type BearerOption func(*BearerAuth)

// BearerHeader overrides the header name (default "Authorization").
func BearerHeader(name string) BearerOption {
	return func(b *BearerAuth) { b.HeaderName = name }
}

// BearerPrefix overrides the value prefix (default "Bearer ").
func BearerPrefix(prefix string) BearerOption {
	return func(b *BearerAuth) { b.ValuePrefix = prefix }
}

// WithBearerToken attaches a pre-issued bearer token as
// `Authorization: Bearer <token>` (defaults). Header name / prefix can be
// overridden via BearerHeader / BearerPrefix.
func WithBearerToken(token *SecretIdentifier, opts ...BearerOption) RequestOption {
	return func(r *ConfidentialHTTPRequest) {
		var secrets []*SecretIdentifier
		name := appendSecret(token, &secrets)
		b := &BearerAuth{TokenSecretName: name}
		for _, o := range opts {
			o(b)
		}
		r.Auth = &AuthConfig{Method: &AuthConfig_Bearer{Bearer: b}}
		r.VaultDonSecrets = append(r.VaultDonSecrets, secrets...)
	}
}

// ---------------------------------------------------------------------------
// HMAC-SHA256
// ---------------------------------------------------------------------------

// HmacSha256Option customizes a HmacSha256 AuthConfig.
type HmacSha256Option func(*HmacSha256)

// HmacIncludeQuery tells the signer to include the query string in the
// canonical URL.
func HmacIncludeQuery(v bool) HmacSha256Option {
	return func(h *HmacSha256) { h.IncludeQuery = v }
}

// HmacEncoding sets the signature encoding ("hex" default, or "base64").
func HmacEncoding(enc string) HmacSha256Option {
	return func(h *HmacSha256) { h.Encoding = enc }
}

// WithHmacSha256 signs requests using HMAC-SHA256 over the canonical string
//
//	method "\n" url "\n" sha256(body) "\n" timestamp.
func WithHmacSha256(secret *SecretIdentifier, signatureHeader, timestampHeader string, opts ...HmacSha256Option) RequestOption {
	return func(r *ConfidentialHTTPRequest) {
		var secrets []*SecretIdentifier
		name := appendSecret(secret, &secrets)
		h := &HmacSha256{
			SecretName:      name,
			SignatureHeader: signatureHeader,
			TimestampHeader: timestampHeader,
		}
		for _, o := range opts {
			o(h)
		}
		r.Auth = &AuthConfig{Method: &AuthConfig_Hmac{Hmac: &HmacAuth{
			Variant: &HmacAuth_Sha256{Sha256: h},
		}}}
		r.VaultDonSecrets = append(r.VaultDonSecrets, secrets...)
	}
}

// ---------------------------------------------------------------------------
// AWS SigV4
// ---------------------------------------------------------------------------

type sigV4Config struct {
	sessionToken    *SecretIdentifier
	signedHeaders   []string
	unsignedPayload bool
}

// SigV4Option customizes an AwsSigV4 AuthConfig.
type SigV4Option func(*sigV4Config)

// WithSessionToken includes a temporary STS session token.
func WithSessionToken(secret *SecretIdentifier) SigV4Option {
	return func(c *sigV4Config) { c.sessionToken = secret }
}

// WithSignedHeaders overrides the default set of signed headers.
func WithSignedHeaders(headers ...string) SigV4Option {
	return func(c *sigV4Config) { c.signedHeaders = headers }
}

// WithUnsignedPayload enables S3-style UNSIGNED-PAYLOAD signing.
func WithUnsignedPayload(v bool) SigV4Option {
	return func(c *sigV4Config) { c.unsignedPayload = v }
}

// WithAwsSigV4 signs outbound requests using AWS Signature Version 4.
//
//	WithAwsSigV4(akSecret, skSecret, "us-east-1", "execute-api")
//	WithAwsSigV4(akSecret, skSecret, "us-east-1", "s3",
//	    WithSessionToken(stsSecret), WithUnsignedPayload(true))
func WithAwsSigV4(
	accessKeyID, secretAccessKey *SecretIdentifier, region, service string, opts ...SigV4Option,
) RequestOption {
	return func(r *ConfidentialHTTPRequest) {
		var secrets []*SecretIdentifier
		akName := appendSecret(accessKeyID, &secrets)
		skName := appendSecret(secretAccessKey, &secrets)

		cfg := &sigV4Config{}
		for _, o := range opts {
			o(cfg)
		}

		a := &AwsSigV4{
			AccessKeyIdSecretName:     akName,
			SecretAccessKeySecretName: skName,
			Region:                    region,
			Service:                   service,
			SignedHeaders:             cfg.signedHeaders,
			UnsignedPayload:           cfg.unsignedPayload,
		}
		if cfg.sessionToken != nil {
			a.SessionTokenSecretName = appendSecret(cfg.sessionToken, &secrets)
		}

		r.Auth = &AuthConfig{Method: &AuthConfig_Hmac{Hmac: &HmacAuth{
			Variant: &HmacAuth_AwsSigV4{AwsSigV4: a},
		}}}
		r.VaultDonSecrets = append(r.VaultDonSecrets, secrets...)
	}
}

// ---------------------------------------------------------------------------
// HMAC Custom
// ---------------------------------------------------------------------------

// Hash identifies the hash algorithm used by HmacCustom.
type Hash = HmacCustom_Hash

const (
	HashSHA256 Hash = HmacCustom_HASH_SHA256
	HashSHA512 Hash = HmacCustom_HASH_SHA512
)

// HmacCustomConfig carries the non-secret parameters for a fully
// user-defined HMAC signing scheme. The secret itself is passed as the first
// argument to WithHmacCustom.
type HmacCustomConfig struct {
	CanonicalTemplate string // Go text/template
	Hash              Hash
	Encoding          string // "hex" (default) or "base64"
	SignatureHeader   string
	SignaturePrefix   string
	TimestampHeader   string // if set, a timestamp header is injected
	NonceHeader       string // if set, a random-nonce header is injected
}

// WithHmacCustom constructs an HMAC AuthConfig that uses a user-defined
// canonical-string template. Template vars available at signing time:
//
//	{{.method}} {{.url}} {{.path}} {{.query}} {{.body}} {{.body_sha256}}
//	{{.timestamp}} {{.nonce}} {{header "X-Foo"}}
func WithHmacCustom(secret *SecretIdentifier, cfg HmacCustomConfig) RequestOption {
	return func(r *ConfidentialHTTPRequest) {
		var secrets []*SecretIdentifier
		name := appendSecret(secret, &secrets)
		r.Auth = &AuthConfig{Method: &AuthConfig_Hmac{Hmac: &HmacAuth{
			Variant: &HmacAuth_Custom{Custom: &HmacCustom{
				SecretName:        name,
				CanonicalTemplate: cfg.CanonicalTemplate,
				Hash:              cfg.Hash,
				Encoding:          cfg.Encoding,
				SignatureHeader:   cfg.SignatureHeader,
				SignaturePrefix:   cfg.SignaturePrefix,
				TimestampHeader:   cfg.TimestampHeader,
				NonceHeader:       cfg.NonceHeader,
			}},
		}}}
		r.VaultDonSecrets = append(r.VaultDonSecrets, secrets...)
	}
}

// ---------------------------------------------------------------------------
// OAuth 2.0 — shared options
// ---------------------------------------------------------------------------

type oauth2Config struct {
	scopes           []string
	audience         string
	clientAuthMethod string
	clientIDSecret   *SecretIdentifier
	clientSecret     *SecretIdentifier
	extraParams      map[string]string
}

// OAuth2Option customizes an OAuth2 AuthConfig. The same type is accepted by
// both WithOAuth2ClientCredentials and WithOAuth2RefreshToken; not every
// option applies to both grants (e.g. WithAudience is only meaningful for
// client_credentials).
type OAuth2Option func(*oauth2Config)

// WithScopes sets the OAuth2 scope list.
func WithScopes(scopes ...string) OAuth2Option {
	return func(o *oauth2Config) { o.scopes = scopes }
}

// WithAudience sets the Auth0-style "audience" parameter (client_credentials).
func WithAudience(a string) OAuth2Option {
	return func(o *oauth2Config) { o.audience = a }
}

// WithOAuth2ClientBasic sends client_id/client_secret via HTTP Basic Auth
// on the token endpoint (default behavior).
func WithOAuth2ClientBasic() OAuth2Option {
	return func(o *oauth2Config) { o.clientAuthMethod = "basic_auth" }
}

// WithOAuth2ClientBody sends client_id/client_secret in the request body.
func WithOAuth2ClientBody() OAuth2Option {
	return func(o *oauth2Config) { o.clientAuthMethod = "request_body" }
}

// WithClientID attaches a client_id secret (refresh_token grant).
func WithClientID(secret *SecretIdentifier) OAuth2Option {
	return func(o *oauth2Config) { o.clientIDSecret = secret }
}

// WithClientSecret attaches a client_secret secret (refresh_token grant).
func WithClientSecret(secret *SecretIdentifier) OAuth2Option {
	return func(o *oauth2Config) { o.clientSecret = secret }
}

// WithExtraParams merges extra form params sent to the token endpoint.
func WithExtraParams(params map[string]string) OAuth2Option {
	return func(o *oauth2Config) { o.extraParams = params }
}

// ---------------------------------------------------------------------------
// OAuth 2.0 — Client Credentials
// ---------------------------------------------------------------------------

// WithOAuth2ClientCredentials constructs an OAuth2 client_credentials
// RequestOption. The capability exchanges client_id + client_secret for an
// access token at tokenURL, caches it per-workflow-owner, and attaches
// `Authorization: Bearer <access_token>` to the outbound request.
//
// tokenURL must be https://.
func WithOAuth2ClientCredentials(
	tokenURL string, clientID, clientSecret *SecretIdentifier, opts ...OAuth2Option,
) RequestOption {
	return func(r *ConfidentialHTTPRequest) {
		var secrets []*SecretIdentifier
		cidName := appendSecret(clientID, &secrets)
		csName := appendSecret(clientSecret, &secrets)

		o := &oauth2Config{}
		for _, opt := range opts {
			opt(o)
		}

		cc := &OAuth2ClientCredentials{
			TokenUrl:               tokenURL,
			ClientIdSecretName:     cidName,
			ClientSecretSecretName: csName,
			Scopes:                 o.scopes,
			Audience:               o.audience,
			ClientAuthMethod:       o.clientAuthMethod,
			ExtraParams:            o.extraParams,
		}
		r.Auth = &AuthConfig{Method: &AuthConfig_Oauth2{Oauth2: &OAuth2Auth{
			Variant: &OAuth2Auth_ClientCredentials{ClientCredentials: cc},
		}}}
		r.VaultDonSecrets = append(r.VaultDonSecrets, secrets...)
	}
}

// ---------------------------------------------------------------------------
// OAuth 2.0 — Refresh Token
// ---------------------------------------------------------------------------

// WithOAuth2RefreshToken constructs an OAuth2 refresh_token RequestOption.
// The workflow must have a long-lived refresh_token stored in Vault. The
// capability exchanges it at tokenURL for an access_token on cache miss.
//
// tokenURL must be https://.
func WithOAuth2RefreshToken(
	tokenURL string, refreshToken *SecretIdentifier, opts ...OAuth2Option,
) RequestOption {
	return func(r *ConfidentialHTTPRequest) {
		var secrets []*SecretIdentifier
		rtName := appendSecret(refreshToken, &secrets)

		o := &oauth2Config{}
		for _, opt := range opts {
			opt(o)
		}

		rt := &OAuth2RefreshToken{
			TokenUrl:               tokenURL,
			RefreshTokenSecretName: rtName,
			Scopes:                 o.scopes,
			ExtraParams:            o.extraParams,
		}
		if o.clientIDSecret != nil {
			rt.ClientIdSecretName = appendSecret(o.clientIDSecret, &secrets)
		}
		if o.clientSecret != nil {
			rt.ClientSecretSecretName = appendSecret(o.clientSecret, &secrets)
		}

		r.Auth = &AuthConfig{Method: &AuthConfig_Oauth2{Oauth2: &OAuth2Auth{
			Variant: &OAuth2Auth_RefreshToken{RefreshToken: rt},
		}}}
		r.VaultDonSecrets = append(r.VaultDonSecrets, secrets...)
	}
}
