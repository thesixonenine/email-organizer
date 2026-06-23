# decisions.md

## 技术决策记录

### D-001: 选择 Go 作为开发语言

- **状态**: ✅ 已确认
- **时间**: 2026-06-23
- **背景**: 需要一个跨平台、高性能、低资源占用的后端服务来处理邮箱操作
- **决策**: 使用 Go 1.26.x
- **理由**:
  - 编译为单一二进制，部署简单
  - 标准库强大，无需大量第三方依赖
  - goroutine 模型天然适合多邮箱并发处理
  - 良好的 IMAP 库生态（go-imap）
- **替代方案**: Python（部署依赖重）、Java/Node（资源占用大）、Rust（开发周期长）

### D-002: 配置存储使用 SQLite

- **状态**: ✅ 已确认
- **时间**: 2026-06-23
- **背景**: 需要持久化存储邮箱元数据、邮件缓存、规则配置，同时便于本地部署
- **决策**: 使用 SQLite (mattn/go-sqlite3) 作为数据库
- **理由**:
  - 嵌入式，零配置，无需单独数据库服务
  - 支持标准 SQL，查询灵活
  - 对于单机应用性能足够
  - 配合 RESTful API 使用简单
- **替代方案**: PostgreSQL（重了）、bolt/bbolt（K/V 不支持 SQL 查询）、JSON 文件（不支持复杂查询和并发）

### D-003: 使用 chi 作为 HTTP 路由框架

- **状态**: ✅ 已确认
- **时间**: 2026-06-23
- **背景**: 需要提供 RESTful API，但不想引入过重的 Web 框架
- **决策**: 使用 go-chi/chi/v5
- **理由**:
  - 轻量，兼容 net/http 标准库
  - 中间件链设计优雅
  - 社区活跃，文档完善
  - 比 gin/echo 更接近标准库，依赖更少
- **替代方案**: gin（代码生成多，隐式行为多）、echo（依赖较重）、net/http 标准库（路由能力有限）

### D-004: 配置格式使用 YAML

- **状态**: ✅ 已确认
- **时间**: 2026-06-23
- **背景**: 需要配置文件管理邮箱认证信息和规则
- **决策**: 使用 YAML 格式，通过 viper 库加载
- **理由**:
  - 可读性好，支持注释
  - 支持复杂嵌套结构（规则条件的天然表达）
  - viper 支持多配置源（文件、环境变量），未来扩展性好
- **替代方案**: JSON（无注释）、TOML（社区支持不如 YAML）、环境变量（复杂结构难表达）

### D-005: 双协议邮箱支持（IMAP + Exchange）

- **状态**: ✅ 已确认
- **时间**: 2026-06-23
- **背景**: 用户需要同时支持 IMAP 和 Exchange 两种邮箱协议
- **决策**: 采用**接口 + 工厂模式**，定义统一 `MailClient` 接口，`mail/factory.go` 根据配置中的 `protocol` 字段创建对应的实现
- **理由**:
  - 对上层（api/engine/scheduler）完全透明，切换协议无需修改业务代码
  - IMAP 使用 go-imap v2 实现，Exchange 使用 EWS (SOAP) 为主、Microsoft Graph 为可选
  - EWS 兼容性最好：同时支持本地 Exchange Server (2007+) 和 Exchange Online
  - Microsoft Graph 只适用于 Exchange Online，但 REST API 更现代
- **替代方案**: 单一协议（牺牲 Exchange 用户）、两个独立客户端（代码重复、维护成本高）
- **注意**: EWS 需要自行构造 SOAP/XML 请求；Graph 需要 OAuth2 和 Azure AD 注册

### D-006: 规则引擎架构

- **状态**: ✅ 已确认
- **时间**: 2026-06-23
- **背景**: 需要灵活可扩展的规则匹配和动作执行系统
- **决策**: 规则引擎采用匹配器(Matcher) + 执行器(Executor) 分离架构
- **理由**:
  - 单一职责：Matcher 只负责判断是否匹配，Executor 只负责执行动作
  - 易扩展：新增匹配条件或执行动作时无需修改核心引擎代码
  - 可测试：匹配逻辑和动作逻辑可独立测试
  - 规则以 JSON 形式存储，便于 API 序列化

### D-007: API 认证策略（待确认）

- **状态**: ❓ 待确认
- **时间**: 2026-06-23
- **背景**: RESTful API 需要防止未授权访问
- **决策**: 待确定
- **待确认项**:
  - 选项 A: 简单 Token（API 配置中预置）
  - 选项 B: 基本认证（用户名/密码）
  - 选项 C: JWT（适合多前端场景）
  - 选项 D: 仅限本地监听（127.0.0.1），依赖反向代理认证
- **建议**: 初期使用选项 D（127.0.0.1 绑定）+ 选项 A（固定 Token）组合，后续可升级 JWT

### D-008: 定时轮询策略

- **状态**: ✅ 已确认
- **时间**: 2026-06-23
- **背景**: 需要及时发现新邮件并执行规则
- **决策**: 使用 time.Ticker 每分钟轮询，每次查询自上次同步后的新邮件
- **理由**:
  - IMAP IDLE 扩展虽然支持推送，但对多邮箱实现复杂
  - 分钟级轮询对邮件整理场景足够实时
  - 实现简单可靠
- **替代方案**: IMAP IDLE（实时性好，但连接多时资源消耗大、实现复杂）

### D-009: 授权码管理

- **状态**: ✅ 已确认
- **时间**: 2026-06-23
- **背景**: 需要使用授权码（而非密码）登录邮箱
- **决策**: 授权码存储在配置文件中，数据库不存储原始授权码
- **理由**:
  - 安全性：配置文件权限设为 600，敏感信息不落入数据库
  - .gitignore 中已排除 config.yaml，避免误提交
  - 后续可考虑加密存储

