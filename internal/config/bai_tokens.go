package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// keychainService is the OS keychain service name bai stores its credentials
// under (accounts "access_token" and "refresh_token"). abaper reads/writes
// the same entries so a single `bai login` authenticates both tools.
const keychainService = "bai"

// baiAuth mirrors the `auth:` block of ~/.bai/config.yaml. When the OS
// keychain is available, bai leaves these fields empty and stores the real
// values in the keychain; otherwise it writes them here directly.
type baiAuth struct {
	AccessToken  string    `yaml:"access_token"`
	RefreshToken string    `yaml:"refresh_token"`
	TokenExpiry  time.Time `yaml:"token_expiry"`
}

// baiConfig mirrors ~/.bai/config.yaml. abaper only reads/updates the auth
// block; the rest is preserved as-is on save.
type baiConfig struct {
	Gateway  string         `yaml:"gateway"`
	Endpoint string         `yaml:"endpoint"`
	Domain   string         `yaml:"domain"`
	Realm    string         `yaml:"realm"`
	Auth     baiAuth        `yaml:"auth"`
	Defaults map[string]any `yaml:"defaults"`
}

// keychainGet/keychainSet are swappable so tests never touch the real OS
// keychain.
var (
	keychainGet = defaultKeychainGet
	keychainSet = defaultKeychainSet
)

func defaultKeychainGet(account string) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("security", "find-generic-password", "-s", keychainService, "-a", account, "-w").Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	case "linux":
		out, err := exec.Command("secret-tool", "lookup", "service", keychainService, "account", account).Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	default:
		return "", fmt.Errorf("OS keychain not supported on %s", runtime.GOOS)
	}
}

func defaultKeychainSet(account, value string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("security", "add-generic-password", "-s", keychainService, "-a", account, "-w", value, "-U").Run()
	case "linux":
		cmd := exec.Command("secret-tool", "store", "--label="+keychainService, "service", keychainService, "account", account)
		cmd.Stdin = strings.NewReader(value)
		return cmd.Run()
	default:
		return fmt.Errorf("OS keychain not supported on %s", runtime.GOOS)
	}
}

func baiConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".bai", "config.yaml")
}

func loadBaiConfig() (*baiConfig, error) {
	data, err := os.ReadFile(baiConfigPath())
	if err != nil {
		return nil, fmt.Errorf("read bai config: %w", err)
	}
	bc := &baiConfig{}
	if err := yaml.Unmarshal(data, bc); err != nil {
		return nil, fmt.Errorf("parse bai config: %w", err)
	}
	return bc, nil
}

func saveBaiConfig(bc *baiConfig) error {
	data, err := yaml.Marshal(bc)
	if err != nil {
		return fmt.Errorf("marshal bai config: %w", err)
	}
	return os.WriteFile(baiConfigPath(), data, 0600)
}

// resolveSecret returns the keychain value for account if present, else
// falls back to the plaintext value bai wrote to its config file. bai's
// encrypted-at-rest fallback (used when no keychain is available) is
// prefixed "enc:" and can't be decrypted outside bai itself.
func resolveSecret(account, fallback string) (string, error) {
	if v, err := keychainGet(account); err == nil && v != "" {
		return v, nil
	}
	if strings.HasPrefix(fallback, "enc:") {
		return "", fmt.Errorf("bai credentials are encrypted without OS keychain access — enable your OS keychain (or set BAI_NO_KEYCHAIN=0) and re-run 'bai login'")
	}
	return fallback, nil
}

// LoadTokens reads the credentials `bai login` stored, from the OS keychain
// (service "bai") or ~/.bai/config.yaml. abaper keeps no token store of its
// own so that `abaper login`/`bai login` are a single shared session.
func LoadTokens() (*Tokens, error) {
	bc, err := loadBaiConfig()
	if err != nil {
		return nil, fmt.Errorf("not logged in — run 'abaper login': %w", err)
	}

	access, err := resolveSecret("access_token", bc.Auth.AccessToken)
	if err != nil {
		return nil, err
	}
	if access == "" {
		return nil, fmt.Errorf("not logged in — run 'abaper login'")
	}

	refresh, err := resolveSecret("refresh_token", bc.Auth.RefreshToken)
	if err != nil {
		return nil, err
	}

	realm := bc.Realm
	if realm == "" {
		realm = DefaultRealm
	}

	return &Tokens{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    bc.Auth.TokenExpiry.UnixMilli(),
		Realm:        realm,
	}, nil
}

// SaveTokens persists a refreshed access/refresh token pair back into bai's
// shared store (keychain when available, else the config file directly),
// keeping both tools in sync on the same session.
func SaveTokens(tokens *Tokens) error {
	bc, err := loadBaiConfig()
	if err != nil {
		return fmt.Errorf("load bai config: %w", err)
	}

	bc.Auth.TokenExpiry = time.UnixMilli(tokens.ExpiresAt)

	if err := keychainSet("access_token", tokens.AccessToken); err == nil {
		bc.Auth.AccessToken = ""
	} else {
		bc.Auth.AccessToken = tokens.AccessToken
	}
	if err := keychainSet("refresh_token", tokens.RefreshToken); err == nil {
		bc.Auth.RefreshToken = ""
	} else {
		bc.Auth.RefreshToken = tokens.RefreshToken
	}

	return saveBaiConfig(bc)
}
