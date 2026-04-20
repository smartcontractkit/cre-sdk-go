package confidentialhttp

import (
	"github.com/smartcontractkit/cre-sdk-go/cre"
)

// This file is hand-written (not generated). It provides ergonomic builders
// for the AuthConfig variants and a Send() convenience wrapper over the
// generated SendRequest.
//
// The generated SendRequest remains available for advanced callers that
// need to construct the raw *ConfidentialHTTPRequest directly.

// Send is the recommended entry point for sending a confidential HTTP
// request. It assembles a *ConfidentialHTTPRequest from the supplied
// *HTTPRequest and a variadic list of RequestOptions.
//
// Typical usage:
//
//	client.Send(runtime,
//	    &confhttp.HTTPRequest{Url: "https://example.com", Method: "GET"},
//	    confhttp.WithSecrets(mySecret),
//	    confhttp.WithAuth(confhttp.WithApiKey("x-api-key", "my_api_key")),
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

// WithSecrets declares the Vault-DON secrets that the capability must fetch
// before executing the request. Every secret name referenced by an
// AuthConfig must also appear here.
func WithSecrets(ids ...*SecretIdentifier) RequestOption {
	return func(r *ConfidentialHTTPRequest) {
		r.VaultDonSecrets = append(r.VaultDonSecrets, ids...)
	}
}

// WithAuth attaches an AuthConfig to the request so the capability signs
// the outbound request using the selected method.
func WithAuth(a *AuthConfig) RequestOption {
	return func(r *ConfidentialHTTPRequest) {
		r.Auth = a
	}
}

// -----------------------------------------------------------------------------
// API Key
// -----------------------------------------------------------------------------

// WithApiKey constructs an AuthConfig that attaches a secret value to the
// named header. Example:
//
//	WithApiKey("x-api-key", "coingecko_api_key")
//	WithApiKey("Authorization", "pager_token", "ApiKey ")  // with prefix
func WithApiKey(headerName, secretName string, valuePrefix ...string) *AuthConfig {
	prefix := ""
	if len(valuePrefix) > 0 {
		prefix = valuePrefix[0]
	}
	return &AuthConfig{Method: &AuthConfig_ApiKey{ApiKey: &ApiKeyAuth{
		HeaderName:  headerName,
		SecretName:  secretName,
		ValuePrefix: prefix,
	}}}
}

// -----------------------------------------------------------------------------
// Basic
// -----------------------------------------------------------------------------

// WithBasicAuth constructs an AuthConfig that sends
// `Authorization: Basic base64(username:password)`.
func WithBasicAuth(usernameSecretName, passwordSecretName string) *AuthConfig {
	return &AuthConfig{Method: &AuthConfig_Basic{Basic: &BasicAuth{
		UsernameSecretName: usernameSecretName,
		PasswordSecretName: passwordSecretName,
	}}}
}

// -----------------------------------------------------------------------------
// Bearer
// -----------------------------------------------------------------------------

// BearerOption customizes a BearerToken AuthConfig.
type BearerOption func(*BearerAuth)

// BearerHeader overrides the header name (default "Authorization").
func BearerHeader(name string) BearerOption {
	return func(b *BearerAuth) { b.HeaderName = name }
}

// BearerPrefix overrides the value prefix (default "Bearer ").
// Useful for e.g. GitHub's "Authorization: token <pat>".
func BearerPrefix(prefix string) BearerOption {
	return func(b *BearerAuth) { b.ValuePrefix = prefix }
}

// WithBearerToken attaches a pre-issued bearer token as
// `Authorization: Bearer <token>` (defaults). Header name / prefix can be
// overridden via BearerHeader / BearerPrefix.
func WithBearerToken(tokenSecretName string, opts ...BearerOption) *AuthConfig {
	b := &BearerAuth{TokenSecretName: tokenSecretName}
	for _, o := range opts {
		o(b)
	}
	return &AuthConfig{Method: &AuthConfig_Bearer{Bearer: b}}
}

