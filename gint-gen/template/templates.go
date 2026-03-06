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

package template

// GoModTmpl go.mod 模板
var GoModTmpl = `module {{.Name}}

go 1.25

require (
	github.com/gin-gonic/gin v1.12.0
	github.com/ink-yht-code/gint/gint v0.0.0
	github.com/ink-yht-code/gint/gintx v0.0.0
)

replace (
	github.com/ink-yht-code/gint/gint => github.com/ink-yht-code/gint/gint v0.1.9
	github.com/ink-yht-code/gint/gintx => github.com/ink-yht-code/gint/gintx v0.1.5
)
`

// GintTmpl .gint 文件模板
var GintTmpl = `syntax = "v1"

type HelloReq {
    Name string ` + "`" + `json:"name"` + "`" + `
}

type HelloResp {
    Message string ` + "`" + `json:"message"` + "`" + `
}

server {
    prefix "/api/v1"
}

service {{.Name}} {
    public {
        GET "/hello" Hello(HelloReq) -> HelloResp
    }
}
`

// ConfigYamlTmpl 配置文件模板
var ConfigYamlTmpl = `service:
  id: {{.ServiceID}}
  name: {{.Name}}

http:
  enabled: {{.HasHTTP}}
  addr: ":8080"

grpc:
  enabled: {{.HasRPC}}
  addr: ":9090"

db:
  dsn: "user:pass@tcp(127.0.0.1:3306)/{{.Name}}?charset=utf8mb4&parseTime=True&loc=Local"
  max_open: 100
  max_idle: 10
  log_level: "info"

redis:
  addr: "127.0.0.1:6379"
  password: ""
  db: 0

log:
  level: "info"
  encoding: "json"
  output: "stdout"

outbox:
  enabled: true
  batch_size: 100
  poll_interval: "5s"
`

// MainTmpl main.go 模板
var MainTmpl = `package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"{{.Name}}/internal/wiring"
)

func main() {
	// 加载配置
	cfg, err := wiring.LoadConfig("configs/{{.Name}}.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 构建应用
	app, err := wiring.BuildApp(cfg)
	if err != nil {
		fmt.Printf("Failed to build app: %v\n", err)
		os.Exit(1)
	}

	// 启动
	go func() {
		if err := app.Run(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	fmt.Printf("Service {{.Name}} started\n")

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	app.Shutdown(ctx)
	fmt.Println("Service exited")
}
`

// ConfigGoTmpl config.go 模板
var ConfigGoTmpl = `package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Service ServiceConfig ` + "`" + `yaml:"service"` + "`" + `
	HTTP    HTTPConfig    ` + "`" + `yaml:"http"` + "`" + `
	GRPC    GRPCConfig    ` + "`" + `yaml:"grpc"` + "`" + `
	DB      DBConfig      ` + "`" + `yaml:"db"` + "`" + `
	Redis   RedisConfig   ` + "`" + `yaml:"redis"` + "`" + `
	Log     LogConfig     ` + "`" + `yaml:"log"` + "`" + `
	Outbox  OutboxConfig  ` + "`" + `yaml:"outbox"` + "`" + `
}

type ServiceConfig struct {
	ID   int    ` + "`" + `yaml:"id"` + "`" + `
	Name string ` + "`" + `yaml:"name"` + "`" + `
}

type HTTPConfig struct {
	Enabled bool   ` + "`" + `yaml:"enabled"` + "`" + `
	Addr    string ` + "`" + `yaml:"addr"` + "`" + `
}

type GRPCConfig struct {
	Enabled bool   ` + "`" + `yaml:"enabled"` + "`" + `
	Addr    string ` + "`" + `yaml:"addr"` + "`" + `
}

type DBConfig struct {
	DSN     string ` + "`" + `yaml:"dsn"` + "`" + `
	MaxOpen int    ` + "`" + `yaml:"max_open"` + "`" + `
	MaxIdle int    ` + "`" + `yaml:"max_idle"` + "`" + `
	LogLevel string ` + "`" + `yaml:"log_level"` + "`" + `
}

type RedisConfig struct {
	Addr     string ` + "`" + `yaml:"addr"` + "`" + `
	Password string ` + "`" + `yaml:"password"` + "`" + `
	DB       int    ` + "`" + `yaml:"db"` + "`" + `
}

type LogConfig struct {
	Level    string ` + "`" + `yaml:"level"` + "`" + `
	Encoding string ` + "`" + `yaml:"encoding"` + "`" + `
	Output   string ` + "`" + `yaml:"output"` + "`" + `
}

type OutboxConfig struct {
	Enabled     bool          ` + "`" + `yaml:"enabled"` + "`" + `
	BatchSize   int           ` + "`" + `yaml:"batch_size"` + "`" + `
	PollInterval time.Duration ` + "`" + `yaml:"poll_interval"` + "`" + `
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
`

