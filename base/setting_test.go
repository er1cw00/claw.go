package base

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetProviderConfig(t *testing.T) {
	path := filepath.Join("..", "main", "goclaw.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("config not found: %v", err)
	}

	if err := parseSettings(path); err != nil {
		t.Fatalf("parseSettings failed: %v", err)
	}

	cfg := GetProviderConfig("deepseek")
	if cfg.APIBase != "https://api.deepseek.com" {
		t.Errorf("unexpected deepseek api base: %s", cfg.APIBase)
	}
	if cfg.APIKey != "sk-xxxx" {
		t.Errorf("unexpected deepseek api key: %s", cfg.APIKey)
	}

	cfg = GetProviderConfig("openrouter")
	if cfg.APIKey != "" {
		t.Errorf("unexpected openrouter api key: %s", cfg.APIKey)
	}

	cfg = GetProviderConfig("UNKNOWN")
	if cfg != (ProviderConfig{}) {
		t.Errorf("expected empty config for unknown provider, got %+v", cfg)
	}
}
