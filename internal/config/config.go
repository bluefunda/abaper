package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	DefaultBaseURL = "https://api.bluefunda.com"
	DefaultRealm   = "individual"
	// ClientID matches bai's own Keycloak client ID, so tokens issued by
	// `bai login` are valid for the ABAPer gateway too — see LoadTokens.
	ClientID   = "bai"
	ConfigDir  = ".abaper"
	ConfigFile = "config"
)

type Config struct {
	BaseURL string `mapstructure:"base_url"`
	Org     string `mapstructure:"org"`
	Realm   string `mapstructure:"realm"`
}

// Tokens is abaper's in-memory view of the credentials `bai login` stores.
// abaper has no token store of its own — see LoadTokens/SaveTokens.
type Tokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
	Realm        string
}

func ConfigDirPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ConfigDir)
}

func Init() {
	configDir := ConfigDirPath()

	viper.SetConfigName(ConfigFile)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)

	viper.SetDefault("base_url", DefaultBaseURL)
	viper.SetDefault("org", "default")
	viper.SetDefault("realm", DefaultRealm)

	viper.SetEnvPrefix("ABAPER")
	viper.AutomaticEnv()

	_ = viper.ReadInConfig()
}

func Load() *Config {
	return &Config{
		BaseURL: viper.GetString("base_url"),
		Org:     viper.GetString("org"),
		Realm:   viper.GetString("realm"),
	}
}

func EnsureConfigDir() error {
	dir := ConfigDirPath()
	return os.MkdirAll(dir, 0700)
}
