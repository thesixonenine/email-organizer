# SMTP 转发/回复实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 IMAP 和 Exchange EWS/Graph 客户端的 ForwardMessage/ReplyMessage 方法，使其不再返回 "not implemented"，而是通过 SMTP 真实发送邮件。

**Architecture:** 新建 `internal/mail/smtp` 包封装标准库 `net/smtp` 的邮件发送逻辑；`MailboxConfig` 新增 SMTP 配置字段（支持自动推导）；`mail/factory.go` 在创建邮件客户端时同时创建 SMTP 客户端并注入到 IMAP/Exchange 客户端中。

**Tech Stack:** Go 标准库 `net/smtp`（邮件发送），`net/mail`（地址解析），`crypto/tls`（TLS 连接）

## Global Constraints

- 使用 Go 标准库 `net/smtp`，不引入第三方 SMTP 库
- SMTP 认证使用 `smtp.PlainAuth`（适配 QQ/163/Gmail 等主流邮箱的应用专用密码）
- 纯文本邮件格式，不加 HTML
- 配置文件中的 SMTP 字段为可选（可自动推导），不破坏现有配置兼容性
- 所有改动必须 `make build` 通过

---

### Task 1: SMTP 配置字段 + SMTP 客户端包

**Files:**
- Modify: `internal/config/config.go:25-28` — 新增 SMTPServer, SMTPPort 字段
- Create: `internal/mail/smtp/client.go` — SMTP 客户端实现
- (Tests added in M4, not now)

**Interfaces:**
- Consumes: models.Message
- Produces: `smtp.Client` with methods `SendForward`, `SendReply`

- [ ] **Step 1: 在 MailboxConfig 添加 SMTP 字段**

编辑 `internal/config/config.go`，在 `MailboxConfig` 结构体中添加两行：

```go
type MailboxConfig struct {
	Name         string `mapstructure:"name"`
	Protocol     string `mapstructure:"protocol"`
	IMAPServer   string `mapstructure:"imap_server"`
	IMAPPort     int    `mapstructure:"imap_port"`
	ExchangeType string `mapstructure:"exchange_type"`
	EWSEndpoint  string `mapstructure:"ews_endpoint"`
	TenantID     string `mapstructure:"tenant_id"`
	ClientID     string `mapstructure:"client_id"`
	Email        string `mapstructure:"email"`
	AuthCode     string `mapstructure:"auth_code"`
	FetchCount   int    `mapstructure:"fetch_count"`
	SMTPServer   string `mapstructure:"smtp_server"`   // 新增
	SMTPPort     int    `mapstructure:"smtp_port"`     // 新增
}
```

- [ ] **Step 2: 创建 SMTP 客户端包**

创建 `internal/mail/smtp/client.go`，先写包声明和 Client 结构体：

```go
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
```

- [ ] **Step 3: 实现底层 send 方法**

在 `internal/mail/smtp/client.go` 中添加 `send` 方法：

```go
func (c *Client) send(from, to, subject, body string) error {
	auth := smtp.PlainAuth("", c.email, c.authCode, c.server)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n\r\n%s",
		from, to, subject, body)
	addr := fmt.Sprintf("%s:%d", c.server, c.port)
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}
```

- [ ] **Step 4: 实现邮件正文构造函数**

在 `internal/mail/smtp/client.go` 中添加两个私有的正文构造函数：

```go
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
```

- [ ] **Step 5: 实现 SendForward 和 SendReply 公开方法**

在 `internal/mail/smtp/client.go` 中添加公开方法：

```go
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
```

- [ ] **Step 6: 验证编译**

```bash
cd /home/simple/github/email-organizer && go build ./internal/mail/smtp/
```

预期输出：无错误（新包编译通过）

- [ ] **Step 7: 提交**

```bash
git add internal/config/config.go internal/mail/smtp/client.go
git commit -m "feat(mail): add SMTP client package and config fields for forward/reply"
```

---

### Task 2: 工厂 SMTP 推导 + IMAP 客户端集成

**Files:**
- Modify: `internal/mail/factory.go` — 新增 `resolveSMTP`，SMTP 客户端创建，传递给 IMAP 构造函数
- Modify: `internal/mail/imap/client.go` — 构造函数新增 `smtpClient` 参数，实现 Forward/Reply

**Interfaces:**
- Consumes: `smtp.Client` (from Task 1), `config.MailboxConfig` (with SMTP fields)
- Produces: Updated `mail.NewMailClient` that creates and injects `smtp.Client`

- [ ] **Step 1: 给 factory.go 添加 smtp 导入和 resolveSMTP 函数**

编辑 `internal/mail/factory.go`，添加导入和 `resolveSMTP`：

```go
package mail

import (
	"fmt"
	"strings"

	"email-organizer/internal/config"
	"email-organizer/internal/mail/exchange"
	"email-organizer/internal/mail/imap"
	"email-organizer/internal/mail/smtp"  // 新增
)

// resolveSMTP derives the SMTP server address and port from config.
// If smtp_server is explicitly set, use it. Otherwise derive from IMAP server.
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
```

- [ ] **Step 2: 更新 factory.NewMailClient 创建 SMTP 客户端并传入 IMAP**

编辑 `internal/mail/factory.go`，修改 `NewMailClient`：

```go
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
```

- [ ] **Step 3: 更新 IMAP 客户端构造函数**

编辑 `internal/mail/imap/client.go`，修改导入、结构体和构造函数：

