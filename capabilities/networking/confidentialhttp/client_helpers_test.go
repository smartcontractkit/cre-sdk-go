package confidentialhttp

import (
	"testing"
)

func applyOpts(opts ...RequestOption) *ConfidentialHTTPRequest {
	r := &ConfidentialHTTPRequest{Request: &HTTPRequest{Url: "https://x", Method: "GET"}}
	for _, o := range opts {
		o(r)
	}
	return r
}

func requireSecrets(t *testing.T, r *ConfidentialHTTPRequest, wantKeys ...string) {
	t.Helper()
	if len(r.VaultDonSecrets) != len(wantKeys) {
		t.Fatalf("VaultDonSecrets: got %d, want %d", len(r.VaultDonSecrets), len(wantKeys))
	}
	for i, k := range wantKeys {
		if r.VaultDonSecrets[i].Key != k {
			t.Fatalf("VaultDonSecrets[%d].Key = %q, want %q", i, r.VaultDonSecrets[i].Key, k)
		}
	}
}

func sid(key string) *SecretIdentifier {
	return &SecretIdentifier{Key: key, Namespace: "ns"}
}

// ---------------------------------------------------------------------------
// API Key
// ---------------------------------------------------------------------------

func TestWithApiKey(t *testing.T) {
	r := applyOpts(WithApiKey("x-api-key", sid("cg")))
	ak := r.Auth.GetApiKey()
	if ak == nil {
		t.Fatalf("expected ApiKey variant, got %T", r.Auth.GetMethod())
	}
	if ak.GetHeaderName() != "x-api-key" {
		t.Fatalf("header=%q", ak.GetHeaderName())
	}
	if ak.GetSecret().GetKey() != "cg" {
		t.Fatalf("secret=%q", ak.GetSecret().GetKey())
	}
	if ak.GetValuePrefix() != "" {
		t.Fatalf("prefix=%q", ak.GetValuePrefix())
	}
	requireSecrets(t, r, "cg")
}

func TestWithApiKey_Prefix(t *testing.T) {
	r := applyOpts(WithApiKey("Authorization", sid("tok"), "ApiKey "))
	if r.Auth.GetApiKey().GetValuePrefix() != "ApiKey " {
		t.Fatalf("prefix=%q", r.Auth.GetApiKey().GetValuePrefix())
	}
	requireSecrets(t, r, "tok")
}

// ---------------------------------------------------------------------------
// Basic Auth
// ---------------------------------------------------------------------------

func TestWithBasicAuth_BothSecrets(t *testing.T) {
	r := applyOpts(WithBasicAuth(sid("u"), sid("p")))
	b := r.Auth.GetBasic()
	if b == nil {
		t.Fatalf("expected Basic")
	}
	if b.GetUsername().GetSecret().GetKey() != "u" {
		t.Fatalf("username=%v", b.GetUsername())
	}
	if b.GetPassword().GetKey() != "p" {
		t.Fatalf("password=%v", b.GetPassword())
	}
	requireSecrets(t, r, "u", "p")
}

func TestWithBasicAuth_StringUsername(t *testing.T) {
	r := applyOpts(WithBasicAuth("admin", sid("p")))
	b := r.Auth.GetBasic()
	if b == nil {
		t.Fatalf("expected Basic")
	}
	if b.GetUsername().GetPlain() != "admin" {
		t.Fatalf("username=%v", b.GetUsername())
	}
	if b.GetPassword().GetKey() != "p" {
		t.Fatalf("password=%v", b.GetPassword())
	}
	requireSecrets(t, r, "p")
}

// ---------------------------------------------------------------------------
// Bearer
// ---------------------------------------------------------------------------

func TestWithBearerToken(t *testing.T) {
	r := applyOpts(WithBearerToken(sid("pat")))
	b := r.Auth.GetBearer()
	if b == nil {
		t.Fatalf("no bearer")
	}
	if b.GetToken().GetKey() != "pat" {
		t.Fatalf("token=%v", b.GetToken())
	}
	if b.GetHeaderName() != "" || b.GetValuePrefix() != "" {
		t.Fatalf("defaults should be empty, got header=%q prefix=%q", b.GetHeaderName(), b.GetValuePrefix())
	}
	requireSecrets(t, r, "pat")
}

