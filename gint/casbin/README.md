# Casbin 权限管理快速指南

> 📖 **详细文档**：请查看 [Casbin权限管理.md](../docs/Casbin权限管理.md)

## 快速开始

### 1. 使用 GORM 定义模型

```go
type Permission struct {
    ID       uint   `gorm:"primaryKey"`
    Role     string `gorm:"type:varchar(50);not null;index"`
    Resource string `gorm:"type:varchar(255);not null;index"`
    Action   string `gorm:"type:varchar(10);not null"`
    CreatedAt time.Time
}

type UserRole struct {
    ID       uint   `gorm:"primaryKey"`
    UserID   string `gorm:"type:varchar(50);not null;index"`
    Role     string `gorm:"type:varchar(50);not null;index"`
    CreatedAt time.Time
}
```

### 2. 实现 PolicyProvider 接口

```go
type PolicyProvider struct {
    db *gorm.DB
}

func (p *PolicyProvider) LoadPolicies(ctx context.Context) ([]string, error) {
    var permissions []Permission
    if err := p.db.WithContext(ctx).Find(&permissions).Error; err != nil {
        return nil, err
    }

    policies := make([]string, 0, len(permissions))
    for _, perm := range permissions {
        policies = append(policies, fmt.Sprintf("p, %s, %s, %s", 
            perm.Role, perm.Resource, perm.Action))
    }
    return policies, nil
}

func (p *PolicyProvider) LoadRolePolicies(ctx context.Context) ([]string, error) {
    var userRoles []UserRole
    if err := p.db.WithContext(ctx).Find(&userRoles).Error; err != nil {
        return nil, err
    }

    rolePolicies := make([]string, 0, len(userRoles))
    for _, ur := range userRoles {
        rolePolicies = append(rolePolicies, fmt.Sprintf("g, %s, %s", 
            ur.UserID, ur.Role))
    }
    return rolePolicies, nil
}
```

### 3. 初始化并使用

```go
// 创建管理器
policyProvider := &PolicyProvider{db: db}
manager, _ := casbin.NewManager(casbin.Options{
    PolicyProvider: policyProvider,
    CacheEnabled:  true,
})

// 创建中间件
casbinMiddleware := casbin.NewBuilder(manager).Build()

// 应用到路由
r.Use(casbinMiddleware)
```

## 重要说明

### ✅ 使用 GORM 完全没有影响

无论你使用：
- GORM
- Ent
- XORM
- 原生 SQL

只要实现了 `PolicyProvider` 接口，就可以正常工作。这正是接口抽象设计的优势！

### 📚 完整文档

详细的使用说明、集成步骤、最佳实践等，请查看：
- **[Casbin权限管理.md](../docs/Casbin权限管理.md)** - 完整的使用指南

## 核心概念

- **策略格式**：`p, 角色, 资源, 操作`，如 `p, admin, /api/users, GET`
- **角色继承**：`g, 用户, 角色`，如 `g, user123, admin`
- **资源匹配**：默认格式 `路径:方法`，如 `/api/users:GET`

## 常见问题

**Q: 使用 GORM 存储策略会有影响吗？**  
A: 完全没有影响！这正是接口抽象设计的优势。

**Q: 策略什么时候加载？**  
A: 启动时自动加载一次，之后可以手动调用 `RefreshPolicies()` 刷新。

**Q: 如何提高性能？**  
A: 启用缓存（`CacheEnabled: true`），角色信息存储在 Session Claims 中。

更多问题请查看 [详细文档](../docs/Casbin权限管理.md#常见问题)。
