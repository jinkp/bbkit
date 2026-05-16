package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	configpkg "github.com/jinkp/bbkit/internal/config"
	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/scrypt"
)

var (
	keyringGet    = keyring.Get
	keyringSet    = keyring.Set
	keyringDelete = keyring.Delete
)

const (
	serviceName         = "bbkit"
	credentialsFileName = "credentials.enc"
	credentialsSalt     = "bbkit-cli-v1-salt-2024"
	envUsername         = "BITBUCKET_USERNAME"
	envAPIToken         = "BITBUCKET_API_TOKEN"
)

type Credentials struct {
	Username string `json:"username"`
	APIToken string `json:"apiToken"`
}

func Get() (*Credentials, error) {
	creds, _, err := GetWithSource()
	return creds, err
}

func GetWithSource() (*Credentials, string, error) {
	if username, token := envCredentials(); username != "" && token != "" {
		return &Credentials{Username: username, APIToken: token}, "env", nil
	}

	username, err := configpkg.GetUsername()
	if err != nil {
		return nil, "", err
	}

	if username != "" {
		token, err := keyringGet(serviceName, username)
		if err == nil && token != "" {
			return &Credentials{Username: username, APIToken: token}, "keyring", nil
		}
	}

	creds, err := loadFromFile()
	if err != nil || creds == nil {
		return creds, "", err
	}

	return creds, "file", nil
}

func Save(creds Credentials) error {
	if creds.Username == "" {
		return errors.New("credentials username is required")
	}
	if creds.APIToken == "" {
		return errors.New("credentials api token is required")
	}

	if err := keyringSet(serviceName, creds.Username, creds.APIToken); err == nil {
		return nil
	}

	return saveToFile(creds)
}

func Clear() error {
	usernames, err := knownUsernames()
	if err != nil {
		return err
	}

	var clearErr error
	for _, username := range usernames {
		if username == "" {
			continue
		}
		if err := keyringDelete(serviceName, username); err != nil && !errors.Is(err, keyring.ErrNotFound) {
			clearErr = errors.Join(clearErr, fmt.Errorf("delete keyring credentials for %s: %w", username, err))
		}
	}

	if err := clearFile(); err != nil {
		clearErr = errors.Join(clearErr, err)
	}

	return clearErr
}

func envCredentials() (string, string) {
	return os.Getenv(envUsername), os.Getenv(envAPIToken)
}

func credentialsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}

	return filepath.Join(dir, "bbkit", credentialsFileName), nil
}

func saveToFile(creds Credentials) error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create credentials dir: %w", err)
	}

	plaintext, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	payload, err := encrypt(plaintext)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		return fmt.Errorf("write encrypted credentials: %w", err)
	}

	return nil
}

func loadFromFile() (*Credentials, error) {
	path, err := credentialsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read encrypted credentials: %w", err)
	}

	if len(data) == 0 {
		return nil, nil
	}

	plaintext, err := decrypt(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, err
	}

	var creds Credentials
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return nil, fmt.Errorf("decode credentials payload: %w", err)
	}

	if creds.Username == "" || creds.APIToken == "" {
		return nil, nil
	}

	return &creds, nil
}

func clearFile() error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete encrypted credentials: %w", err)
	}

	return nil
}

func knownUsernames() ([]string, error) {
	seen := map[string]struct{}{}
	usernames := make([]string, 0, 3)

	add := func(value string) {
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		usernames = append(usernames, value)
	}

	if username := os.Getenv(envUsername); username != "" {
		add(username)
	}

	configUsername, err := configpkg.GetUsername()
	if err != nil {
		return nil, err
	}
	add(configUsername)

	fileCreds, err := loadFromFile()
	if err != nil {
		return nil, err
	}
	if fileCreds != nil {
		add(fileCreds.Username)
	}

	return usernames, nil
}

func deriveKey() ([]byte, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("resolve hostname: %w", err)
	}

	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("resolve current user: %w", err)
	}

	machineID := fmt.Sprintf("%s-%s", hostname, currentUser.Username)
	key, err := scrypt.Key([]byte(machineID), []byte(credentialsSalt), 1<<15, 8, 1, 32)
	if err != nil {
		return nil, fmt.Errorf("derive encryption key: %w", err)
	}

	return key, nil
}

func encrypt(plaintext []byte) (string, error) {
	key, err := deriveKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCMWithNonceSize(block, 16)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	sealed := gcm.Seal(nil, nonce, plaintext, nil)
	tagSize := gcm.Overhead()
	ciphertext := sealed[:len(sealed)-tagSize]
	authTag := sealed[len(sealed)-tagSize:]

	return hex.EncodeToString(nonce) + hex.EncodeToString(authTag) + hex.EncodeToString(ciphertext), nil
}

func decrypt(payload string) ([]byte, error) {
	if len(payload) < 64 {
		return nil, errors.New("invalid encrypted credentials payload")
	}

	key, err := deriveKey()
	if err != nil {
		return nil, err
	}

	nonce, err := hex.DecodeString(payload[:32])
	if err != nil {
		return nil, fmt.Errorf("decode credentials nonce: %w", err)
	}

	authTag, err := hex.DecodeString(payload[32:64])
	if err != nil {
		return nil, fmt.Errorf("decode credentials auth tag: %w", err)
	}

	ciphertext, err := hex.DecodeString(payload[64:])
	if err != nil {
		return nil, fmt.Errorf("decode credentials ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCMWithNonceSize(block, len(nonce))
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, append(ciphertext, authTag...), nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt credentials: %w", err)
	}

	return plaintext, nil
}
