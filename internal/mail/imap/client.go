package imap

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"email-organizer/internal/mail/types"
	"email-organizer/internal/models"
)

type Client struct {
	server   string
	port     int
	email    string
	authCode string
	client   *imapclient.Client
}

func NewClient(server string, port int, email, authCode string) *Client {
	return &Client{server: server, port: port, email: email, authCode: authCode}
}

func (c *Client) Login() error {
	raw, err := imapclient.DialTLS(fmt.Sprintf("%s:%d", c.server, c.port), &imapclient.Options{
		TLSConfig: &tls.Config{InsecureSkipVerify: false},
	})
	if err != nil {
		return fmt.Errorf("dial IMAP: %w", err)
	}
	c.client = raw
	if err := c.client.Login(c.email, c.authCode).Wait(); err != nil {
		return fmt.Errorf("IMAP login: %w", err)
	}
	return nil
}

func (c *Client) Logout() error {
	if c.client != nil {
		return c.client.Logout().Wait()
	}
	return nil
}

func (c *Client) ListFolders() ([]*types.Folder, error) {
	items, err := c.client.List("", "*", nil).Collect()
	if err != nil {
		return nil, fmt.Errorf("list folders: %w", err)
	}
	folders := make([]*types.Folder, 0, len(items))
	for _, item := range items {
		folders = append(folders, &types.Folder{Name: item.Mailbox})
	}
	return folders, nil
}

func (c *Client) FetchMessages(folder string, count int, since time.Time) ([]*models.Message, error) {
	_, err := c.client.Select(folder, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("select folder %s: %w", folder, err)
	}

	var searchData *imap.SearchData
	if !since.IsZero() {
		searchData, err = c.client.UIDSearch(&imap.SearchCriteria{Since: since}, nil).Wait()
	} else {
		searchData, err = c.client.UIDSearch(nil, nil).Wait()
	}
	if err != nil {
		return nil, fmt.Errorf("search messages: %w", err)
	}

	uidSet, ok := searchData.All.(imap.UIDSet)
	if !ok || len(uidSet) == 0 {
		return nil, nil
	}
	uids, _ := uidSet.Nums()
	if len(uids) > count {
		uids = uids[len(uids)-count:]
	}
	if len(uids) == 0 {
		return nil, nil
	}

	messages, err := c.client.Fetch(imap.UIDSetNum(uids...), &imap.FetchOptions{
		Flags:        true,
		Envelope:     true,
		InternalDate: true,
		BodySection:  []*imap.FetchItemBodySection{{}},
	}).Collect()
	if err != nil {
		return nil, fmt.Errorf("fetch messages: %w", err)
	}

	result := make([]*models.Message, 0, len(messages))
	for _, msg := range messages {
		m := &models.Message{
			UID:    fmt.Sprintf("%d", msg.UID),
			Folder: folder,
		}
		if msg.Envelope != nil {
			m.Subject = msg.Envelope.Subject
			if len(msg.Envelope.From) > 0 {
				m.FromName = msg.Envelope.From[0].Name
				m.FromEmail = msg.Envelope.From[0].Mailbox + "@" + msg.Envelope.From[0].Host
			}
		}
		if !msg.InternalDate.IsZero() {
			m.ReceivedAt = msg.InternalDate
		}
		for _, f := range msg.Flags {
			m.Flags = append(m.Flags, string(f))
		}
		for _, bs := range msg.BodySection {
			if len(bs.Bytes) > 0 {
				m.Body = string(bs.Bytes)
				if len(m.Body) > 500 {
					m.BodyPreview = m.Body[:500]
				} else {
					m.BodyPreview = m.Body
				}
				break
			}
		}
		result = append(result, m)
	}
	return result, nil
}

func (c *Client) MoveMessage(folder, uid, targetFolder string) error {
	var uidNum imap.UID
	if _, err := fmt.Sscanf(uid, "%d", &uidNum); err != nil {
		return fmt.Errorf("parse uid: %w", err)
	}
	_, err := c.client.Move(imap.UIDSetNum(uidNum), targetFolder).Wait()
	return err
}

func (c *Client) MarkRead(folder, uid string) error {
	var uidNum imap.UID
	if _, err := fmt.Sscanf(uid, "%d", &uidNum); err != nil {
		return fmt.Errorf("parse uid: %w", err)
	}
	_, err := c.client.Store(imap.UIDSetNum(uidNum), &imap.StoreFlags{
		Op:    imap.StoreFlagsSet,
		Flags: []imap.Flag{imap.FlagSeen},
	}, nil).Collect()
	return err
}

func (c *Client) DeleteMessage(folder, uid string) error {
	var uidNum imap.UID
	if _, err := fmt.Sscanf(uid, "%d", &uidNum); err != nil {
		return fmt.Errorf("parse uid: %w", err)
	}
	_, err := c.client.Store(imap.UIDSetNum(uidNum), &imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}, nil).Collect()
	if err != nil {
		return err
	}
	return c.client.Expunge().Close()
}

func (c *Client) ForwardMessage(msg *models.Message, targetEmail string) error {
	return fmt.Errorf("IMAP forward not implemented — requires SMTP client")
}

func (c *Client) ReplyMessage(msg *models.Message, replyText string) error {
	return fmt.Errorf("IMAP reply not implemented — requires SMTP client")
}