func TestWithBearerToken_WithOverrides(t *testing.T) {
	r := applyOpts(WithBearerToken(sid("gh_pat"), BearerHeader("Authorization"), BearerPrefix("token ")))
	b := r.Auth.GetBearer()
	if b.GetToken().GetKey() != "gh_pat" {
		t.Fatalf("token=%v", b.GetToken())
	}
	if b.GetHeaderName() != "Authorization" || b.GetValuePrefix() != "token " {
		t.Fatalf("overrides not applied: %+v", b)
	}
	requireSecrets(t, r, "gh_pat")
}

// ---------------------------------------------------------------------------
// HMAC-SHA256
// ---------------------------------------------------------------------------

func TestWithHmacSha256(t *testing.T) {
	r := applyOpts(WithHmacSha256(sid("s"), "X-Sig", "X-TS", HmacIncludeQuery(true), HmacEncoding("base64")))
	h := r.Auth.GetHmac().GetSha256()
	if h == nil {
		t.Fatalf("no sha256 variant")
	}
	if h.GetSecret().GetKey() != "s" {
		t.Fatalf("secret=%v", h.GetSecret())
	}
	if !h.GetIncludeQuery() {
		t.Fatalf("include_query not set")
	}
	if h.GetEncoding() != "base64" {
		t.Fatalf("encoding=%q", h.GetEncoding())
	}
	requireSecrets(t, r, "s")
}

// ---------------------------------------------------------------------------
// AWS SigV4
// ---------------------------------------------------------------------------

func TestWithAwsSigV4_BothSecrets(t *testing.T) {
	r := applyOpts(WithAwsSigV4(sid("ak"), sid("sk"), "us-east-1", "s3",
		WithSessionToken(sid("st")),
		WithSignedHeaders("host", "x-amz-date"),
		WithUnsignedPayload(true),
	))
	a := r.Auth.GetHmac().GetAwsSigV4()
	if a == nil {
		t.Fatalf("no aws variant")
	}
	if a.GetAccessKeyId().GetSecret().GetKey() != "ak" {
		t.Fatalf("ak=%v", a.GetAccessKeyId())
	}
	if a.GetSecretAccessKey().GetKey() != "sk" {
		t.Fatalf("sk=%v", a.GetSecretAccessKey())
	}
	if a.GetSessionToken().GetKey() != "st" {
		t.Fatalf("session token not set")
	}
	if len(a.GetSignedHeaders()) != 2 {
		t.Fatalf("signed headers=%v", a.GetSignedHeaders())
	}
	if !a.GetUnsignedPayload() {
		t.Fatalf("unsigned payload not set")
	}
	requireSecrets(t, r, "ak", "sk", "st")
}

func TestWithAwsSigV4_StringAccessKeyID(t *testing.T) {
	r := applyOpts(WithAwsSigV4("AKIAIOSFODNN7EXAMPLE", sid("sk"), "us-east-1", "s3"))
	a := r.Auth.GetHmac().GetAwsSigV4()
	if a == nil {
		t.Fatalf("no aws variant")
	}
	if a.GetAccessKeyId().GetPlain() != "AKIAIOSFODNN7EXAMPLE" {
		t.Fatalf("ak=%v", a.GetAccessKeyId())
	}
	if a.GetSecretAccessKey().GetKey() != "sk" {
		t.Fatalf("sk=%v", a.GetSecretAccessKey())
	}
	if a.GetSessionToken() != nil {
		t.Fatalf("expected nil session token, got %v", a.GetSessionToken())
	}
	requireSecrets(t, r, "sk")
}

func TestWithAwsSigV4_NoSessionToken(t *testing.T) {
	r := applyOpts(WithAwsSigV4(sid("ak"), sid("sk"), "us-east-1", "s3"))
	a := r.Auth.GetHmac().GetAwsSigV4()
	if a.GetSessionToken() != nil {
		t.Fatalf("expected nil session token, got %v", a.GetSessionToken())
	}
	requireSecrets(t, r, "ak", "sk")
}

