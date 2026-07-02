package exchange

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"email-organizer/internal/mail/smtp"
	"email-organizer/internal/mail/types"
	"email-organizer/internal/models"
)

type Client struct {
	endpoint   string
	email      string
	password   string
	httpClient *http.Client
	smtpClient *smtp.Client
}

func NewClient(endpoint, email, password string, smtpClient *smtp.Client) (*Client, error) {
	return &Client{
		endpoint: endpoint,
		email:    email,
		password: password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			},
		},
		smtpClient: smtpClient,
	}, nil
}

func (c *Client) sendSOAP(bodyXML string) ([]byte, error) {
	soapReq := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
  xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages"
  xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types">
  <soap:Header>
    <t:RequestServerVersion Version="Exchange2013"/>
  </soap:Header>
  <soap:Body>%s</soap:Body>
</soap:Envelope>`, bodyXML)

	req, err := http.NewRequest("POST", c.endpoint, bytes.NewBufferString(soapReq))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.SetBasicAuth(c.email, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ews request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return body, nil
}

func (c *Client) Login() error {
	_, err := c.ListFolders()
	return err
}

func (c *Client) Logout() error { return nil }

func (c *Client) ListFolders() ([]*types.Folder, error) {
	soapBody := `<m:FindFolder Traversal="Shallow">
    <m:FolderShape><t:BaseShape>AllProperties</t:BaseShape></m:FolderShape>
    <m:ParentFolderIds>
      <t:DistinguishedFolderId Id="msgfolderroot"/>
    </m:ParentFolderIds>
  </m:FindFolder>`
	data, err := c.sendSOAP(soapBody)
	if err != nil {
		return nil, err
	}
	return parseFolders(data)
}

func parseFolders(data []byte) ([]*types.Folder, error) {
	var env struct {
		Body struct {
			FindFolderResponse struct {
				ResponseMessages struct {
					FindFolderResponseMessage struct {
						RootFolder struct {
							Folders struct {
								Folder []struct {
									DisplayName string `xml:"t:DisplayName"`
								} `xml:"t:Folder"`
							} `xml:"t:Folders"`
						} `xml:"m:RootFolder"`
					} `xml:"m:FindFolderResponseMessage"`
				} `xml:"m:ResponseMessages"`
			} `xml:"m:FindFolderResponse"`
		} `xml:"soap:Body"`
	}
	if err := xml.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("parse folders: %w", err)
	}
	folders := make([]*types.Folder, 0)
	root := env.Body.FindFolderResponse.ResponseMessages.FindFolderResponseMessage.RootFolder
	for _, f := range root.Folders.Folder {
		folders = append(folders, &types.Folder{Name: f.DisplayName})
	}
	return folders, nil
}

func (c *Client) FetchMessages(folder string, count int, since time.Time) ([]*models.Message, error) {
	soapBody := fmt.Sprintf(`<m:FindItem Traversal="Shallow">
    <m:ItemShape><t:BaseShape>IdOnly</t:BaseShape></m:ItemShape>
    <m:ParentFolderIds>
      <t:DistinguishedFolderId Id="%s"/>
    </m:ParentFolderIds>
    <m:Restriction>
      <t:IsGreaterThan>
        <t:FieldURI FieldURI="item:DateTimeReceived"/>
        <t:FieldURIOrConstant>
          <t:Constant Value="%s"/>
        </t:FieldURIOrConstant>
      </t:IsGreaterThan>
    </m:Restriction>
  </m:FindItem>`, folder, since.Format(time.RFC3339))
	data, err := c.sendSOAP(soapBody)
	if err != nil {
		return nil, err
	}
	return parseFindItemResponse(data)
}

func parseFindItemResponse(data []byte) ([]*models.Message, error) {
	var env struct {
		Body struct {
			FindItemResponse struct {
				ResponseMessages struct {
					FindItemResponseMessage struct {
						RootFolder struct {
							Items struct {
								Message []struct {
									ItemID struct {
										ID string `xml:"Id,attr"`
									} `xml:"t:ItemId"`
									Subject string `xml:"t:Subject"`
								} `xml:"t:Message"`
							} `xml:"t:Items"`
						} `xml:"m:RootFolder"`
					} `xml:"m:FindItemResponseMessage"`
				} `xml:"m:ResponseMessages"`
			} `xml:"m:FindItemResponse"`
		} `xml:"soap:Body"`
	}
	if err := xml.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("parse FindItem response: %w", err)
	}

	items := env.Body.FindItemResponse.ResponseMessages.FindItemResponseMessage.RootFolder.Items.Message
	result := make([]*models.Message, 0, len(items))
	for _, item := range items {
		result = append(result, &models.Message{
			UID:     item.ItemID.ID,
			Subject: item.Subject,
		})
	}
	return result, nil
}

func (c *Client) MoveMessage(folder, uid, targetFolder string) error {
	soapBody := fmt.Sprintf(`<m:MoveItem>
    <m:ToFolderId>
      <t:DistinguishedFolderId Id="%s"/>
    </m:ToFolderId>
    <m:ItemIds>
      <t:ItemId Id="%s"/>
    </m:ItemIds>
  </m:MoveItem>`, targetFolder, uid)
	_, err := c.sendSOAP(soapBody)
	return err
}

func (c *Client) MarkRead(folder, uid string) error {
	soapBody := fmt.Sprintf(`<m:UpdateItem MessageDisposition="SaveOnly" ConflictResolution="AlwaysOverwrite">
    <m:ItemChanges>
      <t:ItemChange>
        <t:ItemId Id="%s"/>
        <t:Updates>
          <t:SetItemField>
            <t:FieldURI FieldURI="message:IsRead"/>
            <t:Message><t:IsRead>true</t:IsRead></t:Message>
          </t:SetItemField>
        </t:Updates>
      </t:ItemChange>
    </m:ItemChanges>
  </m:UpdateItem>`, uid)
	_, err := c.sendSOAP(soapBody)
	return err
}

func (c *Client) DeleteMessage(folder, uid string) error {
	soapBody := fmt.Sprintf(`<m:DeleteItem DeleteType="HardDelete" SendMeetingCancellations="SendToNone">
    <m:ItemIds><t:ItemId Id="%s"/></m:ItemIds>
  </m:DeleteItem>`, uid)
	_, err := c.sendSOAP(soapBody)
	return err
}

func (c *Client) ForwardMessage(msg *models.Message, targetEmail string) error {
	if c.smtpClient == nil {
		return fmt.Errorf("SMTP not configured — cannot forward")
	}
	return c.smtpClient.SendForward(msg, targetEmail)
}

func (c *Client) ReplyMessage(msg *models.Message, replyText string) error {
	if c.smtpClient == nil {
		return fmt.Errorf("SMTP not configured — cannot reply")
	}
	return c.smtpClient.SendReply(msg, replyText)
}