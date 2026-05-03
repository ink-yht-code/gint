# JWT 令牌管理

`jwt` 包负责令牌的生成与校验，支持访问令牌和刷新令牌。

## 快速开始

```go
import (
	"time"

	"github.com/ink-yht-code/gint/jwt"
)

opts := jwt.NewOptions(
	"your-secret-key",
	2*time.Hour,
	7*24*time.Hour,
)
opts.Issuer = "my-app"

manager := jwt.NewManager(opts)
```

## 声明（Claims）

```go
type Claims struct {
	UserId string            `json:"user_id"`
	SSID   string            `json:"ssid"`
	Data   map[string]string `json:"data"`
	jwt.RegisteredClaims
}
```

字段说明：

- `UserId`：用户标识
- `SSID`：会话标识
- `Data`：附加字段

## 生成令牌

```go
token, err := manager.GenerateToken(jwt.Claims{
	UserId: "u-100",
	SSID:   "s-abc",
})
```

## 生成令牌对

```go
pair, err := manager.GenerateTokenPair(jwt.Claims{
	UserId: "u-100",
	SSID:   "s-abc",
	Data: map[string]string{
		"role": "admin",
	},
})
if err != nil {
	// 处理错误
}

_ = pair.AccessToken
_ = pair.RefreshToken
```

## 校验令牌

```go
claims, err := manager.VerifyToken(accessToken)
if err != nil {
	// token 无效或已过期
}
_ = claims
```

## 校验刷新令牌

```go
claims, err := manager.VerifyRefreshToken(refreshToken)
if err != nil {
	// refresh token 无效或已过期
}
_ = claims
```

## 配合认证中间件

```go
r.Use(auth.Middleware(auth.Config{
	JWTManager: manager,
}))
```

## 建议

- 访问令牌用于业务请求
- 刷新令牌建议放在受控载体中，如 HttpOnly Cookie 或专用请求头
- 多服务之间要统一签名配置与过期策略
