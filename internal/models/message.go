package models

import "time"

type Message struct {
	ID          int64     `json:"id"`
	MailboxID   int64     `json:"mailbox_id"`
	UID         string    `json:"uid"`
	Folder      string    `json:"folder"`
	Subject     string    `json:"subject,omitempty"`
	BodyPreview string    `json:"body_preview,omitempty"`
	Body        string    `json:"body,omitempty"`
	FromName    string    `json:"from_name,omitempty"`
	FromEmail   string    `json:"from_email,omitempty"`
	ToList      []string  `json:"to_list,omitempty"`
	CcList      []string  `json:"cc_list,omitempty"`
	ReceivedAt  time.Time `json:"received_at"`
	Flags       []string  `json:"flags,omitempty"`
	IsDeleted   bool      `json:"is_deleted"`
	CreatedAt   time.Time `json:"created_at"`
}