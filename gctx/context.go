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
//
// This file is derived from ginx (https://github.com/ecodeclub/ginx)
// Original Copyright by ecodeclub and contributors
// Modifications: Added more convenience methods, optimized EventStream

package gctx

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
)

// 错误定义
var (
	// ErrUserContextNotFound 用户上下文未找到
	ErrUserContextNotFound = errors.New("user context not found")
)

// UserContext 用户上下文信息
// 用于在微服务间传递用户身份信息
const (
	// CtxUserContextKey 在 Context 中存储 UserContext 的 key
	CtxUserContextKey = "gint:user_context"
)

// UserContext 用户上下文信息
// 网关认证后注入，下游微服务可直接获取
type UserContext struct {
	UserId     string            `json:"user_id"`    // 用户ID
	Username   string            `json:"username"`   // 用户名
	Role       string            `json:"role"`       // 角色
	TenantId   string            `json:"tenant_id"`  // 租户ID
	Department string            `json:"department"` // 部门
	Email      string            `json:"email"`      // 邮箱
	Extra      map[string]string `json:"extra"`      // 扩展字段
}

// IsEmpty 判断 UserContext 是否为空
func (u *UserContext) IsEmpty() bool {
	return u == nil || u.UserId == ""
}

// Context 是对 gin.Context 的增强封装
// 提供了更便捷的参数获取和类型转换方法
type Context struct {
	*gin.Context
}

// Value 封装了值和错误，支持链式类型转换
type Value struct {
	val string
	err error
}

// String 返回字符串值
func (v Value) String() (string, error) {
	return v.val, v.err
}

// StringOr 返回字符串值，如果有错误则返回默认值
func (v Value) StringOr(defaultVal string) string {
	if v.err != nil {
		return defaultVal
	}
	return v.val
}

// Int 将值转换为 int
func (v Value) Int() (int, error) {
	if v.err != nil {
		return 0, v.err
	}
	return strconv.Atoi(v.val)
}

// IntOr 将值转换为 int，如果失败则返回默认值
func (v Value) IntOr(defaultVal int) int {
	if v.err != nil {
		return defaultVal
	}
	val, err := strconv.Atoi(v.val)
	if err != nil {
		return defaultVal
	}
	return val
}

// Int64 将值转换为 int64
func (v Value) Int64() (int64, error) {
	if v.err != nil {
		return 0, v.err
	}
	return strconv.ParseInt(v.val, 10, 64)
}

// Int64Or 将值转换为 int64，如果失败则返回默认值
func (v Value) Int64Or(defaultVal int64) int64 {
	if v.err != nil {
		return defaultVal
	}
	val, err := strconv.ParseInt(v.val, 10, 64)
	if err != nil {
		return defaultVal
	}
	return val
}

// Bool 将值转换为 bool
func (v Value) Bool() (bool, error) {
	if v.err != nil {
		return false, v.err
	}
	return strconv.ParseBool(v.val)
}

// BoolOr 将值转换为 bool，如果失败则返回默认值
func (v Value) BoolOr(defaultVal bool) bool {
	if v.err != nil {
		return defaultVal
	}
	val, err := strconv.ParseBool(v.val)
	if err != nil {
		return defaultVal
	}
	return val
}

// Param 获取路径参数
func (c *Context) Param(key string) Value {
	return Value{
		val: c.Context.Param(key),
	}
}

// Query 获取查询参数
func (c *Context) Query(key string) Value {
	return Value{
		val: c.Context.Query(key),
	}
}

// Cookie 获取 Cookie 值
func (c *Context) Cookie(key string) Value {
	val, err := c.Context.Cookie(key)
	return Value{
		val: val,
		err: err,
	}
}

// Header 获取请求头
func (c *Context) Header(key string) Value {
	return Value{
		val: c.Context.GetHeader(key),
	}
}

// UserContext 从上下文中获取完整的用户上下文
// 通常由网关或认证中间件设置
func (c *Context) UserContext() *UserContext {
	val, exists := c.Get(CtxUserContextKey)
	if !exists {
		return nil
	}
	uc, ok := val.(*UserContext)
	if !ok {
		return nil
	}
	return uc
}

