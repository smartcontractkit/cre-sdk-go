package confidentialhttp

import (
	"testing"
)

// These tests verify the helper builders construct the expected proto
// oneof shape. They don't execute the request or invoke the capability.

func TestWithApiKey(t *testing.T) {
	ac := WithApiKey("x-api-key", "cg")
	if ac.GetApiKey() == nil {
		t.Fatalf("expected ApiKey variant, got %T", ac.GetMethod())
	}
	if ac.GetApiKey().GetHeaderName() != "x-api-key" {
		t.Fatalf("header=%q", ac.GetApiKey().GetHeaderName())
	}
	if ac.GetApiKey().GetSecretName() != "cg" {
		t.Fatalf("secret=%q", ac.GetApiKey().GetSecretName())
	}
	if ac.GetApiKey().GetValuePrefix() != "" {
		t.Fatalf("prefix=%q", ac.GetApiKey().GetValuePrefix())
	}
}

func TestWithApiKey_Prefix(t *testing.T) {
	ac := WithApiKey("Authorization", "tok", "ApiKey ")
	if ac.GetApiKey().GetValuePrefix() != "ApiKey " {
		t.Fatalf("prefix=%q", ac.GetApiKey().GetValuePrefix())
	}
}

func TestWithBasicAuth(t *testing.T) {
	ac := WithBasicAuth("u", "p")
	b := ac.GetBasic()
	if b == nil {
		t.Fatalf("expected Basic")
	}
	if b.GetUsernameSecretName() != "u" || b.GetPasswordSecretName() != "p" {
		t.Fatalf("names wrong: %+v", b)
	}
}

func TestWithBearerToken_DefaultsAndOverrides(t *testing.T) {
	ac := WithBearerToken("pat")
	b := ac.GetBearer()
	if b == nil {
		t.Fatalf("no bearer")
	}
	if b.GetTokenSecretName() != "pat" {
		t.Fatalf("token=%q", b.GetTokenSecretName())
	}
	// Defaults are resolved by the signer, not the helper — helper leaves
	// them empty.
	if b.GetHeaderName() != "" || b.GetValuePrefix() != "" {
		t.Fatalf("defaults should be empty in proto, got header=%q prefix=%q", b.GetHeaderName(), b.GetValuePrefix())
	}

	ac2 := WithBearerToken("pat", BearerHeader("Authorization"), BearerPrefix("token "))
	b2 := ac2.GetBearer()
	if b2.GetHeaderName() != "Authorization" || b2.GetValuePrefix() != "token " {
		t.Fatalf("overrides not applied: %+v", b2)
	}
}

func TestWithHmacSha256(t *testing.T) {
	ac := WithHmacSha256("s", "X-Sig", "X-TS", HmacIncludeQuery(true), HmacEncoding("base64"))
	h := ac.GetHmac().GetSha256()
	if h == nil {
		t.Fatalf("no sha256 variant")
	}
	if !h.GetIncludeQuery() {
		t.Fatalf("include_query not set")
	}
	if h.GetEncoding() != "base64" {
		t.Fatalf("encoding=%q", h.GetEncoding())
	}
}

func TestWithAwsSigV4_AllOptions(t *testing.T) {
	ac := WithAwsSigV4("ak", "sk", "us-east-1", "s3",
		WithSessionToken("st"),
		WithSignedHeaders("host", "x-amz-date"),
		WithUnsignedPayload(true),
	)
	a := ac.GetHmac().GetAwsSigV4()
	if a == nil {
		t.Fatalf("no aws variant")
	}
	if a.GetSessionTokenSecretName() != "st" {
		t.Fatalf("session token not set")
	}
	if len(a.GetSignedHeaders()) != 2 {
		t.Fatalf("signed headers=%v", a.GetSignedHeaders())
	}
	if !a.GetUnsignedPayload() {
		t.Fatalf("unsigned payload not set")
	}
}

func TestWithHmacCustom(t *testing.T) {
	ac := WithHmacCustom(HmacCustomOpts{
		SecretName:        "k",
		CanonicalTemplate: `{{.method}}`,
		Hash:              HashSHA512,
		Encoding:          "base64",
		SignatureHeader:   "X-Sig",
		SignaturePrefix:   "HMAC-SHA512 ",
		TimestampHeader:   "X-TS",
		NonceHeader:       "X-Nonce",
	})
	c := ac.GetHmac().GetCustom()
	if c == nil {
		t.Fatalf("no custom variant")
	}
	if c.GetHash() != HmacCustom_HASH_SHA512 {
		t.Fatalf("hash=%v", c.GetHash())
	}
}

func TestWithOAuth2ClientCredentials(t *testing.T) {
	ac := WithOAuth2ClientCredentials(
		"https://idp/token",
		"cid", "csec",
		WithScopes("read", "write"),
		WithAudience("aud"),
		WithOAuth2ClientBody(),
		WithExtraParams(map[string]string{"foo": "bar"}),
	)
	cc := ac.GetOauth2().GetClientCredentials()
	if cc == nil {
		t.Fatalf("no client_credentials variant")
	}
	if cc.GetTokenUrl() != "https://idp/token" {
		t.Fatalf("token_url=%q", cc.GetTokenUrl())
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
}

func TestWithOAuth2RefreshToken(t *testing.T) {
	ac := WithOAuth2RefreshToken(
		"https://idp/token", "rt",
		WithClientID("cid"),
		WithClientSecret("csec"),
		WithScopes("read"),
	)
	rt := ac.GetOauth2().GetRefreshToken()
	if rt == nil {
		t.Fatalf("no refresh_token variant")
	}
	if rt.GetRefreshTokenSecretName() != "rt" {
		t.Fatalf("refresh secret=%q", rt.GetRefreshTokenSecretName())
	}
	if rt.GetClientIdSecretName() != "cid" || rt.GetClientSecretSecretName() != "csec" {
		t.Fatalf("client creds not set: %+v", rt)
	}
}

func TestRequestOptions(t *testing.T) {
	ids := []*SecretIdentifier{{Key: "a"}, {Key: "b"}}
	auth := WithApiKey("h", "a")

	cr := &ConfidentialHTTPRequest{Request: &HTTPRequest{Url: "https://x", Method: "GET"}}
	WithSecrets(ids...)(cr)
	WithAuth(auth)(cr)

	if len(cr.VaultDonSecrets) != 2 {
		t.Fatalf("secrets=%d", len(cr.VaultDonSecrets))
	}
	if cr.GetAuth() == nil || cr.GetAuth().GetApiKey() == nil {
		t.Fatalf("auth not attached")
	}
}
