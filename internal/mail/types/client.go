package types

import (
	"time"

	"email-organizer/internal/models"
)

// Folder represents an email folder/mailbox.
type Folder struct {
	Name string `json:"name"`
}

// MailClient defines the interface for email protocol clients (IMAP, Exchange, etc.).
type MailClient interface {
	Login() error
	Logout() error
	ListFolders() ([]*Folder, error)
	FetchMessages(folder string, count int, since time.Time) ([]*models.Message, error)
	MoveMessage(folder string, uid string, targetFolder string) error
	MarkRead(folder string, uid string) error
	DeleteMessage(folder string, uid string) error
	ForwardMessage(msg *models.Message, targetEmail string) error
	ReplyMessage(msg *models.Message, replyText string) error
}