// ---------------------------------------------------------------------------
// HMAC Custom
// ---------------------------------------------------------------------------

func TestWithHmacCustom(t *testing.T) {
	r := applyOpts(WithHmacCustom(sid("k"), HmacCustomConfig{
		CanonicalTemplate: `{{.method}}`,
		Hash:              HashSHA512,
		Encoding:          "base64",
		SignatureHeader:   "X-Sig",
		SignaturePrefix:   "HMAC-SHA512 ",
		TimestampHeader:   "X-TS",
		NonceHeader:       "X-Nonce",
	}))
	c := r.Auth.GetHmac().GetCustom()
	if c == nil {
		t.Fatalf("no custom variant")
	}
	if c.GetHash() != HmacCustom_HASH_SHA512 {
		t.Fatalf("hash=%v", c.GetHash())
	}
	if c.GetSecret().GetKey() != "k" {
		t.Fatalf("secret=%v", c.GetSecret())
	}
	requireSecrets(t, r, "k")
}

// ---------------------------------------------------------------------------
// OAuth2 Client Credentials
// ---------------------------------------------------------------------------

func TestWithOAuth2ClientCredentials_BothSecrets(t *testing.T) {
	r := applyOpts(WithOAuth2ClientCredentials(
		"https://idp/token",
		sid("cid"), sid("csec"),
		WithScopes("read", "write"),
		WithAudience("aud"),
		WithOAuth2ClientBody(),
		WithExtraParams(map[string]string{"foo": "bar"}),
	))
	cc := r.Auth.GetOauth2().GetClientCredentials()
	if cc == nil {
		t.Fatalf("no client_credentials variant")
	}
	if cc.GetTokenUrl() != "https://idp/token" {
		t.Fatalf("token_url=%q", cc.GetTokenUrl())
	}
	if cc.GetClientId().GetSecret().GetKey() != "cid" {
		t.Fatalf("client_id=%v", cc.GetClientId())
	}
	if cc.GetClientSecret().GetKey() != "csec" {
		t.Fatalf("client_secret=%v", cc.GetClientSecret())
	}
	if len(cc.GetScopes()) != 2 {
		t.Fatalf("scopes=%v", cc.GetScopes())
	}
	if cc.GetClientAuthMethod() != "request_body" {
		t.Fatalf("auth_method=%q", cc.GetClientAuthMethod())
	}
	if cc.GetExtraParams()["foo"] != "bar" {
		t.Fatalf("extra_params missing")
	}
	requireSecrets(t, r, "cid", "csec")
}

func TestWithOAuth2ClientCredentials_StringClientID(t *testing.T) {
	r := applyOpts(WithOAuth2ClientCredentials(
		"https://idp/token",
		"my-client-id", sid("csec"),
	))
	cc := r.Auth.GetOauth2().GetClientCredentials()
	if cc == nil {
		t.Fatalf("no client_credentials variant")
	}
	if cc.GetClientId().GetPlain() != "my-client-id" {
		t.Fatalf("client_id=%v", cc.GetClientId())
	}
	if cc.GetClientSecret().GetKey() != "csec" {
		t.Fatalf("client_secret=%v", cc.GetClientSecret())
	}
	requireSecrets(t, r, "csec")
}

// ---------------------------------------------------------------------------
// OAuth2 Refresh Token
// ---------------------------------------------------------------------------

func TestWithOAuth2RefreshToken_BothSecrets(t *testing.T) {
	r := applyOpts(WithOAuth2RefreshToken(
		"https://idp/token", sid("rt"),
		WithClientID(sid("cid")),
		WithClientSecret(sid("csec")),
		WithScopes("read"),
	))
	rt := r.Auth.GetOauth2().GetRefreshToken()
	if rt == nil {
		t.Fatalf("no refresh_token variant")
	}
	if rt.GetRefreshToken().GetKey() != "rt" {
		t.Fatalf("refresh secret=%v", rt.GetRefreshToken())
	}
	if rt.GetClientId().GetSecret().GetKey() != "cid" {
		t.Fatalf("client_id=%v", rt.GetClientId())
	}
	if rt.GetClientSecret().GetKey() != "csec" {
		t.Fatalf("client_secret=%v", rt.GetClientSecret())
	}
	requireSecrets(t, r, "rt", "cid", "csec")
}

