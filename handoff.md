# handoff.md

## 项目状态快照

> **生成时间**: 2026-07-02 12:00 (CST)
> **当前分支**: `feature/smtp-forward-reply`
> **最新 Commit**: `03e7646` — docs: add SMTP config and forward/reply rule examples
> **项目阶段**: 功能完善中（~75%）

---

## 当前完成度

### 已完成

| 模块 | 完成度 | 说明 |
|------|--------|------|
| 项目初始化 | 100% | go.mod, Makefile, .gitignore, LICENSE |
| 设计文档 | 100% | AGENTS.md, architecture.md, TASKS.md, decisions.md, handoff.md |
| 配置管理 | 100% | YAML 加载 + IMAP/Exchange 校验（3/3 测试通过）|
| 数据模型 | 100% | Mailbox, Message, Rule, RuleLog, RuleCondition, RuleAction |
| 数据库 | 100% | SQLite + WAL + 4 张表 + 3 个 Repository |
| MailClient 接口+工厂 | 100% | `mail/types` 接口 + 工厂按 protocol 创建 |
| IMAP 客户端 | 100% | go-imap v2：登录/文件夹/邮件/移动/标记/删除 |
| Exchange EWS 客户端 | 100% | SOAP/XML：FindFolder/FindItem/MoveItem/UpdateItem |
| 规则引擎 | 100% | Matcher(5条件) + Executor(4动作) + Engine(规则循环) |
| RESTful API | 100% | chi, 10 端点, 统一响应, 日志/恢复中间件 |
| 定时调度器 | 100% | time.Ticker 分钟级轮询 |
| 主入口 wiring | 100% | 依赖注入, 启动登录, 优雅关闭 |
| **整体进度** | **~70%** | 测试覆盖率待提升, 部分功能待完善 |

### 待完成

| 模块 | 完成度 | 说明 |
|------|--------|------|
| 测试覆盖 | 10% | 仅 config + store 有测试（2个测试文件）|
| SMTP 转发/回复 | 100% | SMTP 客户端包、工厂集成、IMAP/EWS/Graph Forward/Reply |
| Microsoft Graph | 0% | stub 实现 |
| 文档 | 20% | 详细 README 待补充 |
| Docker 化 | 0% | 待实现 |
| **整体进度** | **~75%** | SMTP 转发/回复已完成，测试覆盖率待提升 |

---

## 代码统计

```
┌──────────────────────────────┬────────┬─────────────────┐
│ 包                           │ 行数   │ 测试文件         │
├──────────────────────────────┼────────┼─────────────────┤
│ cmd/organizer/               │ 224    │ 无              │
│ internal/config/             │ 245    │ config_test.go  │
│ internal/models/             │ 102    │ 无              │
│ internal/store/              │ 467    │ db_test.go      │
│ internal/mail/               │ 14     │ 无              │
│ internal/mail/types/         │ 25     │ 无              │
│ internal/mail/smtp/          │ 87     │ 无              │
│ internal/mail/imap/          │ 185    │ 无              │
│ internal/mail/exchange/      │ 270    │ 无              │
│ internal/engine/             │ 133    │ 无              │
│ internal/api/                │ 329    │ 无              │
│ internal/scheduler/          │ 72     │ 无              │
├──────────────────────────────┼────────┼─────────────────┤
│ Go 源码总计                  │ ~2200  │ 2 个测试文件     │
│ 配置文件/文档/Makefile       │ ~600   │ —               │
└──────────────────────────────┴────────┴─────────────────┘
```

---

## 当前风险

### 高优先级

| 风险 | 可能性 | 影响 | 缓解措施 |
|------|--------|------|---------|
| 测试覆盖率过低 | 高 | 质量风险 | 优先补充 engine/mail/api 核心包测试 |
| IMAP/Exchange 未在真实服务器验证 | 高 | 兼容性问题 | 需要对照各主流邮箱测试 |
| Forward/Reply 未实现 | 高 | 规则动作无效 | ✅ 已解决 — SMTP 客户端已实现 |
| EWS SOAP 解析可能不完整 | 中 | 特定邮箱故障 | 需要真实 Exchange 服务器测试 |

