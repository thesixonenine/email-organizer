package exchange

import (
	"time"

	"email-organizer/internal/mail/smtp"
	"email-organizer/internal/mail/types"
	"email-organizer/internal/models"
)

// GraphClient implements mail.MailClient via Microsoft Graph REST API.
// This is a stub — full implementation requires OAuth2 flow.
type GraphClient struct {
	tenantID   string
	clientID   string
	email      string
	secret     string
	smtpClient *smtp.Client
}

func NewGraphClient(tenantID, clientID, email, secret string, smtpClient *smtp.Client) (*GraphClient, error) {
	return &GraphClient{tenantID: tenantID, clientID: clientID, email: email, secret: secret, smtpClient: smtpClient}, nil
}

func (g *GraphClient) Login() error                                 { return nil }
func (g *GraphClient) Logout() error                                { return nil }
func (g *GraphClient) ListFolders() ([]*types.Folder, error)        { return nil, nil }
func (g *GraphClient) FetchMessages(string, int, time.Time) ([]*models.Message, error) { return nil, nil }
func (g *GraphClient) MoveMessage(string, string, string) error     { return nil }
func (g *GraphClient) MarkRead(string, string) error                { return nil }
func (g *GraphClient) DeleteMessage(string, string) error           { return nil }

func (g *GraphClient) ForwardMessage(msg *models.Message, targetEmail string) error {
	if g.smtpClient == nil {
		return nil
	}
	return g.smtpClient.SendForward(msg, targetEmail)
}

func (g *GraphClient) ReplyMessage(msg *models.Message, replyText string) error {
	if g.smtpClient == nil {
		return nil
	}
	return g.smtpClient.SendReply(msg, replyText)
}