# gint

[![License](https://img.shields.io/badge/License-Proprietary-red.svg)](#license)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://golang.org)
[![Version](https://img.shields.io/badge/version-v1.0.0-blue.svg)](https://github.com/ink-yht-code/gint/releases)

`gint` 是一个基于 Gin 的企业级 HTTP 框架，提供 Handler 包装器、参数校验、Session 与 JWT、常用中间件、配置管理、Swagger/OpenAPI 以及服务治理相关能力，帮助团队更快搭建 Go 服务。

## 亮点

- 简洁的 Handler 包装器
- 统一的结果结构与错误处理
- Session 与 JWT 支持
- 常用中间件能力，如 CORS、访问日志、认证、限流、健康检查
- 配置文件与环境变量加载
- Swagger / OpenAPI 构建支持
- 面向服务的基础设施能力

## 安装

```bash
go get github.com/ink-yht-code/gint
```

## 快速开始

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint"
	"github.com/ink-yht-code/gint/gctx"
)

func main() {
	r := gin.Default()

	r.GET("/ping", gint.W(func(ctx *gctx.Context) (gint.Result, error) {
		return gint.Result{Code: 0, Msg: "pong"}, nil
	}))

	_ = r.Run(":8080")
}
```

## 核心包

- `gint`: Handler 包装器、结果结构、校验能力
- `gctx`: 请求上下文增强
- `jwt`: Token 生成与校验
- `session`: Session 抽象与 Provider
- `middlewares`: 常用 HTTP 中间件
- `config`: 配置加载
- `logger`: 基于 zap 的日志封装
- `swagger`: OpenAPI 构建与 UI
- `server`: HTTP 服务与优雅关闭

## 日志示例

```go
package main

import (
	"github.com/ink-yht-code/gint/logger"
	"go.uber.org/zap"
)

func main() {
	err := logger.Init(logger.Config{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		panic(err)
	}

	logger.Info("服务启动", zap.String("addr", ":8080"))
}
```

## 配置示例

```go
cfg, err := config.LoadApp("config/dev.yaml")
if err != nil {
	panic(err)
}
```

## 说明

- 本仓库使用专有授权
- 部分模块属于可选能力，按需引入即可
- 本地开发建议优先使用 `stdout` 或 `both` 输出日志

## License

本项目使用专有许可证，具体使用限制请以仓库中的版权与许可说明为准。
