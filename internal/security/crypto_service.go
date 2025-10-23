package security

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
)

type CryptoService interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
	Sign(payload []byte) ([]byte, error)
	Verify(payload, signature []byte) error
}

// ---- Implementation ----

type cryptoService struct {
	aead      cipher.AEAD // AES-256-GCM
	nonceSize int         // e.g., 12 (nonce = iv "intialization vector")
	rsaPub    *rsa.PublicKey
	rsaPriv   *rsa.PrivateKey // optional; nil => verify-only
}

func NewCryptoService(cm *CryptoMaterial) (CryptoService, error) {
	if len(cm.AESKey) != 32 {
		return nil, fmt.Errorf("aes key must be 32 bytes, got %d", len(cm.AESKey))
	}
	if cm.RSAPub == nil {
		return nil, errors.New("rsa public key required")
	}

	block, err := aes.NewCipher(cm.AESKey)
	if err != nil {
		return nil, fmt.Errorf("aes.NewCipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cipher.NewGCM: %w", err)
	}

	return &cryptoService{
		aead:      aead,
		nonceSize: aead.NonceSize(),
		rsaPub:    cm.RSAPub,
		rsaPriv:   cm.RSAPri,
	}, nil
}

func (cs *cryptoService) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, cs.nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("rand nonce: %w", err)
	}
	ct := cs.aead.Seal(nil, nonce, plaintext, nil)

	// concat: nonce || ct
	out := make([]byte, 0, len(nonce)+len(ct))
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

func (cs *cryptoService) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < cs.nonceSize+cs.aead.Overhead() {
		return nil, errors.New("ciphertext too short")
	}
	nonce := ciphertext[:cs.nonceSize]
	ct := ciphertext[cs.nonceSize:]
	pt, err := cs.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("gcm open: %w", err)
	}
	return pt, nil
}

func (cs *cryptoService) Sign(payload []byte) ([]byte, error) {
	if cs.rsaPriv == nil {
		return nil, errors.New("signing not configured (no RSA private key)")
	}
	sum := sha256.Sum256(payload)
	sig, err := rsa.SignPKCS1v15(rand.Reader, cs.rsaPriv, crypto.SHA256, sum[:])
	if err != nil {
		return nil, fmt.Errorf("rsa sign: %w", err)
	}
	return sig, nil
}

func (cs *cryptoService) Verify(payload, signature []byte) error {
	sum := sha256.Sum256(payload)
	if err := rsa.VerifyPKCS1v15(cs.rsaPub, crypto.SHA256, sum[:], signature); err != nil {
		return fmt.Errorf("rsa verify: %w", err)
	}
	return nil
}
