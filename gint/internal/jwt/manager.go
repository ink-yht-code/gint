// Copyright 2025 ink-yht-code
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var _ Manager = (*manager)(nil)

// manager JWT 管理器实现
type manager struct {
	opts Options
}

// NewManager 创建 JWT 管理器
func NewManager(opts Options) Manager {
	return &manager{
		opts: opts,
	}
}

// GenerateToken 生成 Access Token（兼容旧版本）
func (m *manager) GenerateToken(claims Claims) (string, error) {
	return m.generateToken(claims, m.opts.AccessExpire)
}

// GenerateTokenPair 生成 Token 对（Access Token + Refresh Token）
func (m *manager) GenerateTokenPair(claims Claims) (*TokenPair, error) {
	// 生成 Access Token
	accessToken, err := m.generateToken(claims, m.opts.AccessExpire)
	if err != nil {
		return nil, fmt.Errorf("生成 Access Token 失败: %w", err)
	}

	// 生成 Refresh Token
	refreshToken, err := m.generateToken(claims, m.opts.RefreshExpire)
	if err != nil {
		return nil, fmt.Errorf("生成 Refresh Token 失败: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// generateToken 生成 Token 的内部方法
func (m *manager) generateToken(claims Claims, expire time.Duration) (string, error) {
	now := time.Now()

	// 设置标准声明
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    m.opts.Issuer,
		ExpiresAt: jwt.NewNumericDate(now.Add(expire)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ID:        uuid.New().String(),
	}

	// 创建 Token
	token := jwt.NewWithClaims(m.opts.Method, claims)

	// 签名并返回
	return token.SignedString([]byte(m.opts.SignKey))
}

// VerifyToken 验证 Access Token
func (m *manager) VerifyToken(tokenString string) (*Claims, error) {
	return m.verifyToken(tokenString)
}

// VerifyRefreshToken 验证 Refresh Token
func (m *manager) VerifyRefreshToken(tokenString string) (*Claims, error) {
	return m.verifyToken(tokenString)
}

// verifyToken 验证 Token 的内部方法
func (m *manager) verifyToken(tokenString string) (*Claims, error) {
	// 解析 Token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if token.Method != m.opts.Method {
			return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
		}
		return []byte(m.opts.SignKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("解析 Token 失败: %w", err)
	}

	// 验证 Token 是否有效
	if !token.Valid {
		return nil, fmt.Errorf("无效的 Token")
	}

	// 提取 Claims
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("无效的 Claims 类型")
	}

	return claims, nil
}