// CodesTmpl 错误码模板
var CodesTmpl = `package errs

// ServiceID 服务 ID
const ServiceID = {{.ServiceID}}

// 业务码 = ServiceID * 10000 + BizCode
const (
	CodeSuccess       = ServiceID * 10000 + 0
	CodeInvalidParam  = ServiceID * 10000 + 1
	CodeUnauthorized  = ServiceID * 10000 + 2
	CodeForbidden     = ServiceID * 10000 + 3
	CodeNotFound      = ServiceID * 10000 + 4
	CodeConflict      = ServiceID * 10000 + 5
	CodeInternalError = ServiceID * 10000 + 9999
)
`

// ErrorTmpl 错误类型模板
var ErrorTmpl = `package errs

import "fmt"

// BizError 业务错误
type BizError struct {
	Code  int
	Msg   string
	Cause error
}

func (e *BizError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Msg, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}

func (e *BizError) Unwrap() error {
	return e.Cause
}

// NewBizError 创建业务错误
func NewBizError(code int, msg string, cause ...error) *BizError {
	var c error
	if len(cause) > 0 {
		c = cause[0]
	}
	return &BizError{Code: code, Msg: msg, Cause: c}
}

// InvalidParam 参数错误
func InvalidParam(msg string, cause ...error) *BizError {
	return NewBizError(CodeInvalidParam, msg, cause...)
}

// Unauthorized 未授权
func Unauthorized(msg string, cause ...error) *BizError {
	return NewBizError(CodeUnauthorized, msg, cause...)
}

// Forbidden 无权限
func Forbidden(msg string, cause ...error) *BizError {
	return NewBizError(CodeForbidden, msg, cause...)
}

// NotFound 未找到
func NotFound(msg string, cause ...error) *BizError {
	return NewBizError(CodeNotFound, msg, cause...)
}

// Conflict 冲突
func Conflict(msg string, cause ...error) *BizError {
	return NewBizError(CodeConflict, msg, cause...)
}

// InternalError 内部错误
func InternalError(msg string, cause ...error) *BizError {
	return NewBizError(CodeInternalError, msg, cause...)
}
`

// WiringTmpl wiring 模板
var WiringTmpl = `package wiring

import (
	"context"

	"github.com/ink-yht-code/gint/gintx/httpx"
	"github.com/ink-yht-code/gint/gintx/log"
	"go.uber.org/zap"

	"{{.Name}}/internal/config"
	"{{.Name}}/internal/repository"
	"{{.Name}}/internal/repository/dao"
	"{{.Name}}/internal/service"
	"{{.Name}}/internal/web"
)

// LoadConfig 加载配置
func LoadConfig(path string) (*config.Config, error) {
	return config.Load(path)
}

// App 应用
type App struct {
	cfg  *config.Config
	http *httpx.Server
}

// BuildApp 构建应用
func BuildApp(cfg *config.Config) (*App, error) {
	// 初始化日志
	if err := log.Init(log.Config{
		Level:    cfg.Log.Level,
		Encoding: cfg.Log.Encoding,
		Output:   cfg.Log.Output,
	}); err != nil {
		return nil, err
	}

	// 创建服务
	var d dao.{{.NameUpper}}DAO = dao.New{{.NameUpper}}DAO()
	repo := repository.New{{.NameUpper}}Repository(d)
	svc := service.New{{.NameUpper}}Service(repo)

	// 创建 Handler
	handler := web.NewHandler(svc)

	// 创建 HTTP server
	var httpServer *httpx.Server
	if cfg.HTTP.Enabled {
		httpServer = httpx.NewServer(httpx.Config{
			Enabled: cfg.HTTP.Enabled,
			Addr:    cfg.HTTP.Addr,
		})
		
		// 注册公开路由
		handler.PublicRoutes(httpServer.Engine)
		
		// 注册私有路由 (需要认证)
		// 可以使用 JWT 中间件保护私有路由
		// authGroup := httpServer.Engine.Group("/")
		// authGroup.Use(jwtMiddleware)
		// handler.PrivateRoutes(httpServer.Engine)
		
		// 健康检查
		httpServer.Engine.GET("/health", handler.Health)
	}

	return &App{cfg: cfg, http: httpServer}, nil
}

// Run 启动应用
func (a *App) Run() error {
	if a.http != nil {
		log.Info("HTTP server starting", zap.String("addr", a.cfg.HTTP.Addr))
		return a.http.Run()
	}
	return nil
}

// Shutdown 关闭应用
func (a *App) Shutdown(ctx context.Context) error {
	if a.http != nil {
		return a.http.Shutdown(ctx)
	}
	return nil
}
`

