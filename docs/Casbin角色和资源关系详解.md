# Casbin 角色和资源关系详解

## 概述

本文档详细解释 gint 中 Casbin 集成的角色和资源关系，包括：
- 一个用户可以有几个角色？
- 一个用户有多个角色会如何？
- 一个角色可以有多个资源吗？

## 核心关系图

```
用户 (User)
  │
  ├─→ 可以有多个角色 (多对多关系)
  │   ├─→ 角色1: admin
  │   ├─→ 角色2: user
  │   └─→ 角色3: editor
  │
角色 (Role)
  │
  ├─→ 可以有多个资源权限 (一对多关系)
  │   ├─→ 资源1: /api/users (GET, POST, DELETE)
  │   ├─→ 资源2: /api/articles (GET, POST)
  │   └─→ 资源3: /api/comments (GET)
```

## 1. 一个用户可以有几个角色？

### 答案：可以有多个角色（多对多关系）

**设计支持**：
- ✅ 一个用户可以拥有多个角色
- ✅ 数据库设计支持多对多关系
- ✅ 代码实现支持角色列表

### 数据库设计

```sql
-- 用户角色表（支持一个用户多个角色）
CREATE TABLE user_roles (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id VARCHAR(50) NOT NULL,  -- 用户 ID
    role VARCHAR(50) NOT NULL,     -- 角色名称
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_role (user_id, role),  -- 防止重复
    INDEX idx_user_id (user_id),
    INDEX idx_role (role)
);
```

**数据示例**：

```sql
-- 用户 user123 有 3 个角色
INSERT INTO user_roles (user_id, role) VALUES
('user123', 'admin'),   -- 管理员角色
('user123', 'user'),    -- 普通用户角色
('user123', 'editor');  -- 编辑角色
```

### 代码实现

从 `casbin/middleware.go` 可以看到，角色是以**数组**形式存储的：

```go
// 获取用户角色
var roles []string  // 支持多个角色

if b.opts.UserRoleProvider != nil {
    // 从 UserRoleProvider 获取角色列表
    roles, err = b.opts.UserRoleProvider.GetUserRoles(ctx, userId)
    // 返回：["admin", "user", "editor"]
} else {
    // 从 Session Claims 中获取角色
    if roleStr, ok := sess.Claims().Data["roles"]; ok {
        roles = parseRoles(roleStr)  // 解析逗号分隔的字符串
        // "admin,user,editor" -> ["admin", "user", "editor"]
    }
}
```

### 在登录时设置多个角色

```go
// 登录时，获取用户的所有角色
var userRoles []UserRole
db.Where("user_id = ?", user.ID).Find(&userRoles)

// 提取角色列表
roles := make([]string, 0, len(userRoles))
for _, ur := range userRoles {
    roles = append(roles, ur.Role)
}
// roles = ["admin", "user", "editor"]

// 创建 Session，将多个角色放入 JWT Data
sess, err := session.NewSession(ctx, user.ID,
    map[string]string{
        "roles": strings.Join(roles, ","), // "admin,user,editor"
    },
    map[string]any{},
)
```

## 2. 一个用户有多个角色会如何？

### 权限检查逻辑：**只要有一个角色有权限，就允许访问**

从代码实现可以看到：

```go
// 检查权限（检查用户是否有权限访问该资源）
allowed := false
for _, role := range roles {  // 遍历所有角色
    ok, err := b.manager.Enforce(role, resource, action)
    if err != nil {
        // 错误处理...
    }
    if ok {  // 只要有一个角色有权限
        allowed = true
        break  // 立即退出，不需要检查其他角色
    }
}
```

### 工作原理

**示例场景**：

假设用户 `user123` 有 3 个角色：`admin`、`user`、`editor`

**权限配置**：
```
p, admin, /api/users, GET      # admin 可以查看用户
p, admin, /api/users, POST     # admin 可以创建用户
p, user, /api/profile, GET     # user 可以查看自己的资料
p, editor, /api/articles, GET  # editor 可以查看文章
p, editor, /api/articles, POST # editor 可以创建文章
```

**权限检查过程**：

1. **访问 `/api/users` (GET)**：
   - 检查 `admin` 角色 → ✅ 有权限 → **允许访问**
   - 不需要检查 `user` 和 `editor` 角色

