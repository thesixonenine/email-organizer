package mail

import (
	"fmt"

	"email-organizer/internal/config"
	"email-organizer/internal/mail/exchange"
	"email-organizer/internal/mail/imap"
)

func NewMailClient(cfg config.MailboxConfig) (MailClient, error) {
	switch cfg.Protocol {
	case "imap":
		return imap.NewClient(cfg.IMAPServer, cfg.IMAPPort, cfg.Email, cfg.AuthCode), nil
	case "exchange":
		switch cfg.ExchangeType {
		case "ews":
			return exchange.NewClient(cfg.EWSEndpoint, cfg.Email, cfg.AuthCode)
		case "graph":
			return exchange.NewGraphClient(cfg.TenantID, cfg.ClientID, cfg.Email, cfg.AuthCode)
		default:
			return nil, fmt.Errorf("unsupported exchange type: %s", cfg.ExchangeType)
		}
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", cfg.Protocol)
	}
}