// SetUserContext 设置完整的用户上下文
func (c *Context) SetUserContext(uc *UserContext) {
	c.Set(CtxUserContextKey, uc)
	// 兼容旧接口，同时设置 user_id
	if uc != nil {
		c.Set("user_id", uc.UserId)
	}
}

// UserId 从上下文中获取用户 ID
// 通常由认证中间件设置
func (c *Context) UserId() string {
	uc := c.UserContext()
	if uc != nil {
		return uc.UserId
	}
	// 兼容旧的设置方式
	val, exists := c.Get("user_id")
	if !exists {
		return ""
	}
	userId, ok := val.(string)
	if !ok {
		return ""
	}
	return userId
}

// SetUserId 设置用户 ID 到上下文
// 注意：推荐使用 SetUserContext 设置完整用户信息
func (c *Context) SetUserId(userId string) {
	c.Set("user_id", userId)
}

// Role 获取用户角色
func (c *Context) Role() string {
	uc := c.UserContext()
	if uc != nil {
		return uc.Role
	}
	return ""
}

// TenantId 获取租户ID
func (c *Context) TenantId() string {
	uc := c.UserContext()
	if uc != nil {
		return uc.TenantId
	}
	return ""
}

// Username 获取用户名
func (c *Context) Username() string {
	uc := c.UserContext()
	if uc != nil {
		return uc.Username
	}
	return ""
}

// Email 获取用户邮箱
func (c *Context) Email() string {
	uc := c.UserContext()
	if uc != nil {
		return uc.Email
	}
	return ""
}

// MustUserId 获取用户ID，如果不存在则返回错误
func (c *Context) MustUserId() (string, error) {
	userId := c.UserId()
	if userId == "" {
		return "", ErrUserContextNotFound
	}
	return userId, nil
}

// MustUserContext 获取用户上下文，如果不存在则返回错误
func (c *Context) MustUserContext() (*UserContext, error) {
	uc := c.UserContext()
	if uc == nil || uc.UserId == "" {
		return nil, ErrUserContextNotFound
	}
	return uc, nil
}

// TraceId 从上下文中获取链路追踪 ID
// 由 TraceMiddleware 自动设置
func (c *Context) TraceId() string {
	val, exists := c.Get("trace_id")
	if !exists {
		return ""
	}
	traceId, ok := val.(string)
	if !ok {
		return ""
	}
	return traceId
}

// SetTraceId 设置链路追踪 ID 到上下文
func (c *Context) SetTraceId(traceId string) {
	c.Set("trace_id", traceId)
}

// SetTraceAndUser 同时设置 traceID 和 userID
func (c *Context) SetTraceAndUser(traceId, userId string) {
	c.Set("trace_id", traceId)
	c.Set("user_id", userId)
}

// EventStream 返回一个用于 Server-Sent Events 的通道
// 用于实现服务器推送功能
// 注意：调用者需要在完成后关闭返回的 channel
func (c *Context) EventStream() chan []byte {
	// 设置 SSE 响应头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	eventCh := make(chan []byte, 10)

	// 启动协程处理事件发送
	go func() {
		for {
			select {
			case eventData, ok := <-eventCh:
				if !ok {
					// channel 已被调用者关闭
					return
				}
				if len(eventData) > 0 {
					c.sendEvent(eventData)
				}
			case <-c.Request.Context().Done():
				// 客户端断开连接
				return
			}
		}
	}()

	return eventCh
}

// sendEvent 发送 SSE 事件
func (c *Context) sendEvent(data []byte) {
	_, _ = c.Writer.Write(data)
	c.Writer.Flush()
}

// JSON 返回 JSON 响应
func (c *Context) JSON(code int, obj any) {
	c.Context.JSON(code, obj)
}

// Success 返回成功响应
func (c *Context) Success(data any) {
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "success",
		"data": data,
	})
}

// Error 返回错误响应
func (c *Context) Error(code int, msg string) {
	c.JSON(200, gin.H{
		"code": code,
		"msg":  msg,
		"data": nil,
	})
}