```go
package imap

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"email-organizer/internal/mail/smtp"      // 新增
	"email-organizer/internal/mail/types"
	"email-organizer/internal/models"
)

type Client struct {
	server     string
	port       int
	email      string
	authCode   string
	client     *imapclient.Client
	smtpClient *smtp.Client  // 新增
}

func NewClient(server string, port int, email, authCode string, smtpClient *smtp.Client) *Client {
	return &Client{
		server:     server,
		port:       port,
		email:      email,
		authCode:   authCode,
		smtpClient: smtpClient,
	}
}
```

- [ ] **Step 4: 实现 IMAP ForwardMessage 和 ReplyMessage**

在 `internal/mail/imap/client.go` 中替换原有的 stub 实现：

```go
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
```

- [ ] **Step 5: 验证编译**

```bash
cd /home/simple/github/email-organizer && go build ./internal/mail/...
```

预期输出：无错误

- [ ] **Step 6: 提交**

```bash
git add internal/mail/factory.go internal/mail/imap/client.go
git commit -m "feat(mail): integrate SMTP client with factory and IMAP forward/reply"
```

---

### Task 3: Exchange EWS 和 Graph 客户端集成

**Files:**
- Modify: `internal/mail/exchange/client.go` — 构造函数新增 `smtpClient` 参数，实现 Forward/Reply
- Modify: `internal/mail/exchange/graph.go` — 构造函数新增 `smtpClient` 参数，实现 Forward/Reply

- [ ] **Step 1: 更新 EWS 客户端**

编辑 `internal/mail/exchange/client.go`，修改结构体、构造函数，替换 Forward/Reply：

添加导入：
```go
import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"email-organizer/internal/mail/smtp"      // 新增
	"email-organizer/internal/mail/types"
	"email-organizer/internal/models"
)
```

修改结构体（新增 `smtpClient` 字段）：
```go
type Client struct {
	endpoint   string
	email      string
	password   string
	httpClient *http.Client
	smtpClient *smtp.Client  // 新增
}
```

修改构造函数（新增 `smtpClient` 参数）：
```go
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
```

替换原有的 ForwardMessage 和 ReplyMessage stub：
```go
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
```

- [ ] **Step 2: 更新 Graph 客户端**

编辑 `internal/mail/exchange/graph.go`：

添加导入：
```go
import (
	"time"

	"email-organizer/internal/mail/smtp"      // 新增
	"email-organizer/internal/mail/types"
	"email-organizer/internal/models"
)
```

修改结构体（新增 `smtpClient` 字段）：
```go
type GraphClient struct {
	tenantID   string
	clientID   string
	email      string
	secret     string
	smtpClient *smtp.Client
}
```

修改构造函数（新增 `smtpClient` 参数）：
```go
func NewGraphClient(tenantID, clientID, email, secret string, smtpClient *smtp.Client) (*GraphClient, error) {
	return &GraphClient{tenantID: tenantID, clientID: clientID, email: email, secret: secret, smtpClient: smtpClient}, nil
}
```

替换原有的 ForwardMessage 和 ReplyMessage：
```go
func (g *GraphClient) ForwardMessage(msg *models.Message, targetEmail string) error {
	if g.smtpClient == nil {
		return fmt.Errorf("SMTP not configured — cannot forward")
	}
	return g.smtpClient.SendForward(msg, targetEmail)
}

func (g *GraphClient) ReplyMessage(msg *models.Message, replyText string) error {
	if g.smtpClient == nil {
		return fmt.Errorf("SMTP not configured — cannot reply")
	}
	return g.smtpClient.SendReply(msg, replyText)
}
```

- [ ] **Step 3: 验证编译**

```bash
cd /home/simple/github/email-organizer && go build ./internal/mail/exchange/...
```

预期输出：无错误

- [ ] **Step 4: 提交**

```bash
git add internal/mail/exchange/client.go internal/mail/exchange/graph.go
git commit -m "feat(mail): integrate SMTP client with Exchange EWS and Graph forward/reply"
```

---

### Task 4: 配置示例 + 全量编译验证

**Files:**
- Modify: `configs/config.example.yaml` — 添加 SMTP 配置示例

- [ ] **Step 1: 更新配置示例**

编辑 `configs/config.example.yaml`，在三个邮箱示例中添加 SMTP 字段：

```yaml
mailboxes:
  - name: "个人QQ邮箱"
    protocol: "imap"
    imap_server: "imap.qq.com"
    imap_port: 993
    smtp_server: "smtp.qq.com"   # 新增
    smtp_port: 587               # 新增
    email: "your-qq-number@qq.com"
    auth_code: "your-authorization-code"
    fetch_count: 500

  - name: "公司Exchange邮箱"
    protocol: "exchange"
    exchange_type: "ews"
    ews_endpoint: "https://mail.company.com/EWS/Exchange.asmx"
    smtp_server: "smtp.company.com"   # 新增（可选）
    smtp_port: 587                     # 新增（可选）
    email: "your-name@company.com"
    auth_code: "your-password"
    fetch_count: 200

  - name: "Office 365邮箱"
    protocol: "exchange"
    exchange_type: "graph"
    tenant_id: "your-tenant-id"
    client_id: "your-client-id"
    smtp_server: "smtp.office365.com"   # 新增（可选，可自动推导）
    smtp_port: 587                       # 新增（可选）
    email: "user@contoso.onmicrosoft.com"
    auth_code: "your-oauth-secret"
```

- [ ] **Step 2: 全量编译**

```bash
cd /home/simple/github/email-organizer && go build ./...
```

预期输出：无错误

- [ ] **Step 3: 运行现有测试确保无回归**

```bash
cd /home/simple/github/email-organizer && make test
```

预期输出：所有现有测试通过

- [ ] **Step 4: 提交**

```bash
git add configs/config.example.yaml
git commit -m "docs: add SMTP config example for forward/reply"
```