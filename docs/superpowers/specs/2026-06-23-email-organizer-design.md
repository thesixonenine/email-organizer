# 多邮箱整理助手 (Email Organizer) — 设计文档

> **日期**: 2026-06-23
> **状态**: 初稿待审
> **版本**: v1

## 1. 概述

多邮箱整理助手是一个 Go 语言编写的后端服务，旨在帮助用户自动管理多个邮箱的邮件。它通过 IMAP 或 Exchange (EWS/Graph) 协议登录已配置的邮箱，定时拉取新邮件，根据用户定义的规则自动执行整理操作（移动、标记、转发、回复），并提供 RESTful API 供前端管理和查看。

## 2. 目标与范围

### 核心目标

- 支持多个邮箱同时管理（不同服务商均可）
- 灵活的规则引擎，支持多种触发条件与动作组合
- 定时自动整理，无需人工干预
- RESTful API 暴露数据，便于开发前端页面

### 非目标（当前版本不包含）

- 邮件发送（转发/回复除外）
- 邮件客户端全功能（如撰写新邮件）
- 用户认证系统（初期使用本地绑定+简单Token）
- 分布式部署

## 3. 架构概览

### 分层架构

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   API Layer  │     │ Rule Engine  │     │  Scheduler   │
│  (internal/  │     │ (internal/   │     │ (internal/   │
│   api)       │     │  engine)     │     │  scheduler)  │
└──────┬───────┘     └──────┬───────┘     └──────┬───────┘
       │                    │                     │
       └────────────────────┼─────────────────────┘
                            │
                    ┌───────▼───────┐
                    │  Mail Client  │
                    │ (internal/    │
                    │  mail)        │  ← MailClient 接口
                    │                │
                    │  ┌───┐ ┌────┐ │
                    │  │IMAP││EWS/│ │
                    │  │    ││Graph││
                    │  └───┘ └────┘ │
                    └───────┬───────┘
                            │
                    ┌───────▼───────┐
                    │    Store      │
                    │ (internal/    │
                    │  store)       │ ← SQLite
                    └───────────────┘
```

### 核心技术栈

- **语言**: Go 1.26.x
- **IMAP 客户端**: emersion/go-imap v2
- **Exchange 客户端**: encoding/xml (EWS SOAP) + Microsoft Graph API (可选)
- **HTTP 框架**: go-chi/chi v5
- **数据库**: SQLite (mattn/go-sqlite3)
- **配置**: spf13/viper (YAML)

## 4. 功能详情

### 4.1 邮箱管理

- 通过配置文件定义多个邮箱，每个包含协议类型 (`protocol: imap | exchange`)、服务器地址、邮箱地址、授权码
- 启动时根据 `protocol` 字段创建对应的 MailClient 实例（IMAP 或 Exchange），登录并获取文件夹列表
- 默认拉取最近 500 封邮件，可配置
- IMAP 支持 IMAP over TLS (端口 993)
- Exchange 支持 EWS (SOAP, 兼容本地 Exchange 2007+ 和 Exchange Online) 和 Microsoft Graph API (仅 Exchange Online)

### 4.2 定时同步

- 每分钟定时轮询所有邮箱的新邮件
- 记录每个邮箱的最后同步时间（last_sync_at），增量拉取
- 新邮件入库后触发规则匹配

### 4.3 规则引擎

**规则结构**:

```
规则 = {
  是否启用: true/false
  触发条件: on_arrival | on_delete
  判断条件: AND 关系的条件列表
    主题包含: 字符串 (可选)
    正文包含: 字符串 (可选)
    发件人: 邮箱地址 (可选)
    发件域: 域名 (可选)
    时间范围: HH:mm - HH:mm (可选)
  执行动作:
    类型: move | mark_read | forward | reply
    目标: 文件夹名 / 邮箱地址 / 回复文本
}
```

**匹配逻辑**: 所有指定的条件同时满足（AND）才触发。未指定的条件不参与判断。

**执行流程**:

1. 新邮件到达 → 遍历所有 trigger=on_arrival 且 enabled 的规则
2. 按优先级排序
3. 依次匹配，匹配成功后执行动作（通过 MailClient 统一接口，IMAP/Exchange 协议透明）
4. 记录执行日志到 rule_logs 表

### 4.4 RESTful API

**邮件相关**:

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/mailboxes | 获取所有邮箱 |
| GET | /api/mailboxes/{id}/folders | 获取邮箱文件夹 |
| GET | /api/mailboxes/{id}/folders/{folder}/messages | 分页获取邮件 |
| GET | /api/mailboxes/{id}/folders/{folder}/messages/{msgId} | 邮件详情 |

**规则管理**:

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/rules | 获取所有规则 |
| POST | /api/rules | 新建规则 |
| GET | /api/rules/{id} | 获取单条规则 |
| PUT | /api/rules/{id} | 更新规则 |
| DELETE | /api/rules/{id} | 删除规则 |
| PATCH | /api/rules/{id}/toggle | 启用/禁用 |

## 5. 数据模型

### 核心实体关系

```
Mailbox (1) ──── (N) Message
Rule    (1) ──── (N) RuleLog (执行记录)
Rule    (N) ──── (N) Mailbox (通过配置关联)
```

### SQLite 表设计

详见 architecture.md — 涵盖 mailboxes, messages, rules, rule_logs 四张表。

## 6. 配置设计

```yaml
# configs/config.yaml
api:
  host: "127.0.0.1"
  port: 8080

