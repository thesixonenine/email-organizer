package smtp

import (
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"email-organizer/internal/models"
)

// Client sends emails via SMTP for forward and reply actions.
type Client struct {
	server   string
	port     int
	email    string
	authCode string
}

// NewClient creates a new SMTP client.
func NewClient(server string, port int, email, authCode string) *Client {
	return &Client{
		server:   server,
		port:     port,
		email:    email,
		authCode: authCode,
	}
}

func (c *Client) send(from, to, subject, body string) error {
	auth := smtp.PlainAuth("", c.email, c.authCode, c.server)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n\r\n%s",
		from, to, subject, body)
	addr := fmt.Sprintf("%s:%d", c.server, c.port)
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}

// SendForward forwards a message to targetEmail via SMTP.
// The subject is prefixed with "Fwd:" and the body quotes the original.
func (c *Client) SendForward(original *models.Message, targetEmail string) error {
	subject := "Fwd: " + original.Subject
	body := buildForwardBody(original)
	return c.send(c.email, targetEmail, subject, body)
}

// SendReply replies to the original message's sender via SMTP.
// The subject is prefixed with "Re:" and the body includes replyText with quoted original.
func (c *Client) SendReply(original *models.Message, replyText string) error {
	subject := "Re: " + original.Subject
	body := buildReplyBody(replyText, original)
	return c.send(c.email, original.FromEmail, subject, body)
}

func buildForwardBody(original *models.Message) string {
	var b strings.Builder
	b.WriteString("Forwarded message:\r\n")
	b.WriteString(fmt.Sprintf("> From: %s <%s>\r\n", original.FromName, original.FromEmail))
	b.WriteString(fmt.Sprintf("> Date: %s\r\n", original.ReceivedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("> Subject: %s\r\n", original.Subject))
	b.WriteString(">\r\n")
	for _, line := range strings.Split(original.Body, "\n") {
		b.WriteString("> ")
		b.WriteString(line)
		b.WriteString("\r\n")
	}
	return b.String()
}

func buildReplyBody(replyText string, original *models.Message) string {
	var b strings.Builder
	b.WriteString(replyText)
	b.WriteString("\r\n\r\n")
	b.WriteString(fmt.Sprintf("On %s, %s wrote:\r\n", original.ReceivedAt.Format(time.RFC3339), original.FromEmail))
	for _, line := range strings.Split(original.Body, "\n") {
		b.WriteString("> ")
		b.WriteString(line)
		b.WriteString("\r\n")
	}
	return b.String()
}