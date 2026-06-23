# Architecture.md

## 项目概述

多邮箱整理助手是一个 Go 语言编写的后端服务，支持多邮箱 IMAP / Exchange 双协议登录、邮件定时拉取、基于规则的自动整理（移动/删除/标记/转发），并提供 RESTful API 供前端管理。

---

## 目录结构

```
email-organizer/
├── cmd/                           # 应用入口
│   └── organizer/
│       └── main.go                # 主函数：初始化所有模块并启动
│
├── internal/                      # 内部包（不对外导出）
│   ├── api/                       # RESTful API 层
│   │   ├── router.go              # chi 路由注册
│   │   ├── handler_mailbox.go     # 邮箱/文件夹/邮件查看处理器
│   │   ├── handler_rule.go        # 规则管理处理器
│   │   ├── response.go            # 统一 JSON 响应结构
│   │   └── middleware.go          # 日志、恢复、CORS 中间件
│   │
│   ├── config/                    # 配置管理
│   │   ├── config.go              # 配置结构体定义 (Config, MailboxConfig, RuleConfig)
│   │   └── loader.go              # Viper 加载配置，校验和默认值
│   │
│   ├── engine/                    # 规则引擎（核心业务逻辑）
│   │   ├── engine.go              # 引擎主循环：加载规则 → 匹配条件 → 执行动作
│   │   ├── rule.go                # 规则结构体定义与序列化
│   │   ├── matcher.go             # 条件匹配器（主题/正文/发件人/发件域/时间范围）
│   │   └── executor.go            # 动作执行器（移动/标记已读/转发/回复）
│   │
│   ├── mail/                      # 多协议邮箱操作
│   │   ├── client.go              # MailClient 接口定义
│   │   ├── factory.go             # 客户端工厂（按协议创建 IMAP/Exchange 实例）
│   │   ├── imap/                  # IMAP 协议实现
│   │   │   ├── client.go          # IMAP 连接、登录、退出
│   │   │   ├── folder.go          # 文件夹操作
│   │   │   └── message.go         # 邮件拉取、移动、删除、标记
│   │   └── exchange/             # Exchange 协议实现
│   │       ├── client.go          # EWS/SOAP 客户端（支持本地 Exchange 和 Exchange Online）
│   │       ├── folder.go          # 文件夹操作
│   │       ├── message.go         # 邮件拉取、移动、删除、标记
│   │       └── graph.go           # Microsoft Graph API 实现（可选，适用于 Exchange Online）
│   │
│   ├── models/                    # 数据模型（跨包共享的结构体）
│   │   ├── mailbox.go             # Mailbox 模型
│   │   ├── message.go             # Message 模型
│   │   └── rule.go                # Rule 模型
│   │
│   ├── store/                     # 持久化存储层
│   │   ├── db.go                  # SQLite 连接初始化、迁移执行
│   │   ├── mailbox_repo.go        # 邮箱表 CRUD
│   │   ├── message_repo.go        # 邮件表 CRUD
│   │   └── rule_repo.go           # 规则表 CRUD
│   │
│   └── scheduler/                 # 定时调度
│       └── scheduler.go           # time.Ticker 每分钟轮询所有邮箱
│
├── configs/                       # 配置文件
│   ├── config.yaml                # 用户实际配置
│   └── config.example.yaml        # 配置模板
│
├── migrations/                    # SQLite 迁移脚本
│   └── 001_init.sql               # 建表语句
│
├── docs/                          # 文档
│   └── superpowers/
│       └── specs/                 # 设计文档
│           └── YYYY-MM-DD-email-organizer-design.md
│
├── Makefile                       # 构建、测试、运行命令
├── go.mod / go.sum
├── AGENTS.md
├── TASKS.md
├── architecture.md                # 本文件
├── decisions.md
└── handoff.md
```

---

## 模块关系

