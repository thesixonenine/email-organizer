# AGENTS.md

## 技术栈

| 类别 | 技术选择 | 版本 | 说明 |
|------|---------|------|------|
| 语言 | Go | 1.26.x | 主开发语言 |
| IMAP 客户端 | go-imap (emersion/go-imap) | v2 | IMAP 协议邮箱登录、文件夹和邮件操作 |
| Exchange 客户端 | encoding/xml (SOAP) / go-ews | 标准库 | Exchange 协议实现（EWS / Microsoft Graph） |
| HTTP 框架 | chi (go-chi/chi/v5) | v5 | RESTful API 路由 |
| 数据库 | SQLite (mattn/go-sqlite3) | 最新 | 嵌入式持久化存储 |
| 配置管理 | Viper (spf13/viper) | 最新 | 配置文件读取（YAML/JSON） |
| 定时调度 | Go time.Ticker + goroutine | 标准库 | 每分钟邮件轮询 |
| 日志 | slog (log/slog) | 标准库 | 结构化日志 |
| JSON 序列化 | encoding/json | 标准库 | API 数据交互 |
| 测试 | testing + testify | 标准库 + v1 | 单元测试和断言 |
| 构建 | Go Makefile | 项目自定义 | 编译、测试、打包 |

## 代码规范

### 命名约定

- **包名**：小写单数名词，如 `config`、`mail`、`engine`、`api`、`models`
- **文件命名**：snake_case，如 `imap_client.go`、`rule_engine.go`
- **类型/接口**：驼峰大写开头（导出），如 `MailClient`、`RuleEngine`
- **方法/函数**：驼峰大写开头（导出），驼峰小写开头（私有）
- **变量/参数**：驼峰小写开头，如 `mailClient`、`ruleList`
- **常量**：驼峰大写开头，如 `MaxEmailsPerFetch`
- **接口名**：`-er` 后缀，如 `Authenticator`、`RuleMatcher`

### 代码组织

- 每个包一个职责，避免循环依赖
- `internal/` 下的包不允许外部导入
- 接口定义在使用的包中，实现放在实现包中
- 每层通过接口依赖，不直接依赖具体实现

### 错误处理

- 所有错误必须显式处理，不允许 `_ = func()` 忽略错误
- 错误信息使用 `fmt.Errorf("context: %w", err)` 包装
- 上层错误处理区分"可恢复"和"不可恢复"，可恢复的日志记录后继续
- 使用 `errors.Is` / `errors.As` 进行错误判断

### 日志规范

- 使用 `slog` 结构化日志
- 日志级别：`DEBUG`（开发调试）| `INFO`（常规操作）| `WARN`（异常但可恢复）| `ERROR`（不可恢复）
- 关键操作必须记录：邮箱登录成功/失败、规则执行、API 请求
- 禁止打印敏感信息（密码、授权码）

## 项目规则

### 目录结构

```
email-organizer/
├── cmd/
│   └── organizer/          # 应用入口
│       └── main.go
├── internal/
│   ├── api/                # RESTful API 层
│   │   ├── router.go       # 路由注册
│   │   ├── handler.go      # 请求处理器
│   │   ├── response.go     # 统一响应格式
│   │   └── middleware.go   # 中间件（日志、恢复等）
│   ├── config/             # 配置管理
│   │   ├── config.go       # 配置结构定义
│   │   └── loader.go       # 配置加载与验证
│   ├── engine/             # 规则引擎
│   │   ├── engine.go       # 引擎主逻辑
│   │   ├── rule.go         # 规则定义
│   │   ├── matcher.go      # 条件匹配器
│   │   └── executor.go     # 动作执行器
│   ├── mail/               # 多协议邮箱操作（IMAP + Exchange）
│   │   ├── client.go       # MailClient 接口定义
│   │   ├── factory.go      # 客户端工厂
│   │   ├── imap/           # IMAP 实现
│   │   └── exchange/       # Exchange 实现（EWS / Graph）
│   ├── models/             # 数据模型
│   │   ├── mailbox.go      # 邮箱模型
│   │   ├── message.go      # 邮件模型
│   │   └── rule.go         # 规则模型
│   ├── store/              # 持久化存储
│   │   ├── db.go           # 数据库初始化
│   │   ├── mailbox_repo.go # 邮箱数据访问
│   │   ├── message_repo.go # 邮件数据访问
│   │   └── rule_repo.go    # 规则数据访问
│   └── scheduler/          # 定时调度
│       └── scheduler.go    # 轮询调度器
├── configs/
│   ├── config.yaml         # 默认配置文件
│   └── config.example.yaml # 配置示例
├── docs/                   # 文档
│   └── superpowers/
│       └── specs/          # 设计文档
├── migrations/             # 数据库迁移
│   └── 001_init.sql
├── Makefile                # 构建文件
├── go.mod / go.sum
├── AGENTS.md               # 本文件
├── TASKS.md                # 任务跟踪
├── architecture.md         # 架构文档
├── decisions.md            # 决策记录
└── handoff.md              # 交接文档
```

### 测试要求

- 核心逻辑包（`engine`、`mail`、`config`）覆盖率 > 80%
- API 层必须写 HTTP 测试
- 使用 `go test ./...` 全部通过才能提交
- Mock IMAP 服务器用于 mail 包测试

## Agent 工作规则

### 通用规则

1. **阅读上下文** — 在修改代码前先阅读相关文件完整内容
2. **先问再改** — 对于不确定的变更，先提问确认
3. **原子提交** — 每个 commit 只做一件事，消息格式 `type(scope): description`
4. **保持向后兼容** — API 变更前先标记废弃，下个版本移除

### 工作流程

1. 从 TASKS.md 获取当前任务
2. 阅读 architecture.md 理解系统设计
3. 确认没有与其他任务冲突
4. 实现代码，附带测试
5. 运行 `go test ./...` 验证
6. 更新 TASKS.md 标记完成
7. 提交代码

### 代码审查规则

- 任何合并到 main 的代码必须经过审查
- 检查点：编译通过、测试通过、覆盖率达标、无硬编码敏感信息

## 测试规则

- 使用 `testing` 标准库 + `testify` 断言
- 测试文件与被测文件同级，命名 `*_test.go`
- 单元测试不依赖外部服务（IMAP 服务器、数据库等需 mock）
- 集成测试放在 `tests/` 目录，使用 `//go:build integration` 标签
- Makefile 中 `make test` 运行单元测试，`make test-integration` 运行集成测试

## Git 规则

### 分支策略

- `main` — 稳定发布分支
- `feature/<name>` — 功能开发分支
- `fix/<name>` — 修复分支
- `release/<version>` — 发布准备分支

### 提交信息格式

```
<type>(<scope>): <subject>

<body>
```

- type: `feat` | `fix` | `docs` | `style` | `refactor` | `test` | `chore`
- scope: `api` | `engine` | `mail` | `config` | `store` | `scheduler` | `docs`
- subject: 50 字以内，中文描述
- body: 详细说明原因和影响

### 示例

```
feat(engine): 实现邮件整理规则匹配引擎

- 支持主题、正文、发件人、发件域、时间范围条件
- 支持移动、标记已读、转发、回复动作
- 添加单元测试覆盖所有匹配场景
```

### 合并要求

- PR 前必须通过所有测试
- 必须无冲突
- 必须通过代码审查