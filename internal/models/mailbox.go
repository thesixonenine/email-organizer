package models

import "time"

type Mailbox struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Protocol     string     `json:"protocol"`
	Email        string     `json:"email"`
	IMAPServer   string     `json:"imap_server,omitempty"`
	IMAPPort     int        `json:"imap_port,omitempty"`
	ExchangeType string     `json:"exchange_type,omitempty"`
	EWSEndpoint  string     `json:"ews_endpoint,omitempty"`
	TenantID     string     `json:"tenant_id,omitempty"`
	ClientID     string     `json:"client_id,omitempty"`
	FetchCount   int        `json:"fetch_count"`
	LastSyncAt   *time.Time `json:"last_sync_at,omitempty"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}