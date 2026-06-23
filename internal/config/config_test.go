package config

import (
	"os"
	"testing"
)

func TestLoadConfig_Minimal(t *testing.T) {
	yaml := `
api:
  host: "127.0.0.1"
  port: 8080
mailboxes:
  - name: "test"
    protocol: "imap"
    email: "test@example.com"
    auth_code: "abc123"
    imap_server: "imap.example.com"
    imap_port: 993
rules:
  - name: "test-rule"
    enabled: true
    trigger: "on_arrival"
    conditions:
      subject_contains: "[Spam]"
    action:
      type: "move"
      target: "Junk"
`
	tmpFile, _ := os.CreateTemp("", "config-*.yaml")
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(yaml)
	tmpFile.Close()

	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.API.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.API.Port)
	}
	if len(cfg.Mailboxes) != 1 || cfg.Mailboxes[0].Email != "test@example.com" {
		t.Errorf("wrong mailbox config")
	}
	if len(cfg.Rules) != 1 || cfg.Rules[0].Action.Type != "move" {
		t.Errorf("wrong rule config")
	}
}

func TestLoadConfig_Exchange(t *testing.T) {
	yaml := `
mailboxes:
  - name: "exchange-test"
    protocol: "exchange"
    exchange_type: "ews"
    ews_endpoint: "https://mail.company.com/EWS/Exchange.asmx"
    email: "user@company.com"
    auth_code: "pass123"
    fetch_count: 200
rules: []
`
	tmpFile, _ := os.CreateTemp("", "config-*.yaml")
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(yaml)
	tmpFile.Close()

	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Mailboxes[0].Protocol != "exchange" || cfg.Mailboxes[0].EWSEndpoint == "" {
		t.Errorf("wrong exchange config")
	}
}

func TestValidate_FailsOnMissing(t *testing.T) {
	yaml := "mailboxes: []\nrules: []\n"
	tmpFile, _ := os.CreateTemp("", "config-*.yaml")
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(yaml)
	tmpFile.Close()

	_, err := LoadConfig(tmpFile.Name())
	if err == nil {
		t.Fatal("expected error for empty mailboxes")
	}
}