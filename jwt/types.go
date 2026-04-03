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

// Claims JWT 声明结构
type Claims struct {
	UserId string            `json:"user_id"` // 用户 ID（使用 string 类型）
	SSID   string            `json:"ssid"`    // Session ID
	Data   map[string]string `json:"data"`    // 额外数据
	jwt.RegisteredClaims
}

// Options JWT 配置选项
type Options struct {
	// 签名密钥
	SignKey string
	// Access Token 过期时间
	AccessExpire time.Duration
	// Refresh Token 过期时间
	RefreshExpire time.Duration
	// 签名方法
	Method jwt.SigningMethod
	// 发行者
	Issuer string
}

// NewOptions 创建默认的 JWT 配置
// accessExpire: Access Token 过期时间（建议 15 分钟 - 2 小时）
// refreshExpire: Refresh Token 过期时间（建议 7 天 - 30 天）
func NewOptions(signKey string, accessExpire, refreshExpire time.Duration) Options {
	return Options{
		SignKey:       signKey,
		AccessExpire:  accessExpire,
		RefreshExpire: refreshExpire,
		Method:        jwt.SigningMethodHS256,
		Issuer:        "gint",
	}
}

// TokenPair Token 对（Access Token + Refresh Token）
type TokenPair struct {
	AccessToken  string `json:"access_token"`  // 访问令牌（短期有效）
	RefreshToken string `json:"refresh_token"` // 刷新令牌（长期有效）
}

// Manager JWT 管理器接口
type Manager interface {
	// GenerateToken 生成单个 Token（兼容旧版本）
	GenerateToken(claims Claims) (string, error)

	// GenerateTokenPair 生成 Token 对（Access Token + Refresh Token）
	GenerateTokenPair(claims Claims) (*TokenPair, error)

	// VerifyToken 验证 Token
	VerifyToken(token string) (*Claims, error)

	// VerifyRefreshToken 验证 Refresh Token
	VerifyRefreshToken(token string) (*Claims, error)
}