mailboxes:
  - name: "个人QQ邮箱"
    protocol: "imap"
    imap_server: "imap.qq.com"
    imap_port: 993
    email: "user@qq.com"
    auth_code: "your_authorization_code"
    fetch_count: 500

  - name: "公司邮箱"
    protocol: "exchange"
    exchange_type: "ews"
    ews_endpoint: "https://mail.company.com/EWS/Exchange.asmx"
    email: "user@company.com"
    auth_code: "your_password"
    fetch_count: 500

  - name: "Office 365"
    protocol: "exchange"
    exchange_type: "graph"
    tenant_id: "your-tenant-id"
    client_id: "your-client-id"
    email: "user@contoso.onmicrosoft.com"
    auth_code: "your-oauth-secret"
    fetch_count: 500

rules:
  - name: "自动归档通知邮件"
    enabled: true
    trigger: "on_arrival"
    conditions:
      subject_contains: "[通知]"
      time_range_start: "09:00"
      time_range_end: "18:00"
    action:
      type: "move"
      target: "归档/通知"
```

## 7. 错误处理

| 错误类型 | 处理方式 |
|----------|---------|
| IMAP 连接失败 | 重试 3 次（1s/5s/30s），记录日志 |
| Exchange 连接失败 | 重试 3 次（同 IMAP，统一由 factory.go 管理） |
| EWS SOAP 请求失败 | 解析错误详情并重试 |
| Graph API 限流 | 等待 Retry-After 头指定的时间 |
| 规则匹配异常 | 跳过当前规则，继续匹配其他规则 |
| 动作执行失败 | 记录到 rule_logs，不阻塞后续动作 |
| API 参数错误 | 返回 400 + 错误信息 |
| 数据库错误 | 返回 500 + 错误码 |

## 8. 安全考虑

- 默认监听 127.0.0.1，不暴露到外网
- 授权码存储在配置文件，权限设置为 600
- config.yaml 在 .gitignore 中排除
- 数据库不存储原始授权码
- 后续可增加 API Token 认证

## 9. 测试策略

- **单元测试**: engine/matcher, engine/executor, config/loader — mock 依赖
- **集成测试**:
  - `mail/imap` 包使用 mock IMAP 服务器
  - `mail/exchange` 包使用 mock EWS 或 Graph API stub
- **API 测试**: 使用 httptest 包测试 handler
- **覆盖率目标**: 核心包 > 80%；IMAP 和 Exchange 实现各自 > 70%

## 10. 部署

- 编译为单一二进制：`go build -o organizer ./cmd/organizer/`
- 运行方式：`./organizer -config ./configs/config.yaml`
- 进程管理：systemd / supervisor
- 容器化：可选 Docker