```
┌─────────────────────────────────────────────────────────────────┐
│                        cmd/organizer/main.go                    │
│                    (依赖注入 + 启动入口)                         │
└──────────┬──────────────────────┬────────────────────┬─────────┘
           │                      │                    │
           ▼                      ▼                    ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│   internal/api   │  │ internal/engine  │  │internal/scheduler│
│  (RESTful API)   │  │  (规则引擎)      │  │  (定时调度器)     │
│                  │  │                  │  │                  │
│  Handler →       │  │  Engine →        │  │  Ticker →        │
│  Response        │  │  Matcher         │  │  Engine.Execute()│
│                  │  │  → Executor      │  │                  │
└────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘
         │                     │                      │
         │                     ▼                      │
         │            ┌──────────────────────────────────┐
         └───────────►│       internal/mail              │◄───────────┘
                      │   (MailClient 接口 + Factory)    │
                      │                                  │
                      │  ┌──────────┐  ┌──────────────┐  │
                      │  │ imap/    │  │ exchange/    │  │
                      │  │ (go-imap)│  │ (EWS / Graph)│  │
                      │  └──────────┘  └──────────────┘  │
                      └────────────┬─────────────────────┘
                                   │
                         ┌─────────▼─────────┐
                         │   internal/store  │
                         │  (SQLite 存储层)  │
                         │                   │
                         │  MailboxRepo      │
                         │  MessageRepo      │
                         │  RuleRepo         │
                         └───────────────────┘
```

### 依赖方向

```
api → mail (接口), store, models
engine → mail (接口), store, models
scheduler → engine, mail (接口)
mail → models
  ├── imap/ → go-imap, models
  └── exchange/ → EWS/SOAP or Graph client, models
store → models (via go-sqlite3)
```

**核心原则**：
- 依赖指向抽象（`MailClient` 接口），不指向具体实现
- `api/engine/scheduler` 只依赖 `mail` 包导出的接口，对 `imap/` 和 `exchange/` 子包完全无感
- `mail/factory.go` 根据配置中的 `protocol` 字段创建对应的客户端实现
- 每个包在需要时定义自己的接口，实现包满足这些接口。

---

## 数据流

### 1. 启动流程

```
main.go 启动
  │
  ├── config.Load()           ← 读取 config.yaml
  ├── store.InitDB()          ← 连接 SQLite，执行迁移
  ├── mail.InitClients()      ← 根据配置按 protocol 创建 IMAP 或 Exchange 客户端并登录
  ├── engine.Init()           ← 加载规则到引擎
  ├── api.InitRouter()        ← 注册 API 路由
  ├── scheduler.Start()       ← 启动定时轮询
  └── api.StartServer()       ← 启动 HTTP 服务
```

### 2. 定时轮询流程

```
scheduler (每分钟触发)
  │
  ├── for each mailbox:
  │   ├── client.FetchNewMessages(lastSyncTime)
  │   │     ↓
  │   │   store messages (新邮件入库)
  │   │     ↓
  │   └── for each matched rule (trigger=onArrival):
  │         engine.Execute(rule, message)
  │           │
  │           ├── matcher.Matches(rule, message)
  │           │     ↓ true
  │           └── executor.Execute(rule, message)
  │                 ├── MoveToFolder → client.MoveMessage()  (IMAP MOVE / EWS MoveItem)
  │                 ├── MarkAsRead   → client.MarkRead()      (IMAP +FLAGS / EWS UpdateItem)
  │                 ├── Forward      → client.ForwardMessage() (IMAP+SMTP / EWS ForwardItem)
  │                 └── Reply        → client.ReplyMessage()  (SMTP / EWS ReplyToItem)
  │
  └── update lastSyncTime
```

### 3. API 请求流程

```
HTTP Request
  │
  ├── middleware (logging, recovery)
  ├── router → handler
  │     ├── handler_mailbox.go
  │     │   ├── GET  /api/mailboxes              → store.ListMailboxes()
  │     │   ├── GET  /api/mailboxes/{id}/folders  → mail.GetFolders(client)
  │     │   └── GET  /api/mailboxes/{id}/folders/{folder}/messages
  │     │         → store.ListMessages(mailboxID, folder, page, size)
  │     │
  │     └── handler_rule.go
  │         ├── GET    /api/rules         → store.ListRules()
  │         ├── POST   /api/rules         → store.CreateRule(rule)
  │         ├── GET    /api/rules/{id}    → store.GetRule(id)
  │         ├── PUT    /api/rules/{id}    → store.UpdateRule(id, rule)
  │         ├── DELETE /api/rules/{id}    → store.DeleteRule(id)
  │         └── PATCH  /api/rules/{id}/toggle → store.ToggleRule(id)
  │
  └── response.JSON(w, data, status)
```

