package exchange

import (
	"time"

	"email-organizer/internal/mail/types"
	"email-organizer/internal/models"
)

// GraphClient implements mail.MailClient via Microsoft Graph REST API.
// This is a stub — full implementation requires OAuth2 flow.
type GraphClient struct {
	tenantID string
	clientID string
	email    string
	secret   string
}

func NewGraphClient(tenantID, clientID, email, secret string) (*GraphClient, error) {
	return &GraphClient{tenantID: tenantID, clientID: clientID, email: email, secret: secret}, nil
}

func (g *GraphClient) Login() error                                 { return nil }
func (g *GraphClient) Logout() error                                { return nil }
func (g *GraphClient) ListFolders() ([]*types.Folder, error)        { return nil, nil }
func (g *GraphClient) FetchMessages(string, int, time.Time) ([]*models.Message, error) { return nil, nil }
func (g *GraphClient) MoveMessage(string, string, string) error     { return nil }
func (g *GraphClient) MarkRead(string, string) error                { return nil }
func (g *GraphClient) DeleteMessage(string, string) error           { return nil }
func (g *GraphClient) ForwardMessage(*models.Message, string) error { return nil }
func (g *GraphClient) ReplyMessage(*models.Message, string) error   { return nil }