func TestWithOAuth2RefreshToken_StringClientID(t *testing.T) {
	r := applyOpts(WithOAuth2RefreshToken(
		"https://idp/token", sid("rt"),
		WithClientID("my-client-id"),
		WithClientSecret(sid("csec")),
	))
	rt := r.Auth.GetOauth2().GetRefreshToken()
	if rt == nil {
		t.Fatalf("no refresh_token variant")
	}
	if rt.GetClientId().GetPlain() != "my-client-id" {
		t.Fatalf("client_id=%v", rt.GetClientId())
	}
	if rt.GetClientSecret().GetKey() != "csec" {
		t.Fatalf("client_secret=%v", rt.GetClientSecret())
	}
	requireSecrets(t, r, "rt", "csec")
}

func TestWithOAuth2RefreshToken_NoClientCreds(t *testing.T) {
	r := applyOpts(WithOAuth2RefreshToken("https://idp/token", sid("rt")))
	rt := r.Auth.GetOauth2().GetRefreshToken()
	if rt.GetClientId() != nil || rt.GetClientSecret() != nil {
		t.Fatalf("expected nil client creds, got cid=%v csec=%v",
			rt.GetClientId(), rt.GetClientSecret())
	}
	requireSecrets(t, r, "rt")
}

// ---------------------------------------------------------------------------
// WithSecrets + WithAuth (backward compat / manual usage)
// ---------------------------------------------------------------------------

func TestWithSecrets_And_WithAuth(t *testing.T) {
	ids := []*SecretIdentifier{{Key: "a"}, {Key: "b"}}
	auth := &AuthConfig{Method: &AuthConfig_ApiKey{ApiKey: &ApiKeyAuth{
		HeaderName: "h", Secret: &SecretIdentifier{Key: "a"},
	}}}

	r := applyOpts(WithSecrets(ids...), WithAuth(auth))
	if len(r.VaultDonSecrets) != 2 {
		t.Fatalf("secrets=%d", len(r.VaultDonSecrets))
	}
	if r.GetAuth() == nil || r.GetAuth().GetApiKey() == nil {
		t.Fatalf("auth not attached")
	}
}

// ---------------------------------------------------------------------------
// Combined: auth helper + extra WithSecrets
// ---------------------------------------------------------------------------

func TestCombinedSecretsFromAuthAndManual(t *testing.T) {
	extra := &SecretIdentifier{Key: "extra", Namespace: "ns"}
	r := applyOpts(
		WithBasicAuth(sid("user"), sid("pass")),
		WithSecrets(extra),
	)
	if len(r.VaultDonSecrets) != 3 {
		t.Fatalf("expected 3 secrets, got %d", len(r.VaultDonSecrets))
	}
	if r.VaultDonSecrets[0].Key != "user" || r.VaultDonSecrets[1].Key != "pass" || r.VaultDonSecrets[2].Key != "extra" {
		t.Fatalf("unexpected secrets: %+v", r.VaultDonSecrets)
	}
}

func TestCombinedSecretsFromAuthAndManual_StringUsername(t *testing.T) {
	extra := &SecretIdentifier{Key: "extra", Namespace: "ns"}
	r := applyOpts(
		WithBasicAuth("admin", sid("pass")),
		WithSecrets(extra),
	)
	if len(r.VaultDonSecrets) != 2 {
		t.Fatalf("expected 2 secrets (pass + extra), got %d", len(r.VaultDonSecrets))
	}
	if r.VaultDonSecrets[0].Key != "pass" || r.VaultDonSecrets[1].Key != "extra" {
		t.Fatalf("unexpected secrets: %+v", r.VaultDonSecrets)
	}
}
