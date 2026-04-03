# CSRF防护

gint 提供 CSRF（跨站请求伪造）防护中间件，保护应用免受 CSRF 攻击。

## 概述

CSRF防护特性：
- Token验证机制
- 双重提交Cookie
- 自定义Token生成
- 支持多种参数来源

## 基本使用

### 默认配置

```go
import "github.com/ink-yht-code/gint/middlewares/csrf"

r.Use(csrf.Default())
```

### 自定义配置

```go
r.Use(csrf.New(csrf.Config{
    Secret:           "your-secret-key",
    TokenLength:      32,
    TokenLookup:      "header:X-CSRF-Token",
    ErrorFunc:        nil,
    Skipper:          nil,
}))
```

## 配置选项

```go
type Config struct {
    // Secret CSRF密钥
    Secret string
    
    // TokenLength Token长度
    TokenLength int
    
    // TokenLookup Token查找位置
    // 支持: header:X-CSRF-Token, form:_csrf, query:csrf_token
    TokenLookup string
    
    // ErrorFunc 错误处理函数
    ErrorFunc func(c *gin.Context, err error)
    
    // Skipper 跳过中间件的条件
    Skipper func(c *gin.Context) bool
}
```

## Token查找位置

### Header中查找

```go
csrf.New(csrf.Config{
    TokenLookup: "header:X-CSRF-Token",
})
```

客户端请求：
```bash
curl -H "X-CSRF-Token: token_value" http://localhost:8080/api/users
```

### Form中查找

```go
csrf.New(csrf.Config{
    TokenLookup: "form:_csrf",
})
```

HTML表单：
```html
<form method="POST" action="/api/users">
    <input type="hidden" name="_csrf" value="token_value">
    <input type="text" name="name">
    <button type="submit">提交</button>
</form>
```

### Query中查找

```go
csrf.New(csrf.Config{
    TokenLookup: "query:csrf_token",
})
```

客户端请求：
```bash
curl "http://localhost:8080/api/users?csrf_token=token_value" -X POST
```

## 工作流程

### 1. 获取Token

```go
// GET请求获取Token
r.GET("/csrf-token", func(c *gin.Context) {
    token := csrf.GenerateToken(c)
    c.JSON(200, gin.H{
        "token": token,
    })
})
```

### 2. 客户端存储Token

```javascript
// 获取Token
fetch('/csrf-token')
    .then(res => res.json())
    .then(data => {
        localStorage.setItem('csrf_token', data.token);
    });
```

### 3. 发送请求时携带Token

```javascript
// 发送POST请求
fetch('/api/users', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': localStorage.getItem('csrf_token'),
    },
    body: JSON.stringify({
        name: 'John',
        email: 'john@example.com',
    }),
});
```

### 4. 服务器验证Token

中间件自动验证Token，验证失败返回403错误。

## 安全配置

### 1. 使用强密钥

```go
import "crypto/rand"
import "encoding/hex"

func generateSecret() string {
    b := make([]byte, 32)
    rand.Read(b)
    return hex.EncodeToString(b)
}

r.Use(csrf.New(csrf.Config{
    Secret: generateSecret(),
}))
```

### 2. 跳过特定路由

```go
r.Use(csrf.New(csrf.Config{
    Secret: "your-secret-key",
    Skipper: func(c *gin.Context) bool {
        // 跳过GET请求
        if c.Request.Method == "GET" {
            return true
        }
        
        // 跳过特定路由
        if c.FullPath() == "/api/public" {
            return true
        }
        
        // 跳过特定前缀
        if strings.HasPrefix(c.FullPath(), "/webhook") {
            return true
        }
        
        return false
    },
}))
```

### 3. 自定义错误处理

```go
r.Use(csrf.New(csrf.Config{
    Secret: "your-secret-key",
    ErrorFunc: func(c *gin.Context, err error) {
        c.JSON(403, gin.H{
            "code": 403,
            "msg":  "CSRF验证失败",
            "data": nil,
        })
        c.Abort()
    },
}))
```

## 与SameSite Cookie配合

### 设置SameSite属性

```go
r.Use(func(c *gin.Context) {
    // 设置SameSite=Strict
    c.SetSameSite(http.SameSiteLaxMode)
    c.Next()
})

r.Use(csrf.Default())
```

### SameSite选项

| 选项 | 说明 | 安全性 |
|------|------|--------|
| `SameSiteStrictMode` | 严格模式，不发送跨站Cookie | 最高 |
| `SameSiteLaxMode` | 宽松模式，仅顶级导航发送 | 中等 |
| `SameSiteNoneMode` | 无限制，需要Secure标志 | 最低 |

## 完整示例

### 后端实现

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/ink-yht-code/gint"
    "github.com/ink-yht-code/gint/middlewares/csrf"
)

