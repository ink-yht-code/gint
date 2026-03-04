# gint

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

基于 Gin 框架的**微服务开发平台**，采用 Monorepo 架构，提供完整的开发工具链。

## � 项目结构

```
gint/
├── gint/          # 核心库 - Handler 包装器、Session、中间件
├── gintx/         # 运行时框架 - log/tx/db/redis/http/rpc/health
├── gint-gen/      # 代码生成器 - CLI 工具
├── registry/      # ServiceID 注册服务
├── example/       # 示例服务
└── docs/          # 设计文档
```

## � 快速开始

### 安装代码生成器

```bash
cd gint-gen && go install .
```

### 创建新服务

```bash
# 创建 HTTP 服务
gint-gen new service user --transport http

# 创建 HTTP + gRPC 服务
gint-gen new service order --transport http,rpc
```

### 生成代码

```bash
# 从 .gint 文件生成 HTTP 代码
gint-gen api user

# 从 .proto 文件生成 gRPC 代码
gint-gen rpc user

# 从 SQL DDL 生成 repository
gint-gen repo user --ddl schema.sql
```

### 检查分层约束

```bash
gint-gen lint user
```

## 📚 模块说明

### gint - 核心库

轻量级 Gin 增强库，提供：
- **Handler 包装器** - W/B/S/BS 四种包装函数
- **双 Token 认证** - Access Token + Refresh Token
- **Session 管理** - JWT + Redis 混合方案
- **中间件** - 访问日志、限流、CORS

```go
import "github.com/ink-yht-code/gint"
```

### gintx - 运行时框架

提供基础设施组件：
- **log** - zap 日志 + ctx logger
- **tx** - 事务管理 + ctx 注入 DB
- **db** - GORM 初始化
- **redis** - Redis 初始化
- **httpx** - Gin server + middleware
- **rpc** - gRPC server + interceptor
- **health** - HTTP /health + gRPC Health
- **outbox** - Outbox 模式

```go
import "github.com/ink-yht-code/gintx/log"
import "github.com/ink-yht-code/gintx/httpx"
```

### gint-gen - 代码生成器

CLI 工具，支持：
- `new service` - 创建服务骨架
- `api` - 从 .gint 生成 HTTP 代码
- `rpc` - 从 .proto 生成 gRPC 代码
- `repo` - 从 SQL 生成 repository
- `lint` - 分层约束检查

### registry - ServiceID 注册服务

集中式 ServiceID 分配服务：
- HTTP API: `POST /v1/services:allocate`
- SQLite 存储
- 可选 Token 认证

## 🔧 构建命令

### Linux/macOS
```bash
make build      # 构建所有模块
make test       # 运行测试
make install    # 安装 gint-gen 到 GOPATH/bin
make lint       # 代码检查
```

### Windows (PowerShell)
```powershell
.\build.ps1 build      # 构建所有模块
.\build.ps1 test       # 运行测试
.\build.ps1 install    # 安装 gint-gen 到 GOPATH/bin
.\build.ps1 registry   # 运行 registry 服务
.\build.ps1 example    # 运行示例服务
```

### 直接使用 Go 命令
```bash
# 安装 gint-gen
cd gint-gen && go install .

# 使用
gint-gen new service user
```

## 📖 设计文档

- [设计文档](./docs/DESIGN.md) - 完整架构设计

## 📄 开源协议

本项目采用 [Apache License 2.0](./LICENSE) 开源协议。