// ServiceTmpl service 模板
var ServiceTmpl = `package service

import (
	"context"

	"{{.Name}}/internal/domain/port"
)

// {{.NameUpper}}Service {{.Name}} 服务
type {{.NameUpper}}Service struct {
	repo port.{{.NameUpper}}Repository
}

// New{{.NameUpper}}Service 创建服务
func New{{.NameUpper}}Service(repo port.{{.NameUpper}}Repository) *{{.NameUpper}}Service {
	return &{{.NameUpper}}Service{repo: repo}
}

// Hello 示例方法
func (s *{{.NameUpper}}Service) Hello(ctx context.Context, name string) (string, error) {
	return "Hello, " + name, nil
}
`

// HTTPHandlerGenTmpl HTTP handler 生成文件模板（可覆盖）
var HTTPHandlerGenTmpl = `// Code generated by gint-gen. DO NOT EDIT.
// source: {{.Name}}.gint

package web

import (
	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/gint"
	"{{.Name}}/internal/service"
)

// Handler HTTP 处理器
// 实现 gint.Handler 接口
//
// 说明：
// - 本文件可被重复生成覆盖
// - 业务逻辑请在 <service>_handlers.go 中实现
type Handler struct {
	svc *service.{{.NameUpper}}Service
}

// NewHandler 创建 Handler
func NewHandler(svc *service.{{.NameUpper}}Service) *Handler {
	return &Handler{svc: svc}
}

// PrivateRoutes 注册需要认证的路由
func (h *Handler) PrivateRoutes(server *gin.Engine) {
	// TODO: 添加需要认证的路由
}

// PublicRoutes 注册公开的路由
func (h *Handler) PublicRoutes(server *gin.Engine) {
	server.GET("/api/v1/hello", gint.B(h.Hello))
}

// Health 健康检查
func (h *Handler) Health(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}
`

// HTTPHandlerImplTmpl HTTP handler 用户实现文件模板（仅首次创建）
var HTTPHandlerImplTmpl = `package web

import (
	"github.com/ink-yht-code/gint/gint"
	"github.com/ink-yht-code/gint/gint/gctx"
	"{{.Name}}/internal/types"
)

// Hello 示例 handler
func (h *Handler) Hello(ctx *gctx.Context, req *types.HelloReq) (gint.Result, error) {
	return gint.Result{Code: 0, Data: &types.HelloResp{Message: "ok"}}, nil
}
`

// HTTPTmpl HTTP handler 模板
var HTTPTmpl = `// Code generated by gint-gen. DO NOT EDIT.
// source: {{.Name}}.gint

package web

import (
	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/gint"
	"github.com/ink-yht-code/gint/gint/gctx"
	"{{.Name}}/internal/service"
	"{{.Name}}/internal/types"
)

// Handler HTTP 处理器
// 实现 gint.Handler 接口
type Handler struct {
	svc *service.{{.NameUpper}}Service
}

// NewHandler 创建 Handler
func NewHandler(svc *service.{{.NameUpper}}Service) *Handler {
	return &Handler{svc: svc}
}

// PrivateRoutes 注册需要认证的路由
func (h *Handler) PrivateRoutes(server *gin.Engine) {
	// TODO: 添加需要认证的路由
	// 示例: 使用 gint.S (带 Session 的包装器)
	// server.GET("/api/v1/profile", gint.S(h.Profile))
}

// PublicRoutes 注册公开的路由
func (h *Handler) PublicRoutes(server *gin.Engine) {
	// 使用 gint.B (带参数绑定的包装器)
	server.GET("/api/v1/hello", gint.B(h.Hello))
}

// Hello 示例 handler
// 使用 gint.B 包装器自动绑定参数
func (h *Handler) Hello(ctx *gctx.Context, req *types.HelloReq) (gint.Result, error) {
	// 使用 gint.validator 进行参数校验
	vb := gint.NewValidatorBuilder().
		Field("name", req.Name).AddRule(gint.Required()).
		Validate()
	
	if !vb.IsValid() {
		return gint.InvalidParam(vb.GetFirstError()), nil
	}

	msg, err := h.svc.Hello(ctx.Request.Context(), req.Name)
	if err != nil {
		return gint.InternalError(), err
	}
	return gint.Result{Code: 0, Data: &types.HelloResp{Message: msg}}, nil
}

// Health 健康检查
func (h *Handler) Health(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}
`

