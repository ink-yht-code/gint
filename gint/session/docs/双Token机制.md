# 双 Token 机制详解

## 概述

gint 实现了双 Token 机制（Access Token + Refresh Token），提供更安全的身份认证方案，**支持无感刷新**。

> 🎯 **无感刷新**：当 Access Token 过期时，客户端自动使用 Refresh Token 获取新 Token，用户完全无感知，无需重新登录！

### 什么是双 Token？

- **Access Token（访问令牌）** - 短期有效的令牌，用于访问受保护的资源
- **Refresh Token（刷新令牌）** - 长期有效的令牌，用于获取新的 Access Token

### 为什么需要双 Token？

**单 Token 的问题**：
- Token 过期时间短 → 用户频繁登录，体验差
- Token 过期时间长 → 安全风险高，Token 泄露影响大

**双 Token 的优势**：
- ✅ **安全性高** - Access Token 短期有效，即使泄露影响也小
- ✅ **体验好** - Refresh Token 长期有效，无需频繁登录
- ✅ **可控性强** - 可以随时撤销 Refresh Token
- ✅ **灵活性好** - 可以设置不同的过期策略

## Token 中包含的信息

两种 Token 都包含相同的 Claims 信息：

```go
type Claims struct {
    UserId string            `json:"user_id"` // 用户 ID
    SSID   string            `json:"ssid"`    // Session ID
    Data   map[string]string `json:"data"`    // 额外数据（如角色、部门等）
    jwt.RegisteredClaims                      // 标准声明（过期时间等）
}
```

**重要**：Token 中包含 `UserId`，可以直接从 Token 中获取用户 ID，无需额外查询。

## Token 验证机制

### Token 数量

gint 使用**双 Token 机制**，共有 **2 个 Token**：

1. **Access Token（访问令牌）**
   - 短期有效（建议 15 分钟 - 2 小时）
   - 用于访问受保护的 API 接口
   - 每次请求都需要携带

2. **Refresh Token（刷新令牌）**
   - 长期有效（建议 7 天 - 30 天）
   - 用于刷新 Access Token
   - 仅在刷新 Token 时使用

### Token 验证位置

**Token 验证在 gint 项目内部自动完成**，引用项目无需手动实现验证逻辑。

#### 1. 访问受保护接口时：验证 Access Token

当使用 `gint.S()` 或 `gint.BS()` 包装器时，会自动验证 **Access Token**：

```go
r.GET("/profile", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
    // Access Token 已自动验证
    // 如果验证失败，会自动返回 401 Unauthorized
    userId := sess.Claims().UserId
    return gint.Result{Code: 0, Data: userId}, nil
}))
```

**验证流程**：
1. 从请求中提取 Access Token（通过 `tokenCarrier`）
2. 调用 `jwtManager.VerifyToken()` 验证 Token 签名和过期时间
3. 验证 Session 是否存在（Redis 或内存）
4. 如果验证失败，自动返回 401 Unauthorized

#### 2. 刷新 Token 时：验证 Refresh Token

调用 `RenewToken()` 方法时，会验证 **Refresh Token**：

```go
r.POST("/refresh", func(c *gin.Context) {
    ctx := &gctx.Context{Context: c}
    // 验证 Refresh Token 并生成新的 Token 对
    if err := session.GetDefaultProvider().RenewToken(ctx); err != nil {
        c.JSON(401, gin.H{"code": 401, "msg": "刷新失败"})
        return
    }
    c.JSON(200, gin.H{"code": 0, "msg": "刷新成功"})
})
```

**验证流程**：
1. 从 `X-Refresh-Token` Header 中提取 Refresh Token
2. 调用 `jwtManager.VerifyRefreshToken()` 验证 Token
3. 验证 Session 是否存在
4. 生成新的 Access Token 和 Refresh Token

### Token 传输方式

#### Access Token 传输

Access Token 通过 `tokenCarrier`（Token 载体）传输，支持两种方式：

**方式 1：HTTP Header（默认）**

