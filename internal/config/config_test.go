package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeBaiConfig writes a minimal ~/.bai/config.yaml fixture under the
// current (test-scoped) $HOME.
func writeBaiConfig(t *testing.T, yamlBody string) {
	t.Helper()
	dir := filepath.Join(os.Getenv("HOME"), ".bai")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("mkdir bai config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(yamlBody), 0600); err != nil {
		t.Fatalf("write bai config: %v", err)
	}
}

// stubKeychain replaces the keychain get/set seams for the duration of a
// test so tokens never touch the real OS keychain.
func stubKeychain(t *testing.T, store map[string]string) {
	t.Helper()
	origGet, origSet := keychainGet, keychainSet
	keychainGet = func(account string) (string, error) {
		if v, ok := store[account]; ok {
			return v, nil
		}
		return "", os.ErrNotExist
	}
	keychainSet = func(account, value string) error {
		store[account] = value
		return nil
	}
	t.Cleanup(func() {
		keychainGet, keychainSet = origGet, origSet
	})
}

func TestLoadTokens_NotLoggedIn(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stubKeychain(t, map[string]string{})

	if _, err := LoadTokens(); err == nil {
		t.Fatal("expected error when ~/.bai/config.yaml does not exist")
	}
}

func TestLoadTokens_PlaintextFallback(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stubKeychain(t, map[string]string{}) // no keychain entries — falls back to yaml fields

	writeBaiConfig(t, `
realm: individual
domain: bluefunda.com
auth:
    access_token: access-123
    refresh_token: refresh-456
    token_expiry: 2030-01-01T00:00:00Z
`)

	tokens, err := LoadTokens()
	if err != nil {
		t.Fatalf("load tokens: %v", err)
	}
	if tokens.AccessToken != "access-123" || tokens.RefreshToken != "refresh-456" {
		t.Errorf("unexpected tokens: %+v", tokens)
	}
	if tokens.Realm != "individual" {
		t.Errorf("expected realm individual, got %q", tokens.Realm)
	}
	wantExpiry := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	if tokens.ExpiresAt != wantExpiry {
		t.Errorf("expected expiry %d, got %d", wantExpiry, tokens.ExpiresAt)
	}
}

func TestLoadTokens_PrefersKeychain(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stubKeychain(t, map[string]string{
		"access_token":  "keychain-access",
		"refresh_token": "keychain-refresh",
	})

	writeBaiConfig(t, `
realm: individual
auth:
    access_token: ""
    refresh_token: ""
    token_expiry: 2030-01-01T00:00:00Z
`)

	tokens, err := LoadTokens()
	if err != nil {
		t.Fatalf("load tokens: %v", err)
	}
	if tokens.AccessToken != "keychain-access" || tokens.RefreshToken != "keychain-refresh" {
		t.Errorf("expected keychain values, got %+v", tokens)
	}
}

func TestLoadTokens_EncryptedWithoutKeychainErrors(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stubKeychain(t, map[string]string{})

	writeBaiConfig(t, `
realm: individual
auth:
    access_token: "enc:unreadable"
    refresh_token: "enc:unreadable"
    token_expiry: 2030-01-01T00:00:00Z
`)

	if _, err := LoadTokens(); err == nil {
		t.Fatal("expected error for encrypted token without keychain access")
	}
}

func TestSaveTokens_RoundTripsThroughKeychain(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	store := map[string]string{}
	stubKeychain(t, store)

	writeBaiConfig(t, `
gateway: https://ai.bluefunda.com
realm: individual
auth:
    access_token: ""
    refresh_token: ""
    token_expiry: 2020-01-01T00:00:00Z
`)

	saved := &Tokens{
		AccessToken:  "new-access",
		RefreshToken: "new-refresh",
		ExpiresAt:    time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
		Realm:        "individual",
	}
	if err := SaveTokens(saved); err != nil {
		t.Fatalf("save tokens: %v", err)
	}

	if store["access_token"] != "new-access" || store["refresh_token"] != "new-refresh" {
		t.Errorf("expected keychain to hold new tokens, got %+v", store)
	}

	loaded, err := LoadTokens()
	if err != nil {
		t.Fatalf("load tokens: %v", err)
	}
	if loaded.AccessToken != saved.AccessToken || loaded.RefreshToken != saved.RefreshToken || loaded.ExpiresAt != saved.ExpiresAt {
		t.Errorf("loaded tokens do not match saved tokens: %+v", loaded)
	}

	// bai's config file itself must not retain the raw secrets once the
	// keychain store succeeds.
	bc, err := loadBaiConfig()
	if err != nil {
		t.Fatalf("load bai config: %v", err)
	}
	if bc.Auth.AccessToken != "" || bc.Auth.RefreshToken != "" {
		t.Errorf("expected bai config auth fields to stay empty when keychain is used, got %+v", bc.Auth)
	}
	// Unrelated fields must be preserved.
	if bc.Gateway != "https://ai.bluefunda.com" {
		t.Errorf("expected unrelated bai config fields to survive save, got %+v", bc)
	}
}

func TestConfigDirPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	want := filepath.Join(home, ConfigDir)
	if got := ConfigDirPath(); got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}