func main() {
    r := gin.Default()
    
    // CSRF防护
    r.Use(csrf.New(csrf.Config{
        Secret:      "your-secret-key",
        TokenLength: 32,
        TokenLookup: "header:X-CSRF-Token",
    }))
    
    // 获取CSRF Token
    r.GET("/csrf-token", func(c *gin.Context) {
        token := csrf.GenerateToken(c)
        c.JSON(200, gin.H{
            "code": 0,
            "data": gin.H{
                "token": token,
            },
        })
    })
    
    // 创建用户
    type CreateUserReq struct {
        Name  string `json:"name" validate:"required"`
        Email string `json:"email" validate:"required,email"`
    }
    
    r.POST("/api/users", gint.B(func(c *gin.Context, req CreateUserReq) (gint.Result, error) {
        // CSRF验证已由中间件完成
        user := &User{
            Name:  req.Name,
            Email: req.Email,
        }
        
        if err := db.Create(user); err != nil {
            return gint.Result{}, err
        }
        
        return gint.Result{Data: user}, nil
    }))
    
    r.Run(":8080")
}
```

### 前端实现

```html
<!DOCTYPE html>
<html>
<head>
    <title>CSRF防护示例</title>
</head>
<body>
    <h1>创建用户</h1>
    <form id="userForm">
        <input type="text" id="name" placeholder="用户名" required>
        <input type="email" id="email" placeholder="邮箱" required>
        <button type="submit">创建</button>
    </form>

    <script>
        // 初始化：获取CSRF Token
        async function initCSRFToken() {
            const res = await fetch('/csrf-token');
            const data = await res.json();
            localStorage.setItem('csrf_token', data.data.token);
        }

        // 提交表单
        document.getElementById('userForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const csrfToken = localStorage.getItem('csrf_token');
            
            const res = await fetch('/api/users', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-CSRF-Token': csrfToken,
                },
                body: JSON.stringify({
                    name: document.getElementById('name').value,
                    email: document.getElementById('email').value,
                }),
            });
            
            const data = await res.json();
            if (data.code === 0) {
                alert('创建成功');
                document.getElementById('userForm').reset();
            } else {
                alert('创建失败: ' + data.msg);
            }
        });

        // 页面加载时初始化
        initCSRFToken();
    </script>
</body>
</html>
```

## 最佳实践

### 1. 定期更新Token

```javascript
// 每次页面加载时获取新Token
async function refreshCSRFToken() {
    const res = await fetch('/csrf-token');
    const data = await res.json();
    localStorage.setItem('csrf_token', data.data.token);
}

// 页面加载时
window.addEventListener('load', refreshCSRFToken);

// 定期刷新（可选）
setInterval(refreshCSRFToken, 3600000); // 每小时
```

### 2. 使用HTTPS

```go
// 生产环境必须使用HTTPS
r.Use(func(c *gin.Context) {
    if c.Request.Header.Get("X-Forwarded-Proto") != "https" {
        c.Redirect(301, "https://"+c.Request.Host+c.Request.RequestURI)
        c.Abort()
        return
    }
    c.Next()
})
```

### 3. 结合其他安全措施

```go
r.Use(func(c *gin.Context) {
    // X-Frame-Options - 防止点击劫持
    c.Header("X-Frame-Options", "DENY")
    
    // X-Content-Type-Options - 防止MIME嗅探
    c.Header("X-Content-Type-Options", "nosniff")
    
    // X-XSS-Protection - XSS防护
    c.Header("X-XSS-Protection", "1; mode=block")
    
    // Content-Security-Policy - 内容安全策略
    c.Header("Content-Security-Policy", "default-src 'self'")
    
    c.Next()
})

r.Use(csrf.Default())
```

### 4. 异常处理

```go
r.Use(csrf.New(csrf.Config{
    Secret: "your-secret-key",
    ErrorFunc: func(c *gin.Context, err error) {
        // 记录日志
        log.Printf("CSRF验证失败: %v", err)
        
        // 返回错误响应
        c.JSON(403, gin.H{
            "code": 403,
            "msg":  "请求验证失败，请重试",
            "data": nil,
        })
        c.Abort()
    },
}))
```

## 常见问题

### Q: Token过期了怎么办？
A: 可以在客户端捕获403错误，重新获取Token后重试请求。

### Q: 如何在AJAX请求中自动添加Token？
A: 使用拦截器在所有请求中自动添加Token头。

```javascript
// 使用Axios
axios.interceptors.request.use(config => {
    config.headers['X-CSRF-Token'] = localStorage.getItem('csrf_token');
    return config;
});
```

### Q: 是否需要在GET请求中验证Token？
A: 不需要，CSRF防护通常只对修改操作（POST/PUT/DELETE）进行验证。

### Q: 多个域名如何共享Token？
A: 不建议跨域共享Token，应该为每个域名单独生成Token。
