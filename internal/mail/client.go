// Package mail provides the interface and factory for multi-protocol email clients.
// The MailClient interface and Folder type are defined in the sub-package mail/types.
package mail

import (
	"email-organizer/internal/mail/types"
)

// MailClient is the interface for email protocol clients.
// Convenience re-export from mail/types.
type MailClient = types.MailClient

// Folder represents an email folder.
type Folder = types.Folder