---

## 核心服务

### 1. 配置管理 (internal/config)

- 支持 YAML 配置文件
- 配置结构：

```go
type Config struct {
    API      APIConfig      `yaml:"api"`
    Mailboxes []MailboxConfig `yaml:"mailboxes"`
    Rules    []RuleConfig   `yaml:"rules"`
}

type MailboxConfig struct {
    Name         string `yaml:"name"`
    Protocol     string `yaml:"protocol"`     // "imap" | "exchange"
    // IMAP 协议配置
    IMAPServer   string `yaml:"imap_server"`
    IMAPPort     int    `yaml:"imap_port"`
    // Exchange 协议配置
    ExchangeType string `yaml:"exchange_type"`  // "ews" | "graph"
    EWSEndpoint  string `yaml:"ews_endpoint"`   // EWS 的 SOAP 端点 URL
    TenantID     string `yaml:"tenant_id"`      // Microsoft Graph OAuth2 租户 ID
    ClientID     string `yaml:"client_id"`      // Microsoft Graph OAuth2 客户端 ID
    // 通用字段
    Email        string `yaml:"email"`
    AuthCode     string `yaml:"auth_code"`  // IMAP 授权码 / Exchange 密码或 OAuth Secret
    FetchCount   int    `yaml:"fetch_count"` // 默认500
}

type RuleConfig struct {
    Enabled       bool     `yaml:"enabled"`
    Name          string   `yaml:"name"`
    Trigger       string   `yaml:"trigger"`        // "on_arrival" | "on_delete"
    Conditions    Condition `yaml:"conditions"`
    Action        Action   `yaml:"action"`
}

type Condition struct {
    SubjectContains   *string `yaml:"subject_contains"`
    BodyContains      *string `yaml:"body_contains"`
    SenderEmail       *string `yaml:"sender_email"`
    SenderDomain      *string `yaml:"sender_domain"`
    TimeRangeStart    *string `yaml:"time_range_start"` // HH:mm
    TimeRangeEnd      *string `yaml:"time_range_end"`   // HH:mm
    // 多个条件之间的关系：AND（全部满足才触发）
}

type Action struct {
    Type       string `yaml:"type"`       // "move" | "mark_read" | "forward" | "reply"
    Target     string `yaml:"target"`     // 目标文件夹名 / 转发邮箱
    ReplyText  string `yaml:"reply_text"` // 回复内容（仅 reply 动作）
}
```

### 2. 多协议邮箱客户端 (internal/mail)

设计为**接口 + 工厂模式**，统一 `MailClient` 接口，根据配置的 `protocol` 创建对应的实现。

```go
// client.go — 统一接口
type MailClient interface {
    Login() error
    Logout() error
    ListFolders() ([]Folder, error)
    FetchMessages(folder string, count int, since time.Time) ([]*Message, error)
    MoveMessage(folder string, uid string, targetFolder string) error
    MarkRead(folder string, uid string) error
    DeleteMessage(folder string, uid string) error
    ForwardMessage(msg *Message, targetEmail string) error
    ReplyMessage(msg *Message, replyText string) error
}

type MailClientFactory interface {
    Create(config MailboxConfig) (MailClient, error)
}
```

#### IMAP 实现 (`internal/mail/imap/`)

- 基于 `emersion/go-imap/v2`
- 连接管理：每个邮箱一个独立连接，支持断线重连（最多重试3次）
- IMAP 命令映射：
  - `ListFolders()` → `LIST "" "*"`
  - `FetchMessages(folder, N, since)` → `FETCH 1:* (FLAGS INTERNALDATE BODY[]...)` 并筛选
  - `MoveMessage()` → `MOVE uid targetFolder`
  - `MarkRead()` → `STORE uid +FLAGS (\Seen)`
  - `ForwardMessage()` → 通过 SMTP 发送到目标邮箱
  - 缓存连接状态，避免重复登录

#### Exchange 实现 (`internal/mail/exchange/`)

提供两种 Exchange 接入方式，优先使用 **EWS** 以兼容本地部署和 Exchange Online：

