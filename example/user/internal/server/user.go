package server

import (
	"context"

	"example/user/internal/domain/errs"
)

// UserService 用户服务
type UserService struct {
	// TODO: 注入 repository
}

// NewUserService 创建服务
func NewUserService() *UserService {
	return &UserService{}
}

// User 用户实体
type User struct {
	Id    int64
	Name  string
	Email string
}

// CreateUser 创建用户
func (s *UserService) CreateUser(ctx context.Context, name, email string) (*User, error) {
	// TODO: 实现业务逻辑
	if name == "" {
		return nil, errs.InvalidParam("name is required")
	}
	if email == "" {
		return nil, errs.InvalidParam("email is required")
	}

	// 模拟创建
	user := &User{
		Id:    1,
		Name:  name,
		Email: email,
	}
	return user, nil
}

// GetUser 获取用户
func (s *UserService) GetUser(ctx context.Context, id int64) (*User, error) {
	// TODO: 实现业务逻辑
	if id <= 0 {
		return nil, errs.InvalidParam("invalid id")
	}

	// 模拟查询
	if id == 1 {
		return &User{Id: 1, Name: "test", Email: "test@example.com"}, nil
	}
	return nil, errs.NotFound("user not found")
}

// ListUsers 列出用户
func (s *UserService) ListUsers(ctx context.Context, page, size int) ([]*User, int, error) {
	// TODO: 实现业务逻辑
	users := []*User{
		{Id: 1, Name: "test", Email: "test@example.com"},
	}
	return users, 1, nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(ctx context.Context, id int64) error {
	// TODO: 实现业务逻辑
	return nil
}
