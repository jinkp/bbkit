package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	configpkg "github.com/jinkp/bbkit/internal/config"
	"github.com/stretchr/testify/require"
)

func TestGetReturnsEnvCredentialsWhenBothEnvVarsAreSet(t *testing.T) {
	setSecretsConfigHome(t)
	stubKeyring(t)
	t.Setenv(envUsername, "env-user")
	t.Setenv(envAPIToken, "env-token")

	creds, source, err := GetWithSource()
	require.NoError(t, err)
	require.Equal(t, "env", source)
	require.Equal(t, &Credentials{Username: "env-user", APIToken: "env-token"}, creds)
}

func TestGetDoesNotPersistEnvCredentials(t *testing.T) {
	path := setSecretsConfigHome(t)
	stubKeyring(t)
	t.Setenv(envUsername, "env-user")
	t.Setenv(envAPIToken, "env-token")

	creds, err := Get()
	require.NoError(t, err)
	require.Equal(t, &Credentials{Username: "env-user", APIToken: "env-token"}, creds)
	_, statErr := os.Stat(path)
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestEncryptionRoundTripUsesAES256GCM(t *testing.T) {
	path := setSecretsConfigHome(t)
	stubKeyring(t)

	// deriveKey uses os.Hostname and user.Current — skip if unavailable (e.g. CI containers)
	if _, err := deriveKey(); err != nil {
		t.Skipf("skipping: deriveKey unavailable in this environment: %v", err)
	}

	creds := Credentials{Username: "testuser", APIToken: "secret123"}
	require.NoError(t, Save(creds))

	payload, err := os.ReadFile(path)
	require.NoError(t, err)
	plaintext := decryptPayloadManually(t, string(payload))

	var decoded Credentials
	require.NoError(t, json.Unmarshal(plaintext, &decoded))
	require.Equal(t, creds, decoded)
}

func setSecretsConfigHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	require.NoError(t, configpkg.Save(&configpkg.Config{Username: "testuser"}))
	return filepath.Join(home, "bbkit", credentialsFileName)
}

func stubKeyring(t *testing.T) {
	t.Helper()
	originalGet, originalSet, originalDelete := keyringGet, keyringSet, keyringDelete
	t.Cleanup(func() {
		keyringGet = originalGet
		keyringSet = originalSet
		keyringDelete = originalDelete
	})
	keyringGet = func(service, user string) (string, error) { return "", os.ErrNotExist }
	keyringSet = func(service, user, password string) error { return os.ErrPermission }
	keyringDelete = func(service, user string) error { return nil }
}

func decryptPayloadManually(t *testing.T, payload string) []byte {
	t.Helper()
	require.GreaterOrEqual(t, len(payload), 64)

	nonce, err := hex.DecodeString(payload[:32])
	require.NoError(t, err)
	authTag, err := hex.DecodeString(payload[32:64])
	require.NoError(t, err)
	ciphertext, err := hex.DecodeString(payload[64:])
	require.NoError(t, err)

	key, err := deriveKey()
	require.NoError(t, err)
	require.Len(t, key, 32)

	block, err := aes.NewCipher(key)
	require.NoError(t, err)
	gcm, err := cipher.NewGCMWithNonceSize(block, len(nonce))
	require.NoError(t, err)

	plaintext, err := gcm.Open(nil, nonce, append(ciphertext, authTag...), nil)
	require.NoError(t, err)
	return plaintext
}
