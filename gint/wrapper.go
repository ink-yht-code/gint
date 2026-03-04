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
// Modifications: Simplified implementation, removed enterprise features

package gint

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/gctx"
	"github.com/ink-yht-code/gint/session"
)

// W (Wrapper) 基础包装器
// 只接收 Context，适用于不需要参数绑定和 Session 的简单接口
//
// 示例:
//
//	router.GET("/ping", gint.W(func(ctx *gint.Context) (gint.Result, error) {
//	   return gint.Result{Code: 0, Msg: "pong"}, nil
//	}))
func W(fn func(ctx *gctx.Context) (Result, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := &gctx.Context{Context: c}

		// 执行业务逻辑
		res, err := fn(ctx)

		// 处理特殊错误
		if errors.Is(err, ErrNoResponse) {
			slog.Debug("不需要响应", slog.Any("err", err))
			return
		}

		if errors.Is(err, ErrUnauthorized) {
			slog.Debug("未授权", slog.Any("err", err))
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 处理一般错误
		if err != nil {
			slog.Error("执行业务逻辑失败",
				slog.String("path", c.Request.URL.Path),
				slog.Any("err", err))
			if res.Code == 0 {
				res = InternalError()
			} else {
				res.Data = nil
				if res.Msg == "" {
					res.Msg = GetCodeMessage(res.Code)
				}
			}
			c.JSON(http.StatusOK, res)
			return
		}

		// 返回成功响应
		c.JSON(http.StatusOK, res)
	}
}

// B (Bind) 带参数绑定的包装器
// 使用泛型自动绑定请求参数，适用于需要接收请求参数的接口
//
// 示例:
//
//	type LoginReq struct {
//	   Username string `json:"username"`
//	   Password string `json:"password"`
//	}
//
//	router.POST("/login", gint.B(func(ctx *gint.Context, req LoginReq) (gint.Result, error) {
//	   // req 已经自动绑定
//	   return gint.Result{Code: 0, Data: "登录成功"}, nil
//	}))
func B[Req any](fn func(ctx *gctx.Context, req Req) (Result, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := &gctx.Context{Context: c}

		// 绑定请求参数
		var req Req
		if err := c.ShouldBind(&req); err != nil {
			slog.Debug("绑定参数失败",
				slog.String("path", c.Request.URL.Path),
				slog.Any("err", err))
			c.JSON(http.StatusBadRequest, InvalidParam("参数错误"))
			return
		}

		// 执行业务逻辑
		res, err := fn(ctx, req)

		// 处理特殊错误
		if errors.Is(err, ErrNoResponse) {
			slog.Debug("不需要响应", slog.Any("err", err))
			return
		}

		if errors.Is(err, ErrUnauthorized) {
			slog.Debug("未授权", slog.Any("err", err))
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 处理一般错误
		if err != nil {
			slog.Error("执行业务逻辑失败",
				slog.String("path", c.Request.URL.Path),
				slog.Any("err", err))
			if res.Code == 0 {
				res = InternalError()
			} else {
				res.Data = nil
				if res.Msg == "" {
					res.Msg = GetCodeMessage(res.Code)
				}
			}
			c.JSON(http.StatusOK, res)
			return
		}

		// 返回成功响应
		c.JSON(http.StatusOK, res)
	}
}

// S (Session) 带 Session 的包装器
// 自动获取和校验 Session，适用于需要登录态的接口
//
// 示例:
//
//	router.GET("/profile", gint.S(func(ctx *gint.Context, sess session.Session) (gint.Result, error) {
//	   userId := sess.Claims().UserId
//	   return gint.Result{Code: 0, Data: userId}, nil
//	}))
func S(fn func(ctx *gctx.Context, sess session.Session) (Result, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := &gctx.Context{Context: c}

		// 获取 Session
		sess, err := session.Get(ctx)
		if err != nil {
			slog.Debug("获取 Session 失败",
				slog.String("path", c.Request.URL.Path),
				slog.Any("err", err))
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 执行业务逻辑
		res, err := fn(ctx, sess)

		// 处理特殊错误
		if errors.Is(err, ErrNoResponse) {
			slog.Debug("不需要响应", slog.Any("err", err))
			return
		}

		if errors.Is(err, ErrUnauthorized) {
			slog.Debug("未授权", slog.Any("err", err))
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 处理一般错误
		if err != nil {
			slog.Error("执行业务逻辑失败",
				slog.String("path", c.Request.URL.Path),
				slog.String("user_id", sess.Claims().UserId),
				slog.Any("err", err))
			if res.Code == 0 {
				res = InternalError()
			} else {
				res.Data = nil
				if res.Msg == "" {
					res.Msg = GetCodeMessage(res.Code)
				}
			}
			c.JSON(http.StatusOK, res)
			return
		}

		// 返回成功响应
		c.JSON(http.StatusOK, res)
	}
}

// BS (Bind + Session) 带参数绑定和 Session 的包装器
// 结合了 B 和 S 的功能，适用于需要登录且需要接收参数的接口
//
// 示例:
//
//	type UpdateProfileReq struct {
//	   Nickname string `json:"nickname"`
//	   Avatar   string `json:"avatar"`
//	}
//
//	router.POST("/profile", gint.BS(func(ctx *gint.Context, req UpdateProfileReq, sess session.Session) (gint.Result, error) {
//	   userId := sess.Claims().UserId
//	   // 更新用户信息...
//	   return gint.Result{Code: 0, Msg: "更新成功"}, nil
//	}))
func BS[Req any](fn func(ctx *gctx.Context, req Req, sess session.Session) (Result, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := &gctx.Context{Context: c}

		// 获取 Session
		sess, err := session.Get(ctx)
		if err != nil {
			slog.Debug("获取 Session 失败",
				slog.String("path", c.Request.URL.Path),
				slog.Any("err", err))
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 绑定请求参数
		var req Req
		if err := c.ShouldBind(&req); err != nil {
			slog.Debug("绑定参数失败",
				slog.String("path", c.Request.URL.Path),
				slog.String("user_id", sess.Claims().UserId),
				slog.Any("err", err))
			c.JSON(http.StatusBadRequest, InvalidParam("参数错误"))
			return
		}

		// 执行业务逻辑
		res, err := fn(ctx, req, sess)

		// 处理特殊错误
		if errors.Is(err, ErrNoResponse) {
			slog.Debug("不需要响应", slog.Any("err", err))
			return
		}

		if errors.Is(err, ErrUnauthorized) {
			slog.Debug("未授权", slog.Any("err", err))
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 处理一般错误
		if err != nil {
			slog.Error("执行业务逻辑失败",
				slog.String("path", c.Request.URL.Path),
				slog.String("user_id", sess.Claims().UserId),
				slog.Any("err", err))
			if res.Code == 0 {
				res = InternalError()
			} else {
				res.Data = nil
				if res.Msg == "" {
					res.Msg = GetCodeMessage(res.Code)
				}
			}
			c.JSON(http.StatusOK, res)
			return
		}

		// 返回成功响应
		c.JSON(http.StatusOK, res)
	}
}