// -----------------------------------------------------------------------------
// HMAC-SHA256
// -----------------------------------------------------------------------------

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
//
// Signature is attached to signatureHeader (default "X-Signature") and
// the timestamp to timestampHeader (default "X-Timestamp"). Pass empty
// strings to use the defaults.
func WithHmacSha256(secretName, signatureHeader, timestampHeader string, opts ...HmacSha256Option) *AuthConfig {
	h := &HmacSha256{
		SecretName:      secretName,
		SignatureHeader: signatureHeader,
		TimestampHeader: timestampHeader,
	}
	for _, o := range opts {
		o(h)
	}
	return &AuthConfig{Method: &AuthConfig_Hmac{Hmac: &HmacAuth{
		Variant: &HmacAuth_Sha256{Sha256: h},
	}}}
}

// -----------------------------------------------------------------------------
// AWS SigV4
// -----------------------------------------------------------------------------

// SigV4Option customizes an AwsSigV4 AuthConfig.
type SigV4Option func(*AwsSigV4)

// WithSessionToken includes a temporary STS session token.
func WithSessionToken(secretName string) SigV4Option {
	return func(a *AwsSigV4) { a.SessionTokenSecretName = secretName }
}

// WithSignedHeaders overrides the default set of signed headers.
func WithSignedHeaders(headers ...string) SigV4Option {
	return func(a *AwsSigV4) { a.SignedHeaders = headers }
}

// WithUnsignedPayload enables S3-style UNSIGNED-PAYLOAD signing (useful for
// large body uploads).
func WithUnsignedPayload(v bool) SigV4Option {
	return func(a *AwsSigV4) { a.UnsignedPayload = v }
}

// WithAwsSigV4 signs outbound requests using AWS Signature Version 4.
// Example:
//
//	WithAwsSigV4("aws_ak", "aws_sk", "us-east-1", "execute-api")
//	WithAwsSigV4("aws_ak", "aws_sk", "us-east-1", "s3",
//	    WithSessionToken("aws_st"), WithUnsignedPayload(true))
func WithAwsSigV4(accessKeyIDSecretName, secretAccessKeySecretName, region, service string, opts ...SigV4Option) *AuthConfig {
	a := &AwsSigV4{
		AccessKeyIdSecretName:     accessKeyIDSecretName,
		SecretAccessKeySecretName: secretAccessKeySecretName,
		Region:                    region,
		Service:                   service,
	}
	for _, o := range opts {
		o(a)
	}
	return &AuthConfig{Method: &AuthConfig_Hmac{Hmac: &HmacAuth{
		Variant: &HmacAuth_AwsSigV4{AwsSigV4: a},
	}}}
}

// -----------------------------------------------------------------------------
// HMAC Custom
// -----------------------------------------------------------------------------

// Hash identifies the hash algorithm used by HmacCustom.
type Hash = HmacCustom_Hash

const (
	HashSHA256 Hash = HmacCustom_HASH_SHA256
	HashSHA512 Hash = HmacCustom_HASH_SHA512
)

