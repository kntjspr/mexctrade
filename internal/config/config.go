package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	toml "github.com/pelletier/go-toml/v2"
)

type Config struct {
	APIKey      string `toml:"api_key"`
	APISecret   string `toml:"api_secret"`
	BaseURL     string `toml:"base_url"`
	DryRun      bool   `toml:"dry_run"`
	MaxLeverage int    `toml:"max_leverage"`
}

func defaults() Config {
	return Config{
		BaseURL:     "https://contract.mexc.com",
		MaxLeverage: 20,
	}
}

// Load reads config from path (if present) and overlays env vars.
// Returns an error if (a) file exists with mode != 0600,
// or (b) no API key/secret is found in file or env.
func Load(path string) (*Config, error) {
	cfg := defaults()

	if path != "" {
		st, err := os.Stat(path)
		if err == nil {
			if st.Mode().Perm() != 0o600 {
				return nil, fmt.Errorf("config %s must be mode 0600, got %o", path, st.Mode().Perm())
			}
			body, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read config: %w", err)
			}
			if err := toml.Unmarshal(body, &cfg); err != nil {
				return nil, fmt.Errorf("parse config: %w", err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("stat config: %w", err)
		}
	}

	if v := os.Getenv("MEXC_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("MEXC_API_SECRET"); v != "" {
		cfg.APISecret = v
	}
	if v := os.Getenv("MEXCTRADE_DRY_RUN"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("MEXCTRADE_DRY_RUN: %w", err)
		}
		cfg.DryRun = b
	}
	if v := os.Getenv("MEXCTRADE_MAX_LEVERAGE"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("MEXCTRADE_MAX_LEVERAGE: %w", err)
		}
		cfg.MaxLeverage = n
	}

	if cfg.APIKey == "" || cfg.APISecret == "" {
		return nil, errors.New("api_key and api_secret required (in config file or env)")
	}
	return &cfg, nil
}
