package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, contents string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	writeFile(t, path, `
api_key = "k"
api_secret = "s"
`, 0o600)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.APIKey != "k" || cfg.APISecret != "s" {
		t.Errorf("creds wrong: %+v", cfg)
	}
	if cfg.BaseURL != "https://contract.mexc.com" {
		t.Errorf("BaseURL default wrong: %q", cfg.BaseURL)
	}
	if cfg.MaxLeverage != 20 {
		t.Errorf("MaxLeverage default wrong: %d", cfg.MaxLeverage)
	}
	if cfg.DryRun != false {
		t.Errorf("DryRun default wrong")
	}
}

func TestEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	writeFile(t, path, `api_key="file-key"
api_secret="file-secret"
dry_run=false
max_leverage=10
`, 0o600)

	t.Setenv("MEXC_API_KEY", "env-key")
	t.Setenv("MEXC_API_SECRET", "env-secret")
	t.Setenv("MEXCTRADE_DRY_RUN", "true")
	t.Setenv("MEXCTRADE_MAX_LEVERAGE", "50")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.APIKey != "env-key" || cfg.APISecret != "env-secret" {
		t.Errorf("env did not override file: %+v", cfg)
	}
	if cfg.DryRun != true || cfg.MaxLeverage != 50 {
		t.Errorf("env scalar override failed: %+v", cfg)
	}
}

func TestRejectsBadPerms(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	writeFile(t, path, `api_key="k"
api_secret="s"`, 0o644)

	_, err := Load(path)
	if err == nil {
		t.Fatalf("expected perm error")
	}
}

func TestMissingFileIsOKIfEnvProvidesCreds(t *testing.T) {
	t.Setenv("MEXC_API_KEY", "envk")
	t.Setenv("MEXC_API_SECRET", "envs")
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.APIKey != "envk" {
		t.Errorf("env-only path failed: %+v", cfg)
	}
}

func TestMissingFileAndNoEnvFails(t *testing.T) {
	t.Setenv("MEXC_API_KEY", "")
	t.Setenv("MEXC_API_SECRET", "")
	_, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err == nil {
		t.Fatalf("expected error: no creds anywhere")
	}
}
