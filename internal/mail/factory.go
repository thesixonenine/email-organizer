package mail

import (
	"fmt"
	"strings"

	"email-organizer/internal/config"
	"email-organizer/internal/mail/exchange"
	"email-organizer/internal/mail/imap"
	"email-organizer/internal/mail/smtp"
)

func NewMailClient(cfg config.MailboxConfig) (MailClient, error) {
	// Create SMTP client for forward/reply actions
	smtpServer, smtpPort := resolveSMTP(cfg)
	smptClient := smtp.NewClient(smtpServer, smtpPort, cfg.Email, cfg.AuthCode)

	switch cfg.Protocol {
	case "imap":
		return imap.NewClient(cfg.IMAPServer, cfg.IMAPPort, cfg.Email, cfg.AuthCode, smptClient), nil
	case "exchange":
		switch cfg.ExchangeType {
		case "ews":
			return exchange.NewClient(cfg.EWSEndpoint, cfg.Email, cfg.AuthCode, smptClient)
		case "graph":
			return exchange.NewGraphClient(cfg.TenantID, cfg.ClientID, cfg.Email, cfg.AuthCode, smptClient)
		default:
			return nil, fmt.Errorf("unsupported exchange type: %s", cfg.ExchangeType)
		}
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", cfg.Protocol)
	}
}

// resolveSMTP derives the SMTP server address and port from config.
// If smtp_server is explicitly set, use it. Otherwise derive from IMAP server
// or use a default for Exchange.
func resolveSMTP(cfg config.MailboxConfig) (string, int) {
	if cfg.SMTPServer != "" {
		port := cfg.SMTPPort
		if port <= 0 {
			port = 587
		}
		return cfg.SMTPServer, port
	}
	// Derive from IMAP: imap.xxx.com -> smtp.xxx.com
	if cfg.Protocol == "imap" && cfg.IMAPServer != "" {
		return strings.Replace(cfg.IMAPServer, "imap.", "smtp.", 1), 587
	}
	// Default for Exchange
	return "smtp.office365.com", 587
}