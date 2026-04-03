# JWT管理

`jwt` 模块提供 token 生成与校验能力，支持 access/refresh 双 token。

## 快速开始

```go
import (
    "time"

    "github.com/ink-yht-code/gint/jwt"
)

opts := jwt.NewOptions(
    "your-secret-key", // SignKey
    2*time.Hour,       // Access 过期时间
    7*24*time.Hour,    // Refresh 过期时间
)
opts.Issuer = "my-app"

manager := jwt.NewManager(opts)
```

## Claims 结构

当前默认 Claims：

```go
type Claims struct {
    UserId string            `json:"user_id"`
    SSID   string            `json:"ssid"`
    Data   map[string]string `json:"data"`
    jwt.RegisteredClaims
}
```

说明：

- `UserId`：用户唯一标识
- `SSID`：会话标识，常用于和 Session 存储关联
- `Data`：轻量附加字段

## 生成 Token

### 生成单个 token（兼容模式）

```go
token, err := manager.GenerateToken(jwt.Claims{
    UserId: "u-100",
    SSID:   "s-abc",
})
```

### 生成 token 对（推荐）

```go
pair, err := manager.GenerateTokenPair(jwt.Claims{
    UserId: "u-100",
    SSID:   "s-abc",
    Data: map[string]string{
        "role": "admin",
    },
})
if err != nil {
    // handle
}

// pair.AccessToken
// pair.RefreshToken
```

## 校验 Token

### 校验 access token

```go
claims, err := manager.VerifyToken(accessToken)
if err != nil {
    // token 无效/过期/签名错误
}
_ = claims
```

### 校验 refresh token

```go
claims, err := manager.VerifyRefreshToken(refreshToken)
if err != nil {
    // refresh token 无效/过期
}
_ = claims
```

## 典型登录流程

1. 登录成功后调用 `GenerateTokenPair`
2. access token 用于业务接口鉴权
3. access 过期后，客户端携带 refresh token 重新换取新 token 对

## 与认证中间件配合

`middlewares/auth` 的 `Config.JWTManager` 直接接收 `jwt.Manager`：

```go
r.Use(auth.Middleware(auth.Config{
    JWTManager: manager,
}))
```

## 注意事项

- 当前实现使用对称签名（`SignKey` + `Method`）
- `VerifyToken` 与 `VerifyRefreshToken` 都会校验签名与 `RegisteredClaims`
- refresh token 建议放在受控载体（例如 HttpOnly Cookie 或受控 Header）