**方式一：EWS (Exchange Web Services)** — 主实现
- 基于 SOAP/XML 协议，调用 Exchange 的 Web 服务接口
- 认证方式：Basic Auth（本地 Exchange）或 OAuth2（Exchange Online）
- 核心操作映射：
  - `ListFolders()` → `FindFolder` SOAP 请求
  - `FetchMessages(folder, count, since)` → `FindItem` + `GetItem` SOAP 请求
  - `MoveMessage()` → `MoveItem` SOAP 请求
  - `MarkRead()` → `UpdateItem` SOAP 请求（设置 IsRead=true）
  - `DeleteMessage()` → `DeleteItem` SOAP 请求
  - `ForwardMessage()` → `CreateItem` (Forward) SOAP 请求
  - `ReplyMessage()` → `CreateItem` (ReplyTo) SOAP 请求
- EWS 端点格式：`https://<exchange-server>/EWS/Exchange.asmx`
- 实现方式：使用 `encoding/xml` 手动构造 SOAP 请求

**方式二：Microsoft Graph API** — 可选实现（仅 Exchange Online）
- REST API，使用 OAuth2 认证
- 需要注册 Azure AD 应用获取 `ClientID` / `TenantID`
- 核心操作映射：
  - `ListFolders()` → `GET /me/mailFolders`
  - `FetchMessages()` → `GET /me/mailFolders/{id}/messages`
  - `MoveMessage()` → `POST /me/messages/{id}/move`
  - `MarkRead()` → `PATCH /me/messages/{id} (isRead: true)`

### 3. 规则引擎 (internal/engine)

- **Rule**：规则定义，包含条件列表和动作
- **Matcher**：检查一条消息是否匹配规则的所有条件
  - 主题包含：`strings.Contains(msg.Subject, condition.SubjectContains)`
  - 正文包含：`strings.Contains(msg.Body, condition.BodyContains)`
  - 发件人匹配：`msg.From == condition.SenderEmail`
  - 发件域匹配：`strings.HasSuffix(msg.FromDomain, condition.SenderDomain)`
  - 时间范围：`msg.ReceivedTime` 在 `[start, end]` 区间内
- **Executor**：执行规则定义的动作
  - 移动：调用 `client.MoveMessage()`
  - 标记已读：调用 `client.MarkRead()`
  - 转发：通过 SMTP 发送到目标邮箱
  - 回复：通过 SMTP 回复指定内容
- **Engine**：主循环，加载规则列表，对每条新消息/删除事件执行匹配

### 4. 存储层 (internal/store)

- SQLite 数据库，文件存储在 `data/email-organizer.db`
- 数据访问对象（DAO）模式：MailboxRepo, MessageRepo, RuleRepo
- 操作：标准 CRUD + 分页查询

### 5. 调度器 (internal/scheduler)

- 使用 `time.NewTicker(1 * time.Minute)` 实现
- 每次触发：
  1. 遍历所有已登录邮箱
  2. 调用 `FetchNewMessages()` 获取新邮件
  3. 对新邮件触发 `trigger=on_arrival` 的规则
  4. 检查是否有邮件被删除，触发 `trigger=on_delete` 的规则
- 支持手动触发（通过 API）
- 故障恢复：连接断开后自动重试

### 6. RESTful API (internal/api)

- 使用 `chi` 框架，轻量高性能
- 路由设计：

```
GET    /api/mailboxes                          # 获取所有邮箱
GET    /api/mailboxes/:id/folders              # 获取邮箱的文件夹列表
GET    /api/mailboxes/:id/folders/:folder/messages  # 获取文件夹邮件（分页）
GET    /api/mailboxes/:id/folders/:folder/messages/:msgId  # 邮件详情

GET    /api/rules                              # 获取所有规则
POST   /api/rules                              # 创建规则
GET    /api/rules/:id                          # 获取单条规则
PUT    /api/rules/:id                          # 更新规则
DELETE /api/rules/:id                          # 删除规则
PATCH  /api/rules/:id/toggle                   # 启用/禁用规则
```

- 统一响应格式：

```json
{
    "code": 0,
    "message": "success",
    "data": { ... }
}
```

---

## 数据库结构