```go
// 使用 Header 传输（默认使用 "Authorization" Header）
carrier := header.NewCarrier()
// 或自定义 Header 名称
carrier := header.NewCarrierWithHeader("X-Access-Token")
```

客户端请求：
```javascript
fetch('/profile', {
    headers: {
        'Authorization': accessToken  // 或 'X-Access-Token': accessToken
    }
})
```

**方式 2：Cookie**

```go
// 使用 Cookie 传输
carrier := cookie.NewCarrier("gint_token",
    cookie.WithHttpOnly(true),  // 防止 XSS
    cookie.WithSecure(true),    // 仅 HTTPS
)
```

客户端请求：
```javascript
// Cookie 会自动随请求发送，无需手动设置
fetch('/profile', {
    credentials: 'include'  // 确保发送 Cookie
})
```

#### Refresh Token 传输

Refresh Token **固定通过 `X-Refresh-Token` Header 传输**，不支持其他方式：

```javascript
fetch('/refresh', {
    method: 'POST',
    headers: {
        'X-Refresh-Token': refreshToken  // 固定使用此 Header
    }
})
```

### 验证失败处理

如果 Token 验证失败，gint 会自动处理：

- **Access Token 验证失败**：`gint.S()` 和 `gint.BS()` 会自动返回 `401 Unauthorized`
- **Refresh Token 验证失败**：`RenewToken()` 会返回错误，需要手动处理

**示例**：

```go
// 使用 gint.S() 时，验证失败会自动返回 401
r.GET("/profile", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
    // 如果 Access Token 无效或过期，不会执行到这里
    // 而是自动返回 401 Unauthorized
    return gint.Result{Code: 0, Data: sess.Claims().UserId}, nil
}))

// 刷新 Token 时，需要手动处理错误
r.POST("/refresh", func(c *gin.Context) {
    ctx := &gctx.Context{Context: c}
    if err := session.GetDefaultProvider().RenewToken(ctx); err != nil {
        // Refresh Token 验证失败
        c.JSON(401, gin.H{"code": 401, "msg": "刷新失败，请重新登录"})
        return
    }
    c.JSON(200, gin.H{"code": 0, "msg": "刷新成功"})
})
```

## 基本用法

### 初始化配置

```go
import (
    "time"
    "github.com/redis/go-redis/v9"
    "github.com/ink-yht-code/gint/session"
    redisSession "github.com/ink-yht-code/gint/session/redis"
    "github.com/ink-yht-code/gint/session/header"
)

func main() {
    r := gin.Default()
    
    // 创建 Redis 客户端
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    // 创建 Session Provider（双 Token 配置）
    provider := redisSession.NewProvider(
        rdb,
        "your-jwt-secret-key",
        time.Minute * 30,  // Access Token 过期时间：30 分钟
        time.Hour * 24 * 7, // Refresh Token 过期时间：7 天
        header.NewCarrier(),
    )
    
    session.SetDefaultProvider(provider)
    
    r.Run(":8080")
}
```

### 推荐的过期时间

| Token 类型 | 推荐时间 | 说明 |
|-----------|---------|------|
| Access Token | 15 分钟 - 2 小时 | 短期有效，安全性高 |
| Refresh Token | 7 天 - 30 天 | 长期有效，减少登录次数 |

```go
// 高安全场景（如金融应用）
accessExpire := time.Minute * 15  // 15 分钟
refreshExpire := time.Hour * 24 * 7 // 7 天

// 普通场景
accessExpire := time.Hour * 2      // 2 小时
refreshExpire := time.Hour * 24 * 14 // 14 天

// 低安全场景（如内部系统）
accessExpire := time.Hour * 24     // 24 小时
refreshExpire := time.Hour * 24 * 30 // 30 天
```

## 登录流程

### 服务端实现

