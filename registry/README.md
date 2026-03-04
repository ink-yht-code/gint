# registry - ServiceID 注册服务

Registry 是集中式 ServiceID 分配服务，确保微服务 ID 全局唯一。

## 功能

- **ServiceID 分配** - 幂等分配，同名服务返回相同 ID
- **SQLite 存储** - 轻量级持久化
- **HTTP API** - RESTful 接口
- **可选认证** - Token 保护

## 安装

```bash
cd registry && go build -o bin/registry ./cmd/...
```

## 运行

```bash
./bin/registry
# 或
make run-registry
```

默认配置：
- 地址: `:8765`
- 数据库: `registry.db`

## HTTP API

### 分配 ServiceID

```bash
POST /v1/services:allocate
Content-Type: application/json

{
    "name": "user"
}
```

响应：
```json
{
    "id": 1,
    "name": "user",
    "service_id": 101
}
```

幂等性：同名服务多次调用返回相同 ID。

### 查询服务

```bash
GET /v1/services/user
```

响应：
```json
{
    "id": 1,
    "name": "user",
    "service_id": 101
}
```

### 列出所有服务

```bash
GET /v1/services
```

响应：
```json
{
    "services": [
        {"id": 1, "name": "user", "service_id": 101},
        {"id": 2, "name": "order", "service_id": 102}
    ]
}
```

## ServiceID 规则

- 起始 ID: 101
- 递增步长: 1
- 格式: `ServiceID * 10000 + BizCode` 生成业务码

例如：
- user 服务 ID=101，业务码范围: 1010000-1019999
- order 服务 ID=102，业务码范围: 1020000-1029999

## 配置

环境变量：
- `REGISTRY_ADDR` - 监听地址 (默认: `:8765`)
- `REGISTRY_DB` - 数据库路径 (默认: `registry.db`)
- `REGISTRY_TOKEN` - 认证 Token (可选)

## 与 gint-gen 集成

```bash
# gint-gen 创建服务时自动调用 Registry
gint-gen new service user
```

## 存储接口

```go
type Store interface {
    // Allocate 分配 ServiceID（幂等）
    Allocate(name string) (*Service, error)
    // Get 获取服务
    Get(name string) (*Service, error)
    // List 列出所有服务
    List() ([]Service, error)
}
```

可扩展为 MySQL、PostgreSQL 等存储。

## License

Apache License 2.0
