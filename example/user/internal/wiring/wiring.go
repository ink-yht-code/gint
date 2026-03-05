package wiring

import (
	"context"

	"github.com/ink-yht-code/gint/gintx/httpx"
	"github.com/ink-yht-code/gint/gintx/log"
	"go.uber.org/zap"

	"example/user/internal/config"
	"example/user/internal/server"
	"example/user/internal/web"
)

// LoadConfig 加载配置
func LoadConfig(path string) (*config.Config, error) {
	return config.Load(path)
}

// App 应用
type App struct {
	cfg  *config.Config
	http *httpx.Server
	svc  *server.UserService
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
	svc := server.NewUserService()

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

	return &App{
		cfg:  cfg,
		http: httpServer,
		svc:  svc,
	}, nil
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