### 表: mailboxes

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | INTEGER | PK AUTOINCREMENT | 主键 |
| name | TEXT | NOT NULL | 邮箱别名 |
| protocol | TEXT | NOT NULL DEFAULT 'imap' | 'imap' / 'exchange' |
| email | TEXT | NOT NULL UNIQUE | 邮箱地址 |
| imap_server | TEXT | | IMAP 服务器地址（protocol=imap 时使用） |
| imap_port | INTEGER | DEFAULT 993 | IMAP 端口（protocol=imap 时使用） |
| exchange_type | TEXT | | 'ews' / 'graph'（protocol=exchange 时使用） |
| ews_endpoint | TEXT | | EWS 端点 URL（protocol=exchange & type=ews 时使用） |
| tenant_id | TEXT | | Graph 租户 ID（protocol=exchange & type=graph 时使用） |
| client_id | TEXT | | Graph 客户端 ID（protocol=exchange & type=graph 时使用） |
| fetch_count | INTEGER | NOT NULL DEFAULT 500 | 拉取数量 |
| last_sync_at | TEXT | | 最后同步时间 (ISO8601) |
| is_active | INTEGER | NOT NULL DEFAULT 1 | 是否启用 |
| created_at | TEXT | NOT NULL | 创建时间 |
| updated_at | TEXT | NOT NULL | 更新时间 |

### 表: messages

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | INTEGER | PK AUTOINCREMENT | 主键 |
| mailbox_id | INTEGER | FK → mailboxes.id | 所属邮箱 |
| uid | TEXT | NOT NULL | 消息唯一 ID（IMAP 用 UID，Exchange 用 ItemId/ConversationId） |
| folder | TEXT | NOT NULL | 所在文件夹 |
| subject | TEXT | | 邮件主题 |
| body_preview | TEXT | | 正文预览（前500字） |
| body | TEXT | | 全文 |
| from_name | TEXT | | 发件人名称 |
| from_email | TEXT | | 发件人邮箱 |
| to_list | TEXT | | 收件人列表（JSON数组）|
| cc_list | TEXT | | 抄送列表（JSON数组）|
| received_at | TEXT | NOT NULL | 收件时间 (ISO8601)|
| flags | TEXT | | 标记（JSON数组）|
| is_deleted | INTEGER | DEFAULT 0 | 是否已删除 |
| created_at | TEXT | NOT NULL | 入库时间 |
| UNIQUE(mailbox_id, uid, folder) | | | 防止重复 |

### 表: rules

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | INTEGER | PK AUTOINCREMENT | 主键 |
| name | TEXT | NOT NULL | 规则名称 |
| enabled | INTEGER | NOT NULL DEFAULT 1 | 是否启用 |
| trigger | TEXT | NOT NULL | on_arrival / on_delete |
| condition_json | TEXT | NOT NULL | 条件 JSON |
| action_json | TEXT | NOT NULL | 动作 JSON |
| priority | INTEGER | DEFAULT 0 | 优先级（数字越小越优先）|
| created_at | TEXT | NOT NULL | 创建时间 |
| updated_at | TEXT | NOT NULL | 更新时间 |

### 表: rule_logs

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | INTEGER | PK AUTOINCREMENT | 主键 |
| rule_id | INTEGER | FK → rules.id | 触发的规则 |
| mailbox_id | INTEGER | FK → mailboxes.id | 所属邮箱 |
| message_id | INTEGER | FK → messages.id | 关联邮件 |
| action | TEXT | NOT NULL | 执行的动作 |
| result | TEXT | NOT NULL | success / failed |
| error_msg | TEXT | | 失败原因 |
| created_at | TEXT | NOT NULL | 执行时间 |

---

## 错误处理策略

- **连接错误**（IMAP 断开 / Exchange 连接超时）：自动重试最多 3 次，间隔递增（1s, 5s, 30s），不同协议的重试逻辑统一由 `mail/factory.go` 管理
- **配置错误**：启动时校验，失败则打印详细错误并退出
- **规则执行失败**：记录到 rule_logs 表，不影响后续规则执行
- **API 错误**：统一 JSON 错误响应，HTTP 状态码符合 REST 语义
  - 200: 成功
  - 400: 请求参数错误
  - 404: 资源不存在
  - 500: 服务器内部错误