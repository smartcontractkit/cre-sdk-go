package confidentialhttp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"testing"
)

// Shared cross-language test vector. The TypeScript SDK must produce the same output.
const (
	testPassphrase  = "test-passphrase-for-ci"
	testExpectedHex = "521af99325c07c9bd0d224c5bf3ca25666c68b5fbb7fa7884019b4f60a8e6eb5"
)

func TestDeriveEncryptionKey_Deterministic(t *testing.T) {
	k1, err := DeriveEncryptionKey("my-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	k2, err := DeriveEncryptionKey("my-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(k1) != hex.EncodeToString(k2) {
		t.Fatal("same passphrase produced different keys")
	}
}

func TestDeriveEncryptionKey_DifferentPassphrases(t *testing.T) {
	k1, err := DeriveEncryptionKey("passphrase-a")
	if err != nil {
		t.Fatal(err)
	}
	k2, err := DeriveEncryptionKey("passphrase-b")
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(k1) == hex.EncodeToString(k2) {
		t.Fatal("different passphrases produced the same key")
	}
}

func TestDeriveEncryptionKey_CrossLanguageVector(t *testing.T) {
	key, err := DeriveEncryptionKey(testPassphrase)
	if err != nil {
		t.Fatal(err)
	}
	got := hex.EncodeToString(key)
	if got != testExpectedHex {
		t.Fatalf("HKDF vector mismatch:\n  got:  %s\n  want: %s", got, testExpectedHex)
	}
}

func TestNewRequestForEncryptedResponse(t *testing.T) {
	req := &HTTPRequest{
		Url:    "https://example.com",
		Method: "GET",
	}
	owner := "0xDeaDBeeF"
	cr := NewRequestForEncryptedResponse(req, owner)

	if !cr.Request.EncryptOutput {
		t.Fatal("EncryptOutput should be true")
	}
	if len(cr.VaultDonSecrets) != 1 {
		t.Fatalf("expected 1 secret identifier, got %d", len(cr.VaultDonSecrets))
	}
	sid := cr.VaultDonSecrets[0]
	if sid.Key != EncryptionKeySecretName {
		t.Fatalf("secret key = %q, want %q", sid.Key, EncryptionKeySecretName)
	}
	if sid.GetOwner() != owner {
		t.Fatalf("secret owner = %q, want %q", sid.GetOwner(), owner)
	}
}

func TestDecryptResponseBody_RoundTrip(t *testing.T) {
	passphrase := "round-trip-test"
	plaintext := []byte("hello confidential http")

	key, err := DeriveEncryptionKey(passphrase)
	if err != nil {
		t.Fatal(err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	sealed := gcm.Seal(nonce, nonce, plaintext, nil)

	got, err := DecryptResponseBody(sealed, passphrase)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(plaintext) {
		t.Fatalf("decrypted = %q, want %q", got, plaintext)
	}
}

func TestDecryptResponseBody_TooShort(t *testing.T) {
	_, err := DecryptResponseBody(make([]byte, 10), "any")
	if err == nil {
		t.Fatal("expected error for short ciphertext")
	}
}

func TestDecryptResponseBody_WrongPassphrase(t *testing.T) {
	passphrase := "correct-passphrase"
	plaintext := []byte("secret data")

	key, err := DeriveEncryptionKey(passphrase)
	if err != nil {
		t.Fatal(err)
	}

	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	sealed := gcm.Seal(nonce, nonce, plaintext, nil)

	_, err = DecryptResponseBody(sealed, "wrong-passphrase")
	if err == nil {
		t.Fatal("expected error when decrypting with wrong passphrase")
	}
}