```go
type LoginReq struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

r.POST("/login", gint.B(func(ctx *gctx.Context, req LoginReq) (gint.Result, error) {
    // 1. 验证用户名密码
    user, err := validateUser(req.Username, req.Password)
    if err != nil {
        return gint.Result{Code: 401, Msg: "用户名或密码错误"}, nil
    }
    
    // 2. 创建 Session（自动生成双 Token）
    sess, err := session.NewSession(ctx, user.ID,
        // JWT 数据（轻量、不变）
        map[string]string{
            "role": user.Role,
            "dept": user.Department,
        },
        // Session 数据（完整、可变）
        map[string]any{
            "login_time": time.Now().Unix(),
            "login_ip":   ctx.ClientIP(),
        },
    )
    if err != nil {
        return gint.Result{Code: 500, Msg: "创建会话失败"}, err
    }
    
    // 3. 返回成功（Token 已自动注入到响应头）
    return gint.Result{
        Code: 0,
        Msg:  "登录成功",
        Data: map[string]any{
            "user_id": sess.Claims().UserId,
            "role":    sess.Claims().Data["role"],
        },
    }, nil
}))
```

### 响应头

登录成功后，服务端会自动在响应头中注入两个 Token：

```
Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...  (Access Token)
X-Refresh-Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...  (Refresh Token)
```

### 客户端处理

```javascript
// 登录请求
fetch('/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: 'admin', password: '123456' })
})
.then(response => {
    // 1. 从响应头中提取 Token
    const accessToken = response.headers.get('Authorization');
    const refreshToken = response.headers.get('X-Refresh-Token');
    
    // 2. 存储 Token
    localStorage.setItem('access_token', accessToken);
    localStorage.setItem('refresh_token', refreshToken);
    
    return response.json();
})
.then(data => {
    console.log('登录成功', data);
});
```

## 使用 Access Token 访问接口

### 服务端实现

使用 `gint.S()` 或 `gint.BS()` 包装器时，**Access Token 验证会自动完成**：

```go
r.GET("/profile", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
    // ✅ Access Token 已自动验证
    // ✅ 如果 Token 无效或过期，会自动返回 401，不会执行到这里
    // ✅ 可以直接从 Token 中获取用户 ID，无需额外查询
    userId := sess.Claims().UserId
    role := sess.Claims().Data["role"]
    
    return gint.Result{
        Code: 0,
        Data: map[string]any{
            "user_id": userId,
            "role":    role,
        },
    }, nil
}))
```

**验证过程**（自动完成，无需手动处理）：
1. 从请求中提取 Access Token（通过 `tokenCarrier`，默认是 `Authorization` Header）
2. 验证 Token 签名和过期时间
3. 验证 Session 是否存在（Redis 或内存）
4. 如果验证失败，自动返回 `401 Unauthorized`
5. 如果验证成功，将 Session 注入到函数参数中

### 客户端请求

**使用 Header 传输 Access Token**（默认方式）：

```javascript
// 使用 Access Token 访问接口
const accessToken = localStorage.getItem('access_token');

fetch('/profile', {
    headers: {
        'Authorization': accessToken  // 默认使用 Authorization Header
    }
})
.then(response => {
    if (response.status === 401) {
        // Access Token 过期或无效，需要刷新
        return refreshAccessToken().then(() => {
            // 刷新成功后重试
            return fetch('/profile', {
                headers: {
                    'Authorization': localStorage.getItem('access_token')
                }
            });
        });
    }
    return response.json();
})
.then(data => {
    console.log('用户信息', data);
})
.catch(error => {
    console.error('请求失败', error);
});
```

**使用 Cookie 传输 Access Token**（如果配置了 Cookie Carrier）：

```javascript
// Cookie 会自动随请求发送，无需手动设置
fetch('/profile', {
    credentials: 'include'  // 确保发送 Cookie
})
.then(response => response.json())
.then(data => {
    console.log('用户信息', data);
});
```

## 刷新 Access Token

当 Access Token 过期时，使用 Refresh Token 获取新的 Token 对。

### 服务端实现