2. **访问 `/api/profile` (GET)**：
   - 检查 `admin` 角色 → ❌ 无权限
   - 检查 `user` 角色 → ✅ 有权限 → **允许访问**
   - 不需要检查 `editor` 角色

3. **访问 `/api/articles` (POST)**：
   - 检查 `admin` 角色 → ❌ 无权限
   - 检查 `user` 角色 → ❌ 无权限
   - 检查 `editor` 角色 → ✅ 有权限 → **允许访问**

4. **访问 `/api/comments` (DELETE)**：
   - 检查 `admin` 角色 → ❌ 无权限
   - 检查 `user` 角色 → ❌ 无权限
   - 检查 `editor` 角色 → ❌ 无权限
   - 所有角色都没有权限 → **拒绝访问（403）**

### 权限合并（OR 逻辑）

**重要**：多个角色的权限是**合并**的，使用 **OR 逻辑**：

- ✅ 如果用户有角色 A 或角色 B，只要其中一个有权限，就可以访问
- ✅ 用户最终拥有的权限 = 所有角色权限的**并集**

**示例**：

```
用户 user123 的角色：
- admin: 可以访问 /api/users, /api/articles
- editor: 可以访问 /api/articles, /api/comments

用户 user123 最终拥有的权限：
- /api/users (来自 admin)
- /api/articles (来自 admin 和 editor)
- /api/comments (来自 editor)
```

### 实际应用场景

**场景1：超级管理员 + 部门管理员**

```go
// 用户同时是超级管理员和部门管理员
userRoles := []string{"super_admin", "dept_admin"}

// 权限配置
// super_admin 可以访问所有资源
// dept_admin 只能访问部门内的资源

// 权限检查时，会检查两个角色，只要有一个有权限就允许
```

**场景2：普通用户 + VIP 用户**

```go
// 用户同时是普通用户和 VIP 用户
userRoles := []string{"user", "vip"}

// 权限配置
// user 可以访问基础功能
// vip 可以访问高级功能

// 用户拥有基础功能 + 高级功能的权限
```

## 3. 一个角色可以有多个资源吗？

### 答案：可以有多个资源（一对多关系）

**设计支持**：
- ✅ 一个角色可以有多个资源权限
- ✅ 数据库设计支持一对多关系
- ✅ 每个资源可以有多个操作（GET、POST、DELETE 等）

### 数据库设计

```sql
-- 权限表（支持一个角色多个资源）
CREATE TABLE permissions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    role VARCHAR(50) NOT NULL,      -- 角色名称
    resource VARCHAR(255) NOT NULL,  -- 资源路径
    action VARCHAR(10) NOT NULL,     -- 操作（GET, POST, DELETE等）
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_role (role),
    INDEX idx_resource (resource),
    INDEX idx_role_resource (role, resource)
);
```

**数据示例**：

```sql
-- admin 角色有多个资源的权限
INSERT INTO permissions (role, resource, action) VALUES
('admin', '/api/users', 'GET'),      -- 可以查看用户
('admin', '/api/users', 'POST'),     -- 可以创建用户
('admin', '/api/users', 'DELETE'),  -- 可以删除用户
('admin', '/api/articles', 'GET'),   -- 可以查看文章
('admin', '/api/articles', 'POST'),  -- 可以创建文章
('admin', '/api/articles', 'DELETE'), -- 可以删除文章
('admin', '/api/comments', 'GET'),   -- 可以查看评论
('admin', '/api/comments', 'DELETE'); -- 可以删除评论
```

### 代码实现

从 `PolicyProvider` 接口可以看到，加载策略时会加载所有权限：

```go
func (p *PolicyProvider) LoadPolicies(ctx context.Context) ([]string, error) {
    var permissions []Permission
    // 查询所有权限（包括同一角色的多个资源）
    if err := p.db.WithContext(ctx).Find(&permissions).Error; err != nil {
        return nil, err
    }

    policies := make([]string, 0, len(permissions))
    for _, perm := range permissions {
        // 每个权限都是一条策略
        policies = append(policies, 
            fmt.Sprintf("p, %s, %s, %s", perm.Role, perm.Resource, perm.Action))
    }

    return policies, nil
}
```

### 权限配置示例

**admin 角色的完整权限**：

