package confidentialhttp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

const (
	// EncryptionKeySecretName is the VaultDON secret name used for AES-GCM encryption
	// of confidential HTTP responses.
	EncryptionKeySecretName = "san_marino_aes_gcm_encryption_key"

	// hkdfInfo is the context string for HKDF key derivation.
	hkdfInfo = "confidential-http-encryption-key-v1"

	aesKeyLen  = 32 // AES-256
	gcmNonceLen = 12
	gcmTagLen   = 16
)

// DeriveEncryptionKey applies HKDF-SHA256 to a user passphrase and returns a
// 32-byte AES-256 key. The derivation uses no salt and
// "confidential-http-encryption-key-v1" as the info parameter.
func DeriveEncryptionKey(passphrase string) ([]byte, error) {
	r := hkdf.New(sha256.New, []byte(passphrase), nil, []byte(hkdfInfo))
	key := make([]byte, aesKeyLen)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, fmt.Errorf("hkdf expand: %w", err)
	}
	return key, nil
}

// NewRequestForEncryptedResponse wraps an HTTPRequest with EncryptOutput=true and the AES
// key SecretIdentifier auto-injected. Owner is the workflow owner address.
func NewRequestForEncryptedResponse(req *HTTPRequest, owner string) *ConfidentialHTTPRequest {
	req.EncryptOutput = true
	return &ConfidentialHTTPRequest{
		Request: req,
		VaultDonSecrets: []*SecretIdentifier{
			{
				Key:   EncryptionKeySecretName,
				Owner: &owner,
			},
		},
	}
}

// DecryptResponseBody decrypts an AES-GCM encrypted response body using the
// same passphrase that was used to store the encryption key.
//
// Wire format: [12-byte nonce][ciphertext+16-byte GCM tag]
func DecryptResponseBody(ciphertext []byte, passphrase string) ([]byte, error) {
	if len(ciphertext) < gcmNonceLen+gcmTagLen {
		return nil, fmt.Errorf("ciphertext too short: need at least %d bytes, got %d", gcmNonceLen+gcmTagLen, len(ciphertext))
	}

	key, err := DeriveEncryptionKey(passphrase)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cipher.NewGCM: %w", err)
	}

	nonce := ciphertext[:gcmNonceLen]
	sealed := ciphertext[gcmNonceLen:]
	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, fmt.Errorf("gcm decrypt: %w", err)
	}

	return plaintext, nil
}