### 中优先级

| 风险 | 可能性 | 影响 | 缓解措施 |
|------|--------|------|---------|
| go-imap v2 beta.8 API 不稳定 | 中 | 升级成本 | 正式发布后适配 |
| Graph API stub 未实现 | 中 | Exchange Online 不可用 | 后续迭代实现 OAuth2 流程 |
| API 无认证暴露 | 中 | 安全风险 | 默认 127.0.0.1 绑定 |
| 大邮箱拉取性能 | 低 | 同步延迟 | 分批拉取，限制每批 500 封 |

### 低优先级

| 风险 | 可能性 | 影响 | 缓解措施 |
|------|--------|------|---------|
| SQLite 并发写入 | 低 | 写入失败 | WAL 模式已启用 |
| 授权码明文存储 | 低 | 安全风险 | 配置文件权限 600 |

---

## 关键设计决策回顾

| 决策 | 选择 | 原因 |
|------|------|------|
| 开发语言 | Go 1.26 | 部署简单，并发强 |
| 数据库 | SQLite + WAL | 嵌入式零配置 |
| HTTP 框架 | chi v5 | 轻量，兼容标准库 |
| 配置格式 | YAML + Viper | 可读性好，支持嵌套 |
| 协议架构 | MailClient 接口 + 工厂模式 | 避免循环依赖，协议透明 |
| IMAP 实现 | go-imap v2 | 原生 Go 实现 |
| Exchange 实现 | EWS (SOAP) | 兼容本地+在线 |
| Schema 管理 | 内嵌 Go const 字符串 | 避免 embed 路径限制 |
| 定时策略 | 分钟级轮询 | 简单可靠 |
| 规则引擎 | Matcher + Executor 分离 | 可扩展可测试 |

---

## 已知问题

| ID | 问题 | 状态 |
|----|------|------|
| #1 | `mailboxRepo.GetByID(0)` 设计缺陷已修复 | ✅ 已修复 |
| #2 | `mail` 包与 `exchange` 包循环依赖已解决（提取 `mail/types`）| ✅ 已修复 |
| #3 | `//go:embed` 不支持 `..` 路径，改用 Go const | ✅ 已修复 |
| #4 | go-imap v2 beta.8 API 适配多次迭代 | ✅ 已修复 |
| #5 | 取消 `_ = mb` 占位符避免编译警告 | ✅ 已修复 |

---

## 下一步建议

### 立即 (P0)

1. **补充核心测试** — engine/matcher（5条件）+ engine/executor（4动作）+ mail/imap + mail/exchange
2. **在真实邮箱验证** — 用 QQ/163/Gmail 测试 IMAP，用 Exchange Online 测试 EWS

### 短期 (P1)

4. **实现 Graph API** — OAuth2 流程 + REST 调用
5. **API Token 认证** — 简单 Bearer Token 中间件

### 中期 (P2)

6. **Docker 化** — 多阶段构建 Dockerfile
7. **CI/CD** — GitHub Actions 自动测试+构建
8. **README** — 完整使用文档 + API 文档

---

## 联系信息

- **项目目录**: `/home/simple/github/email-organizer/`
- **当前分支**: `feature/smtp-forward-reply`
- **配置文件**: `configs/config.yaml` (用户自建，已 gitignore)
- **配置示例**: `configs/config.example.yaml`
- **数据库文件**: `data/email-organizer.db` (首次运行后生成)
- **构建命令**: `make build` 或 `go build -o bin/organizer ./cmd/organizer/`
- **测试命令**: `make test` 或 `go test ./... -count=1`

---

## 备注

- 本项目使用 Superpowers 工作流，详见 `AGENTS.md`
- 所有任务跟踪在 `TASKS.md` 中
- 架构变更需更新 `architecture.md` 和 `decisions.md`
- 交接前请更新本文件的状态快照