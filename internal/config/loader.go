package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func LoadConfig(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetDefault("api.host", "127.0.0.1")
	v.SetDefault("api.port", 8080)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	for i := range cfg.Mailboxes {
		if cfg.Mailboxes[i].FetchCount <= 0 {
			cfg.Mailboxes[i].FetchCount = 500
		}
		if cfg.Mailboxes[i].IMAPPort <= 0 {
			cfg.Mailboxes[i].IMAPPort = 993
		}
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if len(c.Mailboxes) == 0 {
		return fmt.Errorf("at least one mailbox required")
	}
	for _, mb := range c.Mailboxes {
		if err := mb.Validate(); err != nil {
			return fmt.Errorf("mailbox %q: %w", mb.Name, err)
		}
	}
	return nil
}

func (m *MailboxConfig) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("name required")
	}
	if m.Email == "" {
		return fmt.Errorf("email required")
	}
	if m.AuthCode == "" {
		return fmt.Errorf("auth_code required")
	}
	switch m.Protocol {
	case "imap":
		if m.IMAPServer == "" {
			return fmt.Errorf("imap_server required for imap")
		}
	case "exchange":
		if m.ExchangeType == "" {
			return fmt.Errorf("exchange_type required")
		}
		switch m.ExchangeType {
		case "ews":
			if m.EWSEndpoint == "" {
				return fmt.Errorf("ews_endpoint required")
			}
		case "graph":
			if m.TenantID == "" || m.ClientID == "" {
				return fmt.Errorf("tenant_id/client_id required for graph")
			}
		default:
			return fmt.Errorf("unsupported exchange_type: %s", m.ExchangeType)
		}
	case "":
		return fmt.Errorf("protocol required (imap or exchange)")
	default:
		return fmt.Errorf("unsupported protocol: %s", m.Protocol)
	}
	return nil
}

func (r *RuleConfig) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("rule name required")
	}
	switch r.Trigger {
	case "on_arrival", "on_delete":
	default:
		return fmt.Errorf("trigger must be on_arrival or on_delete, got %s", r.Trigger)
	}
	switch r.Action.Type {
	case "move", "mark_read", "forward", "reply":
	default:
		return fmt.Errorf("invalid action type: %s", r.Action.Type)
	}
	if r.Action.Type == "move" && strings.TrimSpace(r.Action.Target) == "" {
		return fmt.Errorf("target folder required for move action")
	}
	if r.Action.Type == "forward" && strings.TrimSpace(r.Action.Target) == "" {
		return fmt.Errorf("target email required for forward action")
	}
	return nil
}