### D-017: SQL Schema 内嵌策略

- **状态**: ✅ 已确认
- **时间**: 2026-06-23
- **背景**: 需要将 `migrations/001_init.sql` 在运行时加载到 SQLite 数据库
- **决策**: 使用 Go const 字符串将 schema SQL 直接内嵌在 `internal/store/schema.go` 中
- **理由**:
  - `//go:embed` 不支持包含 `..` 的路径，无法从 `internal/store/` 嵌入 `migrations/001_init.sql`
  - 内嵌为 const 字符串是 Go 中最简单可靠的方案
  - `migrations/001_init.sql` 保留为参考副本
- **替代方案**: `//go:embed` + 将 migration 文件移动到 `internal/store/` 子目录（增加了文件位置不一致的问题）

### D-018: MailClient 接口放置策略

- **状态**: ✅ 已确认
- **时间**: 2026-06-23
- **背景**: `mail/factory.go` 需要引用 `mail/imap`、`mail/exchange` 包，而这两个包需要实现 `MailClient` 接口，但接口定义在 `mail` 包中会导致循环依赖
- **决策**: 将 `MailClient` 接口和 `Folder` 类型抽取到单独的 `mail/types` 子包中
- **理由**:
  - `mail/types` 包只依赖 `models` 包（无循环依赖）
  - `mail/clients.go` 通过类型别名 `type MailClient = types.MailClient` 保持外部 API 一致
  - `mail/factory.go` 引用 `mail/imap`、`mail/exchange` 和 `mail/types`（无循环）
- **注意**: 外部使用者仍通过 `mail.MailClient` 访问接口，无需知道 `mail/types` 的存在

### D-019: go-imap v2 命令结果处理方式

- **状态**: ✅ 已确认
- **时间**: 2026-06-23
- **背景**: go-imap v2 beta.8 的 API 与 v1 完全不同，命令结果通过 `.Wait()` 或 `.Collect()` 获取
- **决策**: 使用 `.Collect()` 获取批量结果，使用 `.Wait()` 获取单条结果
- **理由**:
  - `FetchCommand.Collect()` → `[]*FetchMessageBuffer`（批量消息数据）
  - `UIDSearch.SearchData.All` → `imap.UIDSet` 类型断言后调用 `.Nums()` 获取 UID 列表
  - `Store()` 返回 `*FetchCommand`，调用 `.Collect()` 忽略结果
  - `Expunge()` 返回 `*ExpungeCommand`，调用 `.Close()` 等待完成
- **注意**: go-imap v2 当前为 beta.8，正式版 API 可能变化

### D-020: 授权码缓存策略

- **状态**: ✅ 已确认
- **时间**: 2026-06-23
- **背景**: 数据库不存储授权码（安全策略 D-009），但运行时需要授权码创建 MailClient 实例
- **决策**: main.go 启动时从 `config.yaml` 读取授权码，缓存在内存中的 `clientCache` 里，通过 `mailFactory` 闭包携带
- **理由**:
  - 授权码仅在进程生命周期内存中存在
  - 不落入数据库，不写入日志
  - 进程重启后需要重新读取配置文件
- **注意**: mailFactory 闭包需要同时访问 config（授权码）和 DB（其他配置）

---

## 待确认决策

| ID | 决策项 | 建议方案 | 优先级 | 讨论 |
|----|--------|---------|-------|------|
| D-007 | API 认证策略 | 本地监听 + Token | P2 | 需要在安全性和便利性之间平衡 |
| D-011 | 邮件正文存储策略 | 全文存储或仅索引 | P2 | 全文可支持正文条件匹配，但占用空间大；大邮件需截断 |
| D-012 | 并发邮箱操作的 goroutine 模型 | 每邮箱一个 goroutine 或协程池 | P2 | 邮箱数量少时每邮箱一个 goroutine 即可 |
| D-013 | 日志文件管理 | 轮转日志或固定大小 | P3 | 可使用 logrotate 外部处理 |
| D-014 | 邮件转发 SMTP 配置 | 复用原邮箱 SMTP 或统一 SMTP | P1 | 转发用原邮箱 SMTP 更真实，但配置项增多 |
| D-015 | Exchange EWS 库选择 | 使用 Go 标准库 encoding/xml 手动调用 | P1 | Go 生态 EWS 库较少且活跃度低，手动调用更可控 |
| D-016 | Exchange Graph 实现时机 | 先实现 EWS，Graph 作为后续迭代 | P2 | EWS 兼容性更广，Graph 只适用于 Exchange Online |

---

## 架构决策记录 (ADR) 索引

| ADR | 标题 | 状态 | 文件 |
|-----|------|------|------|
| ADR-001 | 选择 Go + SQLite + chi 技术栈 | ACCEPTED | 本文件 D-001~D-003 |
| ADR-002 | YAML 配置 + Viper 加载 | ACCEPTED | D-004 |
| ADR-003 | IMAP + Exchange 双协议（接口+工厂模式） | ACCEPTED | D-005 |
| ADR-004 | 规则引擎 Matcher/Executor 分离 | ACCEPTED | D-006 |
| ADR-005 | 分钟级轮询而非 IMAP IDLE | ACCEPTED | D-008 |
| ADR-006 | SQL Schema 内嵌为 Go const | ACCEPTED | D-017 |
| ADR-007 | MailClient 接口抽取到 mail/types 子包 | ACCEPTED | D-018 |
| ADR-008 | go-imap v2 beta Collect/Wait 模式 | ACCEPTED | D-019 |
| ADR-009 | 授权码内存缓存 + config 加载 | ACCEPTED | D-020 |

---

## 撤回或否决的决策

_（暂无）_