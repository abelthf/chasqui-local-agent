package main

import (
	"os"
	"testing"
	"time"
)

func TestBackoffCapsAtSixtyFourSeconds(t *testing.T) {
	if got := backoff(0); got != time.Second {
		t.Fatalf("backoff(0)=%s", got)
	}
	if got := backoff(7); got != 64*time.Second {
		t.Fatalf("backoff(7)=%s", got)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("CHASQUI_BASE_URL", "")
	t.Setenv("CHASQUI_AGENT_TOKEN", "token")
	t.Setenv("CHASQUI_LOCAL_CALLBACK_URL", "")
	cfg := loadConfig()
	if cfg.BaseURL != "https://chasqui.inkalab.org.pe/api" {
		t.Fatalf("unexpected base url: %s", cfg.BaseURL)
	}
	if cfg.CallbackURL != "http://localhost:5051/inbound" {
		t.Fatalf("unexpected callback url: %s", cfg.CallbackURL)
	}
	if cfg.AgentToken != "token" {
		t.Fatalf("agent token not loaded")
	}
}

func TestOpenDBCreatesStore(t *testing.T) {
	path := t.TempDir() + "/events.db"
	db, err := openDB(path)
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("db file missing: %v", err)
	}
}
