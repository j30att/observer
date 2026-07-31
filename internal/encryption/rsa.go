package encryption

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// Encoding is the HTTP content-encoding token used for RSA-encrypted bodies.
const Encoding = "rsa"

// LoadPublicKey reads an RSA public key from a PEM file.
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("decode public key PEM: no PEM block found")
	}

	if key, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	key, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("parse public key: key is not RSA")
	}

	return key, nil
}

// LoadPrivateKey reads an RSA private key from a PEM file.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("decode private key PEM: no PEM block found")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("parse private key: key is not RSA")
	}

	return key, nil
}

// Encrypt encrypts data in RSA-OAEP blocks using SHA-256.
func Encrypt(data []byte, key *rsa.PublicKey) ([]byte, error) {
	if key == nil {
		return nil, errors.New("encrypt data: public key is nil")
	}
	if len(data) == 0 {
		return []byte{}, nil
	}

	hash := sha256.New()
	chunkSize := key.Size() - 2*hash.Size() - 2
	if chunkSize <= 0 {
		return nil, errors.New("encrypt data: RSA key is too small for OAEP-SHA256")
	}

	blocksCount := (len(data) + chunkSize - 1) / chunkSize
	result := make([]byte, 0, blocksCount*key.Size())
	for start := 0; start < len(data); start += chunkSize {
		end := min(start+chunkSize, len(data))
		block, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, key, data[start:end], nil)
		if err != nil {
			return nil, fmt.Errorf("encrypt data block: %w", err)
		}
		result = append(result, block...)
	}

	return result, nil
}

// Decrypt decrypts fixed-size RSA-OAEP blocks using SHA-256.
func Decrypt(data []byte, key *rsa.PrivateKey) ([]byte, error) {
	if key == nil {
		return nil, errors.New("decrypt data: private key is nil")
	}
	if len(data) == 0 {
		return []byte{}, nil
	}
	if len(data)%key.Size() != 0 {
		return nil, errors.New("decrypt data: invalid encrypted payload size")
	}

	result := make([]byte, 0, len(data))
	for start := 0; start < len(data); start += key.Size() {
		block, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, key, data[start:start+key.Size()], nil)
		if err != nil {
			return nil, fmt.Errorf("decrypt data block: %w", err)
		}
		result = append(result, block...)
	}

	return result, nil
}
