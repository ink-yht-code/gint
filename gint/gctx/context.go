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
//
// This file is derived from ginx (https://github.com/ecodeclub/ginx)
// Original Copyright by ecodeclub and contributors
// Modifications: Added more convenience methods, optimized EventStream

package gctx

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

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

// UserId 从上下文中获取用户 ID
// 通常由认证中间件设置
func (c *Context) UserId() string {
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
func (c *Context) SetUserId(userId string) {
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