```go
r.POST("/refresh", func(c *gin.Context) {
    ctx := &gctx.Context{Context: c}
    
    // 验证 Refresh Token 并生成新的 Token 对
    // RenewToken 会自动：
    // 1. 从 X-Refresh-Token Header 中提取 Refresh Token
    // 2. 验证 Refresh Token 的有效性
    // 3. 验证 Session 是否存在
    // 4. 生成新的 Access Token 和 Refresh Token
    // 5. 将新 Token 注入到响应头中
    if err := session.GetDefaultProvider().RenewToken(ctx); err != nil {
        // Refresh Token 验证失败（过期、无效或 Session 不存在）
        c.JSON(401, gin.H{"code": 401, "msg": "刷新失败，请重新登录"})
        return
    }
    
    // 刷新成功，新的 Token 已自动注入到响应头中
    c.JSON(200, gin.H{"code": 0, "msg": "刷新成功"})
})
```

**验证过程**：
1. 从 `X-Refresh-Token` Header 中提取 Refresh Token
2. 调用 `jwtManager.VerifyRefreshToken()` 验证 Token 签名和过期时间
3. 验证 Session 是否存在（Redis 或内存）
4. 如果验证失败，返回错误
5. 如果验证成功，生成新的 Access Token 和 Refresh Token，并注入到响应头中

### 客户端实现

```javascript
async function refreshAccessToken() {
    const refreshToken = localStorage.getItem('refresh_token');
    
    const response = await fetch('/refresh', {
        method: 'POST',
        headers: {
            'X-Refresh-Token': refreshToken
        }
    });
    
    if (response.ok) {
        // 1. 从响应头中提取新的 Token
        const newAccessToken = response.headers.get('Authorization');
        const newRefreshToken = response.headers.get('X-Refresh-Token');
        
        // 2. 更新存储的 Token
        localStorage.setItem('access_token', newAccessToken);
        localStorage.setItem('refresh_token', newRefreshToken);
        
        return true;
    } else {
        // Refresh Token 也过期了，需要重新登录
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        window.location.href = '/login';
        return false;
    }
}
```

### 无感刷新拦截器（推荐）

**核心思路**：拦截 401 错误 → 自动刷新 Token → 重试原请求

```javascript
// Axios 拦截器示例（实现无感刷新）
axios.interceptors.response.use(
    response => response,
    async error => {
        const originalRequest = error.config;
        
        // 如果是 401 错误且未重试过
        if (error.response.status === 401 && !originalRequest._retry) {
            originalRequest._retry = true;
            
            // 🔄 自动刷新 Token（用户无感知）
            const success = await refreshAccessToken();
            
            if (success) {
                // ✅ 使用新的 Access Token 重试原请求
                const newAccessToken = localStorage.getItem('access_token');
                originalRequest.headers['Authorization'] = newAccessToken;
                return axios(originalRequest);
            } else {
                // ❌ Refresh Token 也过期了，跳转登录
                window.location.href = '/login';
            }
        }
        
        return Promise.reject(error);
    }
);
```

**效果**：
- ✅ 用户点击按钮 → 请求发送 → Token 过期 → 自动刷新 → 请求成功
- ✅ 用户完全不知道 Token 过期了
- ✅ 无需手动处理 Token 刷新逻辑

## 退出登录

```go
r.POST("/logout", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
    // 销毁 Session（同时清除 Access Token 和 Refresh Token）
    if err := sess.Destroy(ctx); err != nil {
        return gint.Result{Code: 500, Msg: "退出失败"}, err
    }
    
    return gint.Result{Code: 0, Msg: "退出成功"}, nil
}))
```

## 完整示例

### 服务端