// TypesTmpl types 模板
var TypesTmpl = `// Code generated by gint-gen. DO NOT EDIT.
// source: {{.Name}}.gint

package types

// HelloReq Hello 请求
type HelloReq struct {
	Name string ` + "`" + `json:"name" form:"name"` + "`" + `
}

// HelloResp Hello 响应
type HelloResp struct {
	Message string ` + "`" + `json:"message"` + "`" + `
}
`

// RepositoryPortTmpl repository port 接口模板
var RepositoryPortTmpl = `package port

import (
	"context"

	"{{.Name}}/internal/domain/entity"
)

// {{.NameUpper}}Repository {{.Name}} 仓储接口
type {{.NameUpper}}Repository interface {
	// Save 保存实体
	Save(ctx context.Context, e *entity.{{.NameUpper}}) error
	// FindByID 根据 ID 查找
	FindByID(ctx context.Context, id int64) (*entity.{{.NameUpper}}, error)
}
`

// RepositoryImplTmpl repository 实现模板
var RepositoryImplTmpl = `package repository

import (
	"context"

	"{{.Name}}/internal/domain/entity"
	"{{.Name}}/internal/domain/port"
	"{{.Name}}/internal/repository/dao"
)

type {{.NameUpper}}Repository struct {
	dao dao.{{.NameUpper}}DAO
}

func New{{.NameUpper}}Repository(dao dao.{{.NameUpper}}DAO) port.{{.NameUpper}}Repository {
	return &{{.NameUpper}}Repository{dao: dao}
}

func (r *{{.NameUpper}}Repository) Save(ctx context.Context, e *entity.{{.NameUpper}}) error {
	return r.dao.Save(ctx, e)
}

func (r *{{.NameUpper}}Repository) FindByID(ctx context.Context, id int64) (*entity.{{.NameUpper}}, error) {
	return r.dao.FindByID(ctx, id)
}
`

// EntityTmpl entity 模板
var EntityTmpl = `package entity

// {{.NameUpper}} {{.Name}} 实体
type {{.NameUpper}} struct {
	ID int64 ` + "`" + `json:"id"` + "`" + `
	// TODO: 添加字段
}
`

// DAOTmpl DAO 接口模板
var DAOTmpl = `package dao

import (
	"context"

	"{{.Name}}/internal/domain/entity"
)

// {{.NameUpper}}DAO {{.Name}} DAO 接口
type {{.NameUpper}}DAO interface {
	Save(ctx context.Context, e *entity.{{.NameUpper}}) error
	FindByID(ctx context.Context, id int64) (*entity.{{.NameUpper}}, error)
}

// {{.NameUpper}}DAOImpl 内存实现（示例）
type {{.NameUpper}}DAOImpl struct {
	data map[int64]*entity.{{.NameUpper}}
}

// New{{.NameUpper}}DAO 创建 DAO
func New{{.NameUpper}}DAO() {{.NameUpper}}DAO {
	return &{{.NameUpper}}DAOImpl{data: make(map[int64]*entity.{{.NameUpper}})}
}

func (d *{{.NameUpper}}DAOImpl) Save(ctx context.Context, e *entity.{{.NameUpper}}) error {
	d.data[e.ID] = e
	return nil
}

func (d *{{.NameUpper}}DAOImpl) FindByID(ctx context.Context, id int64) (*entity.{{.NameUpper}}, error) {
	return d.data[id], nil
}
`
