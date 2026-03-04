# gint-gen - 代码生成器

gint-gen 是 gint 框架的代码生成 CLI 工具。

## 安装

```bash
cd gint-gen && go install .
```

## 命令

### new service - 创建服务骨架

```bash
# 创建 HTTP 服务
gint-gen new service user --transport http

# 创建 HTTP + gRPC 服务
gint-gen new service order --transport http,rpc
```

选项：
- `--transport, -t` - 传输协议: `http`, `rpc`, `http,rpc` (默认: http)
- `--dao` - DAO 类型: `gorm` (默认: gorm)
- `--cache` - Cache 类型: `redis` (默认: redis)

生成的目录结构：
```
user/
├── cmd/main.go
├── configs/user.yaml
├── internal/
│   ├── config/
│   ├── domain/
│   │   ├── errs/
│   │   ├── entity/
│   │   ├── port/
│   │   └── event/
│   ├── repository/
│   │   ├── dao/
│   │   ├── cache/
│   │   └── outbox/
│   ├── types/
│   ├── server/
│   └── web/
├── user.gint
└── go.mod
```

### api - 生成 HTTP 代码

从 `.gint` 文件生成 HTTP 代码。

```bash
gint-gen api user
```

`.gint` 文件格式：
```
syntax = "v1"

type CreateUserReq {
    Name  string `json:"name"`
    Email string `json:"email"`
}

type CreateUserResp {
    Id    int64  `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

@server(
    prefix: /api/v1
)
service user {
    @handler CreateUser
    POST /users (CreateUserReq) returns (CreateUserResp)
    
    @handler GetUser
    GET /users/:id returns (GetUserResp)
}
```

### rpc - 生成 gRPC 代码

从 `.proto` 文件生成 gRPC 代码。

```bash
gint-gen rpc user
```

需要安装 `protoc` 和 Go 插件。

### repo - 生成 Repository

从 SQL DDL 生成 DAO 和 Repository 代码。

```bash
gint-gen repo user --ddl schema.sql
```

### lint - 分层约束检查

检查代码分层约束，防止跨层调用。

```bash
gint-gen lint user
```

默认约束：
- `web` 层不能导入 `repository`、`dao`
- `server` 层不能导入 `web`
- `repository` 层不能导入 `server`

## 模板

模板位于 `template/templates.go`，可以自定义修改。

生成的代码使用 gint 功能：
- `gint.Handler` 接口 (`PrivateRoutes`/`PublicRoutes`)
- `gint.W/B/S` 包装器
- `gint.validator` 参数校验
- `gintx` 运行时组件

## 与 Registry 集成

创建服务时自动从 Registry 分配 ServiceID：

```bash
# 启动 Registry
make run-registry

# 创建服务（自动分配 ServiceID）
gint-gen new service user
```

## License

Apache License 2.0