```go
package main

import (
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/redis/go-redis/v9"
    "github.com/ink-yht-code/gint"
    "github.com/ink-yht-code/gint/gctx"
    "github.com/ink-yht-code/gint/session"
    redisSession "github.com/ink-yht-code/gint/session/redis"
    "github.com/ink-yht-code/gint/session/header"
)

type LoginReq struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

func main() {
    r := gin.Default()
    
    // 初始化 Session（双 Token）
    rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
    provider := redisSession.NewProvider(
        rdb,
        "your-secret-key",
        time.Hour * 2,      // Access Token: 2 小时
        time.Hour * 24 * 7, // Refresh Token: 7 天
        header.NewCarrier(),
    )
    session.SetDefaultProvider(provider)
    
    // 登录
    r.POST("/login", gint.B(func(ctx *gctx.Context, req LoginReq) (gint.Result, error) {
        if req.Username != "admin" || req.Password != "123456" {
            return gint.Result{Code: 401, Msg: "用户名或密码错误"}, nil
        }
        
        sess, err := session.NewSession(ctx, "user_123",
            map[string]string{"role": "admin"},
            map[string]any{"login_time": time.Now().Unix()},
        )
        if err != nil {
            return gint.Result{Code: 500}, err
        }
        
        return gint.Result{
            Code: 0,
            Msg:  "登录成功",
            Data: map[string]any{
                "user_id": sess.Claims().UserId,
            },
        }, nil
    }))
    
    // 获取用户信息（需要 Access Token）
    r.GET("/profile", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
        return gint.Result{
            Code: 0,
            Data: map[string]any{
                "user_id": sess.Claims().UserId,
                "role":    sess.Claims().Data["role"],
            },
        }, nil
    }))
    
    // 刷新 Token（需要 Refresh Token）
    r.POST("/refresh", func(c *gin.Context) {
        ctx := &gctx.Context{Context: c}
        if err := provider.RenewToken(ctx); err != nil {
            c.JSON(401, gin.H{"code": 401, "msg": "刷新失败"})
            return
        }
        c.JSON(200, gin.H{"code": 0, "msg": "刷新成功"})
    })
    
    // 退出登录
    r.POST("/logout", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
        if err := sess.Destroy(ctx); err != nil {
            return gint.Result{Code: 500}, err
        }
        return gint.Result{Code: 0, Msg: "退出成功"}, nil
    }))
    
    r.Run(":8080")
}
```

### 客户端（完整）

```javascript
class AuthService {
    constructor() {
        this.accessToken = localStorage.getItem('access_token');
        this.refreshToken = localStorage.getItem('refresh_token');
    }
    
    // 登录
    async login(username, password) {
        const response = await fetch('/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });
        
        if (response.ok) {
            this.accessToken = response.headers.get('Authorization');
            this.refreshToken = response.headers.get('X-Refresh-Token');
            
            localStorage.setItem('access_token', this.accessToken);
            localStorage.setItem('refresh_token', this.refreshToken);
            
            return await response.json();
        }
        
        throw new Error('登录失败');
    }
    
    // 刷新 Token
    async refreshToken() {
        const response = await fetch('/refresh', {
            method: 'POST',
            headers: {
                'X-Refresh-Token': this.refreshToken
            }
        });
        
        if (response.ok) {
            this.accessToken = response.headers.get('Authorization');
            this.refreshToken = response.headers.get('X-Refresh-Token');
            
            localStorage.setItem('access_token', this.accessToken);
            localStorage.setItem('refresh_token', this.refreshToken);
            
            return true;
        }
        
        return false;
    }
    
    // 发送请求（自动处理 Token 刷新）
    async request(url, options = {}) {
        options.headers = {
            ...options.headers,
            'Authorization': this.accessToken
        };
        
        let response = await fetch(url, options);
        
        // 如果 401，尝试刷新 Token
        if (response.status === 401) {
            const refreshed = await this.refreshToken();
            
            if (refreshed) {
                // 使用新 Token 重试
                options.headers['Authorization'] = this.accessToken;
                response = await fetch(url, options);
            } else {
                // 刷新失败，跳转登录
                window.location.href = '/login';
                throw new Error('需要重新登录');
            }
        }
        
        return response.json();
    }
    
    // 退出登录
    async logout() {
        await this.request('/logout', { method: 'POST' });
        
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        
        this.accessToken = null;
        this.refreshToken = null;
    }
}

// 使用示例
const auth = new AuthService();

// 登录
await auth.login('admin', '123456');

// 访问接口（自动处理 Token 刷新）
const profile = await auth.request('/profile');
console.log(profile);

// 退出登录
await auth.logout();
```

## 安全建议

### 1. Token 存储

```javascript
// ✅ 推荐：使用 HttpOnly Cookie（服务端设置）
// 客户端无法通过 JavaScript 访问，防止 XSS 攻击

// ⚠️ 可接受：使用 localStorage
// 方便但有 XSS 风险，需要做好 XSS 防护

// ❌ 不推荐：使用普通 Cookie
// 容易受到 CSRF 攻击
```

