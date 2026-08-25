package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/BurntSushi/toml"
)

type Config struct {
	GhToken string `toml:"gh_token" env:"BAG_GH_TOKEN"`
}

// defaultConfig is the default config, used as a fallback if options not set in config file or env vars
var defaultConfig = Config{
	GhToken: "",
}

// loaded holds the config after overrides from config file and env vars
var loaded *Config

// Load reads the config file and env vars, and applies them to the loaded config
func Load() error {
	// prevent mutation of defaultConfig
	cfg := defaultConfig
	loaded = &cfg
	configFilePath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("getting config path: %w", err)
	}
	// read config file
	if _, err := toml.DecodeFile(configFilePath, loaded); err != nil {
		// missing config file is fine
		// if it doesn't exist, it just doesn't override default config
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("loading config file: %w", err)
		}
	}
	applyEnv(loaded)
	return nil
}

// Get returns the loaded config after config file and env vars have been applied
func Get() *Config {
	return loaded
}

// applyEnv gets relevant env variables and applies them to the config
func applyEnv(cfg *Config) {
	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()
	for i := range t.NumField() {
		key := t.Field(i).Tag.Get("env")
		if key == "" {
			continue
		}
		if val := os.Getenv(key); val != "" {
			// relies on all config fields being strings
			// if they aren't, this will panic
			v.Field(i).SetString(val)
		}
	}
}

// getConfigPath returns the path to the config file
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home dir: %w", err)
	}
	return filepath.Join(homeDir, ".config/bag/config.toml"), nil
}
