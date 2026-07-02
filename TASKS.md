# TASKS.md

## 项目状态总览

```
[████████████████░░░░░░] 75% 完成
```

> **最后更新**: 2026-07-02
> **当前阶段**: 功能完善中（SMTP 转发/回复已完成）
> **当前分支**: feature/smtp-forward-reply
> **最新 Commit**: `03e7646`

---

## 已完成 ✅

| ID | 任务 | 完成日期 | 说明 |
|----|------|---------|------|
| T-001 | 项目初始化 | 2026-06-23 | go module 初始化，.gitignore 配置，LICENSE 添加 |
| T-002 | 设计文档生成 | 2026-06-23 | 创建 AGENTS.md, TASKS.md, architecture.md, decisions.md, handoff.md |
| T-003 | 项目目录结构搭建 | 2026-06-23 | 创建所有目录、Makefile、migration SQL、main.go 桩代码 |
| T-004 | 配置管理模块 | 2026-06-23 | YAML 配置加载含 IMAP+Exchange 字段校验（TDD, 3/3 测试通过）|
| T-005 | 数据模型定义 | 2026-06-23 | Mailbox, Message, Rule, RuleLog, Condition, Action 结构体 |
| T-006 | 数据库初始化与迁移 | 2026-06-23 | SQLite + WAL 模式，schema 嵌在代码中（TDD, 1/1 测试通过）|
| T-007 | MailClient 接口 + 工厂模式 | 2026-06-23 | 接口定义在 `mail/types` 子包，工厂按 protocol 创建 IMAP/Exchange |
| T-008 | IMAP 客户端实现 | 2026-06-23 | 基于 go-imap v2 beta.8：登录/文件夹列表/邮件拉取/移动/标记删除 |
| T-009 | Exchange EWS 客户端实现 | 2026-06-23 | SOAP/XML：FindFolder/FindItem/MoveItem/UpdateItem 等 |
| T-010 | 存储层实现 | 2026-06-23 | Mailbox/Message/Rule Repository（CRUD + 分页 + 批量插入）|
| T-011 | 规则引擎核心 + 匹配器 + 执行器 | 2026-06-23 | Matcher（5条件AND组合）+ Executor（4动作）+ ProcessMessage |
| T-012 | 调度器实现 | 2026-06-23 | time.Ticker 分钟级轮询所有邮箱，增量拉取新邮件 |
| T-013 | RESTful API（路由/中间件/处理器） | 2026-06-23 | chi router + 10个端点 + 日志/恢复中间件 + 统一响应格式 |
| T-014 | 主入口依赖注入 | 2026-06-23 | cmd/organizer/main.go 完整 wiring |
| T-015 | 配置示例 | 2026-06-23 | configs/config.example.yaml 含 IMAP+EWS+Graph 三种示例 |
| T-021 | SMTP 转发/回复实现 | 2026-07-02 | SMTP 客户端、工厂集成、IMAP/EWS/Graph Forward/Reply 实现 |

---

## 进行中 🚧

| ID | 任务 | 负责人 | 开始日期 | 预期完成 | 状态 |
|----|------|--------|---------|---------|------|
| — | 暂无活跃任务 | — | — | — | ✅ |

---

## Backlog 📋

### M4: 测试覆盖提升（优先级 P1）

| ID | 任务 | 预估工时 | 依赖 | 说明 |
|----|------|---------|------|------|
| T-016 | **Mock IMAP 服务器** | 4h | T-008 | 用于 mail/imap 包的集成测试 |
| T-017 | **Mock Exchange EWS 服务器** | 4h | T-009 | 用于 mail/exchange 包的集成测试 |
| T-018 | **引擎单元测试** | 4h | T-011 | matcher（5条件）+ executor（4动作）+ engine 循环测试 |
| T-019 | **API 层测试** | 4h | T-013 | HTTP 测试覆盖 mailbox + rule 处理 |
| T-020 | **调度器测试** | 2h | T-012 | tick 调用、stop 行为测试 |

### M5: 功能完善（优先级 P2）

| ID | 任务 | 预估工时 | 依赖 | 说明 |
|----|------|---------|------|------|
| T-021 | **SMTP 转发/回复实现** | 6h | 完成 | 2026-07-02 | SMTP 客户端包、IMAP/EWS/Graph 集成，Forward/Reply 真实实现 |
| T-022 | **Graph API 实现** | 8h | — | — | 完成 Microsoft Graph OAuth2 + REST 调用 |
| T-023 | **错误恢复机制** | 3h | T-012 | IMAP 断连重试、Exchange 会话超时重连 |
| T-024 | **API Token 认证** | 2h | T-013 | 增加简单 Token 认证中间件 |

### M6: 部署就绪（优先级 P3）

| ID | 任务 | 预估工时 | 依赖 | 说明 |
|----|------|---------|------|------|
| T-025 | **Docker 化** | 2h | T-020 | Dockerfile + docker-compose |
| T-026 | **CI/CD 配置** | 2h | T-025 | GitHub Actions 自动测试+构建 |
| T-027 | **性能优化** | 4h | T-020 | 大文件夹分批拉取、连接池复用 |
| T-028 | **README 与使用文档** | 3h | T-020 | 运行说明、配置指南、API 文档 |

---

## 里程碑

| 里程碑 | 截止日期 | 交付物 | 状态 |
|--------|---------|--------|------|
| M1: 基础框架 | 2026-06-23 | 项目骨架、配置、数据库、MailClient接口、IMAP+Exchange客户端 | ✅ |
| M2: 规则引擎 | 2026-06-23 | 规则匹配、动作执行、定时轮询 | ✅ |
| M3: RESTful API | 2026-06-23 | 10个API端点：邮件查看+规则管理 | ✅ |
| M4: 测试与文档 | — | mock服务、全面测试覆盖 | ⏳ |
| M5: 功能完善 | — | SMTP转发、Graph API、错误恢复 | ⏳ |
| M6: 部署就绪 | — | Docker、CI/CD、文档 | ⏳ |

---

## 技术债务 / 已知问题

| ID | 问题 | 严重度 | 说明 |
|----|------|--------|------|
| TD-001 | go-imap v2 beta.8 版本 | 低 | beta 版 API 可能变化，正式版需适配 |
| TD-002 | EWS SOAP XML 响应解析 | 中 | 当前实现为简化版，实际 Exchange 服务器响应结构可能不同 |
| TD-003 | Graph API 为 stub | 中 | 需要补充 OAuth2 认证流程 |
| TD-004 | Forward/Reply 未实现 | 中 | ✅ 已解决 — SMTP 客户端已实现，IMAP/EWS/Graph 集成完成 |
| TD-005 | 测试覆盖率低 | 高 | 仅 config 和 store 包有测试（2 个测试文件）|