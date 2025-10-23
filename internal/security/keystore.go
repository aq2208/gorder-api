package security

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/aq2208/gorder-api/configs"
)

type CryptoMaterial struct {
	KeyID  string
	AESKey []byte
	RSAPub *rsa.PublicKey
	RSAPri *rsa.PrivateKey
}

func NewCryptoMaterial(c configs.Config) (*CryptoMaterial, error) {
	cm, err := LoadCryptoMaterial(c)
	return &cm, err
}

func LoadCryptoMaterial(c configs.Config) (CryptoMaterial, error) {
	if c.CryptoConfig.AES256B64 == "" || c.CryptoConfig.RSAPubPEM == "" {
		return CryptoMaterial{}, errors.New("missing aes256_b64url or rsa_pub_pem")
	}
	// --- AES-256 key ---
	key, err := base64.RawURLEncoding.DecodeString(c.CryptoConfig.AES256B64)
	if err != nil {
		return CryptoMaterial{}, fmt.Errorf("decode aes256_b64url: %w", err)
	}
	if len(key) != 32 {
		return CryptoMaterial{}, fmt.Errorf("aes key must be 32 bytes, got %d", len(key))
	}

	// --- RSA public key ---
	pub, err := parseRSAPublicKeyFromPEM([]byte(c.CryptoConfig.RSAPubPEM))
	if err != nil {
		return CryptoMaterial{}, fmt.Errorf("parse rsa pub pem: %w", err)
	}

	// --- RSA private key (optional) ---
	var pri *rsa.PrivateKey
	if c.CryptoConfig.RSAPriPEM != "" {
		pri, err = parseRSAPrivateKeyFromPEM([]byte(c.CryptoConfig.RSAPriPEM))
		if err != nil {
			return CryptoMaterial{}, fmt.Errorf("parse rsa pri pem: %w", err)
		}
	}

	id := c.CryptoConfig.KeyID
	if id == "" {
		id = "v1"
	}
	return CryptoMaterial{
		KeyID:  id,
		AESKey: key,
		RSAPub: pub,
		RSAPri: pri,
	}, nil
}

func parseRSAPublicKeyFromPEM(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("no pem block")
	}
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := pubAny.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not rsa public key")
	}
	return pub, nil
}

func parseRSAPrivateKeyFromPEM(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("no pem block in RSA private key")
	}

	// try PKCS#8 first
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
		return nil, errors.New("not an RSA private key in PKCS#8")
	}

	// fallback to PKCS#1
	rsaKey, err2 := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err2 != nil {
		return nil, fmt.Errorf("parse RSA private key failed (PKCS#8: %v, PKCS#1: %v)", err, err2)
	}
	return rsaKey, nil
}
