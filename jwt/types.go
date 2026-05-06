// Copyright 2025 ink-yht-code
//
// Proprietary License
//
// IMPORTANT: This software is NOT open source.
// You may NOT use, copy, modify, merge, publish, distribute, sublicense,
// or sell copies of this file, in whole or in part, without prior written
// permission from the copyright holder.
//
// This software is provided "AS IS", without warranty of any kind.

package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 是项目统一使用的 JWT 声明结构。
type Claims struct {
	UserId   string            `json:"user_id"`             // 用户 ID
	SSID     string            `json:"ssid"`                // Session ID
	TenantId string            `json:"tenant_id,omitempty"` // 租户 ID
	Role     string            `json:"role,omitempty"`      // 角色
	Username string            `json:"username,omitempty"`  // 用户名
	Email    string            `json:"email,omitempty"`     // 邮箱
	Data     map[string]string `json:"data,omitempty"`      // 额外扩展数据
	jwt.RegisteredClaims
}

// Options 是 JWT 配置项。
type Options struct {
	SignKey       string
	AccessExpire  time.Duration
	RefreshExpire time.Duration
	Method        jwt.SigningMethod
	Issuer        string
}

// NewOptions 创建默认 JWT 配置。
func NewOptions(signKey string, accessExpire, refreshExpire time.Duration) Options {
	return Options{
		SignKey:       signKey,
		AccessExpire:  accessExpire,
		RefreshExpire: refreshExpire,
		Method:        jwt.SigningMethodHS256,
		Issuer:        "gint",
	}
}

// TokenPair 是访问令牌和刷新令牌的组合。
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Manager 定义 JWT 的生成与校验能力。
type Manager interface {
	GenerateToken(claims Claims) (string, error)
	GenerateTokenPair(claims Claims) (*TokenPair, error)
	VerifyToken(token string) (*Claims, error)
	VerifyRefreshToken(token string) (*Claims, error)
}