// HmacCustomOpts carries the parameters for a fully user-defined HMAC
// signing scheme.
type HmacCustomOpts struct {
	SecretName        string
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
func WithHmacCustom(opts HmacCustomOpts) *AuthConfig {
	return &AuthConfig{Method: &AuthConfig_Hmac{Hmac: &HmacAuth{
		Variant: &HmacAuth_Custom{Custom: &HmacCustom{
			SecretName:        opts.SecretName,
			CanonicalTemplate: opts.CanonicalTemplate,
			Hash:              opts.Hash,
			Encoding:          opts.Encoding,
			SignatureHeader:   opts.SignatureHeader,
			SignaturePrefix:   opts.SignaturePrefix,
			TimestampHeader:   opts.TimestampHeader,
			NonceHeader:       opts.NonceHeader,
		}},
	}}}
}

// -----------------------------------------------------------------------------
// OAuth 2.0 — Client Credentials
// -----------------------------------------------------------------------------

// OAuth2Option customizes an OAuth2 AuthConfig. The same type is accepted by
// both WithOAuth2ClientCredentials and WithOAuth2RefreshToken; not every
// option applies to both grants (e.g. WithAudience is only meaningful for
// client_credentials).
type OAuth2Option func(*oauth2Opts)

type oauth2Opts struct {
	scopes           []string
	audience         string
	clientAuthMethod string
	clientIDSecret   string
	clientSecret     string
	extraParams      map[string]string
}

// WithScopes sets the OAuth2 scope list.
func WithScopes(scopes ...string) OAuth2Option {
	return func(o *oauth2Opts) { o.scopes = scopes }
}

// WithAudience sets the Auth0-style "audience" parameter (client_credentials).
func WithAudience(a string) OAuth2Option {
	return func(o *oauth2Opts) { o.audience = a }
}

// WithOAuth2ClientBasic sends client_id/client_secret via HTTP Basic Auth
// on the token endpoint (default behavior).
func WithOAuth2ClientBasic() OAuth2Option {
	return func(o *oauth2Opts) { o.clientAuthMethod = "basic_auth" }
}

// WithOAuth2ClientBody sends client_id/client_secret in the request body.
func WithOAuth2ClientBody() OAuth2Option {
	return func(o *oauth2Opts) { o.clientAuthMethod = "request_body" }
}

// WithClientID attaches a client_id secret name (refresh_token grant).
func WithClientID(secretName string) OAuth2Option {
	return func(o *oauth2Opts) { o.clientIDSecret = secretName }
}

// WithClientSecret attaches a client_secret secret name (refresh_token grant).
func WithClientSecret(secretName string) OAuth2Option {
	return func(o *oauth2Opts) { o.clientSecret = secretName }
}

// WithExtraParams merges extra form params sent to the token endpoint.
func WithExtraParams(params map[string]string) OAuth2Option {
	return func(o *oauth2Opts) { o.extraParams = params }
}

// WithOAuth2ClientCredentials constructs an OAuth2 client_credentials
// AuthConfig. The capability will exchange client_id + client_secret for an
// access token at tokenURL, cache it per-workflow-owner, and attach
// `Authorization: Bearer <access_token>` to the outbound request.
//
// tokenURL must be https://.
func WithOAuth2ClientCredentials(tokenURL, clientIDSecretName, clientSecretSecretName string, opts ...OAuth2Option) *AuthConfig {
	o := &oauth2Opts{}
	for _, opt := range opts {
		opt(o)
	}
	cc := &OAuth2ClientCredentials{
		TokenUrl:               tokenURL,
		ClientIdSecretName:     clientIDSecretName,
		ClientSecretSecretName: clientSecretSecretName,
		Scopes:                 o.scopes,
		Audience:               o.audience,
		ClientAuthMethod:       o.clientAuthMethod,
		ExtraParams:            o.extraParams,
	}
	return &AuthConfig{Method: &AuthConfig_Oauth2{Oauth2: &OAuth2Auth{
		Variant: &OAuth2Auth_ClientCredentials{ClientCredentials: cc},
	}}}
}

// -----------------------------------------------------------------------------
// OAuth 2.0 — Refresh Token
// -----------------------------------------------------------------------------

// WithOAuth2RefreshToken constructs an OAuth2 refresh_token AuthConfig. The
// workflow must have a long-lived refresh_token stored in Vault. The
// capability exchanges it at tokenURL for an access_token on cache miss.
//
// tokenURL must be https://.
//
// Note: if the IdP rotates refresh tokens on every exchange, the capability
// cannot write the new token back to Vault. Prefer IdPs where refresh
// rotation is disabled, or use client_credentials when possible.
func WithOAuth2RefreshToken(tokenURL, refreshTokenSecretName string, opts ...OAuth2Option) *AuthConfig {
	o := &oauth2Opts{}
	for _, opt := range opts {
		opt(o)
	}
	rt := &OAuth2RefreshToken{
		TokenUrl:               tokenURL,
		RefreshTokenSecretName: refreshTokenSecretName,
		ClientIdSecretName:     o.clientIDSecret,
		ClientSecretSecretName: o.clientSecret,
		Scopes:                 o.scopes,
		ExtraParams:            o.extraParams,
	}
	return &AuthConfig{Method: &AuthConfig_Oauth2{Oauth2: &OAuth2Auth{
		Variant: &OAuth2Auth_RefreshToken{RefreshToken: rt},
	}}}
}
