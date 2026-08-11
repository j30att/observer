package encryption_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/encryption"
)

func TestRSAEncryption(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	t.Run("Должен выполниться без ошибок", func(t *testing.T) {
		message := make([]byte, 1024)
		_, err := rand.Read(message)
		require.NoError(t, err)

		encrypted, err := encryption.Encrypt(message, &privateKey.PublicKey)
		require.NoError(t, err)

		decrypted, err := encryption.Decrypt(encrypted, privateKey)

		require.NoError(t, err)
		assert.Equal(t, message, decrypted)
	})

	t.Run("Должен загрузить публичный и приватный ключи из PEM", func(t *testing.T) {
		directory := t.TempDir()
		publicKeyPath := filepath.Join(directory, "public.pem")
		privateKeyPath := filepath.Join(directory, "private.pem")

		publicKeyData, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
		require.NoError(t, err)
		privateKeyData, err := x509.MarshalPKCS8PrivateKey(privateKey)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(publicKeyPath, pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKeyData,
		}), 0o600))
		require.NoError(t, os.WriteFile(privateKeyPath, pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privateKeyData,
		}), 0o600))

		publicKey, err := encryption.LoadPublicKey(publicKeyPath)
		require.NoError(t, err)
		loadedPrivateKey, err := encryption.LoadPrivateKey(privateKeyPath)

		require.NoError(t, err)
		assert.Equal(t, privateKey.PublicKey.N, publicKey.N)
		assert.Equal(t, privateKey.N, loadedPrivateKey.N)
	})

	t.Run("Ошибка, поврежденное зашифрованное сообщение", func(t *testing.T) {
		_, err := encryption.Decrypt([]byte("invalid"), privateKey)

		require.EqualError(t, err, "decrypt data: invalid encrypted payload size")
	})

	t.Run("Ошибка, файл не содержит PEM ключ", func(t *testing.T) {
		keyPath := filepath.Join(t.TempDir(), "invalid.pem")
		require.NoError(t, os.WriteFile(keyPath, []byte("invalid"), 0o600))

		_, publicKeyErr := encryption.LoadPublicKey(keyPath)
		_, privateKeyErr := encryption.LoadPrivateKey(keyPath)

		require.EqualError(t, publicKeyErr, "decode public key PEM: no PEM block found")
		require.EqualError(t, privateKeyErr, "decode private key PEM: no PEM block found")
	})
}
