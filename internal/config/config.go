package config

type Config struct {
	API       APIConfig       `mapstructure:"api"`
	Mailboxes []MailboxConfig `mapstructure:"mailboxes"`
	Rules     []RuleConfig    `mapstructure:"rules"`
}

type APIConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type MailboxConfig struct {
	Name         string `mapstructure:"name"`
	Protocol     string `mapstructure:"protocol"` // "imap" | "exchange"
	IMAPServer   string `mapstructure:"imap_server"`
	IMAPPort     int    `mapstructure:"imap_port"`
	ExchangeType string `mapstructure:"exchange_type"` // "ews" | "graph"
	EWSEndpoint  string `mapstructure:"ews_endpoint"`
	TenantID     string `mapstructure:"tenant_id"`
	ClientID     string `mapstructure:"client_id"`
	Email        string `mapstructure:"email"`
	AuthCode     string `mapstructure:"auth_code"`
	FetchCount   int    `mapstructure:"fetch_count"`
}

type RuleConfig struct {
	Enabled    bool            `mapstructure:"enabled"`
	Name       string          `mapstructure:"name"`
	Trigger    string          `mapstructure:"trigger"`
	Conditions ConditionConfig `mapstructure:"conditions"`
	Action     ActionConfig    `mapstructure:"action"`
}

type ConditionConfig struct {
	SubjectContains *string `mapstructure:"subject_contains"`
	BodyContains    *string `mapstructure:"body_contains"`
	SenderEmail     *string `mapstructure:"sender_email"`
	SenderDomain    *string `mapstructure:"sender_domain"`
	TimeRangeStart  *string `mapstructure:"time_range_start"`
	TimeRangeEnd    *string `mapstructure:"time_range_end"`
}

type ActionConfig struct {
	Type      string `mapstructure:"type"`
	Target    string `mapstructure:"target"`
	ReplyText string `mapstructure:"reply_text"`
}