### 2. HTTPS

```go
// 生产环境必须使用 HTTPS
if os.Getenv("ENV") == "production" {
    carrier := cookie.NewCarrier("session_token",
        cookie.WithSecure(true),  // 仅 HTTPS 传输
        cookie.WithHttpOnly(true), // 防止 XSS
    )
}
```

### 3. Refresh Token 轮换

每次刷新时生成新的 Refresh Token，旧的立即失效：

```go
// gint 已自动实现：每次调用 RenewToken 都会生成新的 Refresh Token
```

### 4. 限制刷新频率

```go
import "github.com/ink-yht-code/gint/middlewares/ratelimit"

// 限制刷新接口的调用频率
limiter := ratelimit.NewSimpleLimiter(10, time.Minute)
r.POST("/refresh", 
    ratelimit.NewBuilder(limiter).WithIPKey().Build(),
    refreshHandler,
)
```

## 常见问题

### Q: Token 验证是在哪里完成的？

A: **Token 验证在 gint 项目内部自动完成**，引用项目无需手动实现验证逻辑。

- 使用 `gint.S()` 或 `gint.BS()` 时，会自动验证 Access Token
- 调用 `RenewToken()` 时，会自动验证 Refresh Token
- 验证失败会自动返回 401 Unauthorized

引用项目只需要：
1. 初始化 Session Provider
2. 使用 `gint.S()` 或 `gint.BS()` 包装需要认证的接口
3. 在客户端正确携带 Token 即可

### Q: 有几个 Token？验证的是哪个？

A: **共有 2 个 Token**：

1. **Access Token** - 访问受保护接口时验证
   - 通过 `tokenCarrier` 传输（默认是 `Authorization` Header 或 Cookie）
   - 使用 `gint.S()` 或 `gint.BS()` 时自动验证

2. **Refresh Token** - 刷新 Token 时验证
   - 固定通过 `X-Refresh-Token` Header 传输
   - 调用 `RenewToken()` 时自动验证

### Q: Access Token 和 Refresh Token 有什么区别？

A:
- **Access Token**: 短期有效（15分钟-2小时），用于访问 API，每次请求都需要携带
- **Refresh Token**: 长期有效（7天-30天），用于获取新的 Access Token，仅在刷新时使用

### Q: 为什么要把 UserId 放在 Token 中？

A: 
- 减少数据库查询，提高性能
- Token 自包含用户信息，无需额外请求
- 方便在中间件中获取用户 ID

### Q: Refresh Token 泄露了怎么办？

A:
- 立即调用退出接口销毁 Session
- Refresh Token 会随 Session 一起失效
- 用户需要重新登录

### Q: 如何实现"记住我"功能？

A:
```go
// 记住我：Refresh Token 设置更长的过期时间
if req.RememberMe {
    refreshExpire = time.Hour * 24 * 30 // 30 天
} else {
    refreshExpire = time.Hour * 24 * 7  // 7 天
}
```

## 总结

双 Token 机制提供了安全性和用户体验的最佳平衡：

- ✅ **Access Token 短期有效** - 即使泄露影响也小
- ✅ **Refresh Token 长期有效** - 用户无需频繁登录
- ✅ **Token 包含 UserId** - 减少数据库查询
- ✅ **自动刷新机制** - 无感知更新 Token
- ✅ **灵活的过期策略** - 根据业务需求调整
- ✅ **自动验证机制** - Token 验证在 gint 内部自动完成，引用项目无需手动实现
- ✅ **灵活的传输方式** - 支持 Header 和 Cookie 两种方式传输 Access Token

**核心优势**：
- Token 验证完全自动化，使用 `gint.S()` 或 `gint.BS()` 即可
- 验证失败自动返回 401，无需手动处理
- 支持 Header 和 Cookie 两种传输方式，适应不同场景

gint 的双 Token 实现简单易用，开箱即用，是构建安全 API 的理想选择！