```
p, admin, /api/users, GET
p, admin, /api/users, POST
p, admin, /api/users, PUT
p, admin, /api/users, DELETE
p, admin, /api/articles, GET
p, admin, /api/articles, POST
p, admin, /api/articles, PUT
p, admin, /api/articles, DELETE
p, admin, /api/comments, GET
p, admin, /api/comments, DELETE
```

**user 角色的权限**：

```
p, user, /api/profile, GET
p, user, /api/profile, POST
p, user, /api/articles, GET
```

**editor 角色的权限**：

```
p, editor, /api/articles, GET
p, editor, /api/articles, POST
p, editor, /api/articles, PUT
p, editor, /api/comments, GET
p, editor, /api/comments, POST
```

## 完整关系示例

### 数据库数据

```sql
-- 用户角色关系
user_roles:
user_id  | role
---------|--------
user123  | admin
user123  | user
user456  | user
user456  | editor
user789  | editor

-- 角色权限关系
permissions:
role   | resource        | action
-------|-----------------|--------
admin  | /api/users      | GET
admin  | /api/users      | POST
admin  | /api/users      | DELETE
admin  | /api/articles   | GET
user   | /api/profile    | GET
user   | /api/profile    | POST
user   | /api/articles   | GET
editor | /api/articles   | GET
editor | /api/articles   | POST
editor | /api/comments   | GET
```

### 权限检查示例

**用户 user123（角色：admin, user）访问不同资源**：

| 资源 | 操作 | admin 权限 | user 权限 | 结果 |
|------|------|-----------|-----------|------|
| /api/users | GET | ✅ 有 | ❌ 无 | ✅ **允许**（admin 有权限） |
| /api/users | POST | ✅ 有 | ❌ 无 | ✅ **允许**（admin 有权限） |
| /api/profile | GET | ❌ 无 | ✅ 有 | ✅ **允许**（user 有权限） |
| /api/articles | GET | ✅ 有 | ✅ 有 | ✅ **允许**（两个都有权限） |
| /api/comments | GET | ❌ 无 | ❌ 无 | ❌ **拒绝**（都没有权限） |

**用户 user456（角色：user, editor）访问不同资源**：

| 资源 | 操作 | user 权限 | editor 权限 | 结果 |
|------|------|-----------|-------------|------|
| /api/users | GET | ❌ 无 | ❌ 无 | ❌ **拒绝** |
| /api/profile | GET | ✅ 有 | ❌ 无 | ✅ **允许**（user 有权限） |
| /api/articles | GET | ✅ 有 | ✅ 有 | ✅ **允许**（两个都有权限） |
| /api/articles | POST | ❌ 无 | ✅ 有 | ✅ **允许**（editor 有权限） |
| /api/comments | GET | ❌ 无 | ✅ 有 | ✅ **允许**（editor 有权限） |

## 总结

### 关系总结表

| 关系类型 | 支持情况 | 说明 |
|---------|---------|------|
| **用户 ↔ 角色** | ✅ 多对多 | 一个用户可以有多角色，一个角色可以分配给多用户 |
| **角色 ↔ 资源** | ✅ 一对多 | 一个角色可以有多个资源权限 |
| **资源 ↔ 操作** | ✅ 一对多 | 一个资源可以有多个操作（GET、POST 等） |

### 权限检查逻辑

1. **获取用户的所有角色**（从 Session 或 UserRoleProvider）
2. **遍历每个角色**，检查是否有权限
3. **只要有一个角色有权限**，就允许访问（OR 逻辑）
4. **如果所有角色都没有权限**，拒绝访问（403）

### 设计优势

- ✅ **灵活性高**：支持复杂的权限组合
- ✅ **易于扩展**：可以随时添加新角色和权限
- ✅ **权限合并**：多个角色的权限自动合并
- ✅ **性能优化**：只要找到一个有权限的角色就立即返回

### 使用建议

1. **角色设计**：
   - 建议设计基础角色（如 user、admin）
   - 可以设计功能角色（如 editor、viewer）
   - 用户可以有多个角色，权限自动合并

2. **权限配置**：
   - 一个角色可以有多个资源权限
   - 每个资源可以有多个操作
   - 使用数据库索引提高查询性能

3. **权限检查**：
   - 权限检查是 OR 逻辑，只要有一个角色有权限就允许
   - 如果所有角色都没有权限，才拒绝访问

