# SMTP 转发/回复实现设计

- **日期**: 2026-07-02
- **状态**: 设计稿
- **关联任务**: T-021

## 背景

当前 IMAP 和 Exchange EWS 客户端的 `ForwardMessage` / `ReplyMessage` 方法均为 stub（返回 "not implemented"），导致规则引擎的 `forward` 和 `reply` 动作无法执行。需要新增 SMTP 邮件发送能力，使上述动作可真实工作。

## 设计概览

在 `internal/mail/smtp` 新建 SMTP 客户端包，使用标准库 `net/smtp` 实现邮件发送。`MailboxConfig` 增加 SMTP 配置字段（可自动推导），工厂在创建 IMAP/Exchange 客户端时同时创建 SMTP 客户端并注入。

## 详细设计

### 1. SMTP 配置（config/config.go）

`MailboxConfig` 新增字段：

```go
type MailboxConfig struct {
    // ...已有字段...
    SMTPServer string `mapstructure:"smtp_server"`
    SMTPPort   int    `mapstructure:"smtp_port"`
}
```

字段说明：
- `smtp_server`: SMTP 服务器地址（可选，可自动推导）
- `smtp_port`: SMTP 端口（可选，默认 587）
- 认证复用 `auth_code` 字段

### 2. SMTP 服务器自动推导（mail/factory.go）

| 场景 | 推导规则 |
|------|---------|
| IMAP 协议 + 未填写 SMTP | `imap.xxx.com` → `smtp.xxx.com`，端口 587 |
| Exchange 协议 + 未填写 SMTP | `smtp.office365.com:587` |
| 用户显式填写 SMTP | 使用用户值 |

### 3. SMTP 客户端包（internal/mail/smtp/client.go）

**类型**：

```go
type Client struct {
    server   string
    port     int
    email    string
    authCode string
}
```

**方法**：

| 方法 | 功能 | 主题格式 |
|------|------|---------|
| `SendForward(original, targetEmail)` | 转发到目标邮箱 | `"Fwd: " + 原始主题` |
| `SendReply(original, replyText)` | 回复发件人 | `"Re: " + 原始主题` |

**邮件格式**：

- 使用 `net/smtp.PlainAuth` + `smtp.SendMail`
- MIME 头部：From, To, Subject, MIME-Version, Content-Type
- 纯文本，UTF-8 编码

转发正文格式：
```
Forwarded message:
> From: 原始发件人 <原始邮箱>
> Date: 原始时间
> Subject: 原始主题
>
> 原始正文
```

回复正文格式：
```
回复内容

On 原始时间, 原始发件人 wrote:
> 原始正文
```

### 4. IMAP 客户端集成（internal/mail/imap/client.go）

- `Client` 结构体新增 `smtpClient *smtp.Client` 字段
- 构造函数 `NewClient` 新增 `smtpClient` 参数
- `ForwardMessage`: 检查 smtpClient 非 nil，调用 `SendForward`
- `ReplyMessage`: 检查 smtpClient 非 nil，调用 `SendReply`

### 5. Exchange 客户端集成（internal/mail/exchange/client.go, graph.go）

- 同样方式：构造函数新增 `smtpClient` 参数
- Forward/Reply 调用 SMTP 客户端
- （EWS 原生 ForwardItem/ReplyToItem 留待后续迭代）

### 6. 工厂集成（mail/factory.go）

- `NewMailClient` 中先调用 `resolveSMTP(cfg)` 获取 SMTP 地址
- 创建 `smtp.Client` 实例
- 传递给 IMAP / Exchange 客户端构造函数

### 7. 配置验证（config/loader.go）

- 不强制 SMTP 配置（可能不使用 forward/reply 动作）
- 若规则使用了 forward/reply 但 SMTP 不可用，执行时返回 error，由引擎记录到 rule_logs

## 改动文件清单

| 文件 | 改动类型 |
|------|---------|
| `internal/mail/smtp/client.go` | **新建** |
| `internal/config/config.go` | 修改 — 新增 SMTPServer, SMTPPort 字段 |
| `internal/mail/factory.go` | 修改 — 新增 resolveSMTP，SMTP 客户端创建与注入 |
| `internal/mail/imap/client.go` | 修改 — 构造函数参数 + Forward/Reply 实现 |
| `internal/mail/exchange/client.go` | 修改 — 构造函数参数 + Forward/Reply 实现 |
| `internal/mail/exchange/graph.go` | 修改 — 构造函数参数 |
| `configs/config.example.yaml` | 修改 — SMTP 配置示例 |

## 不涉及

- `cmd/organizer/main.go` — 工厂封装了 SMTP 创建，main 无需变更
- `internal/engine/` — 引擎调用 Forward/Reply 接口，底层实现透明
- `internal/api/` — 无 API 改动

## 测试策略（后续 M4 阶段）

- SMTP 包使用 mock SMTP 服务器测试发送
- 邮件构造格式（Fwd:/Re: 前缀、正文引用）单独测试
- 工厂的 SMTP 服务器推导逻辑单元测试