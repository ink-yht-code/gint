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

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/registry/api"
	"github.com/ink-yht-code/gint/registry/store"
)

var (
	port     = flag.Int("port", 18080, "HTTP port")
	dataFile = flag.String("data", "registry.db", "SQLite database file")
	token    = flag.String("token", "", "Authentication token")
)

func main() {
	flag.Parse()

	// 初始化存储
	s, err := store.NewSQLiteStore(*dataFile)
	if err != nil {
		fmt.Printf("Failed to init store: %v\n", err)
		os.Exit(1)
	}

	// 初始化 Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// 认证中间件（可选）
	if *token != "" {
		r.Use(func(c *gin.Context) {
			auth := c.GetHeader("Authorization")
			if auth != "Bearer "+*token {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				c.Abort()
				return
			}
			c.Next()
		})
	}

	// 注册路由
	h := api.NewHandler(s)
	h.RegisterRoutes(r)

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 启动服务器
	addr := fmt.Sprintf(":%d", *port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		fmt.Printf("Registry server listening on %s\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Failed to start server: %v\n", err)
			os.Exit(1)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced to shutdown: %v\n", err)
	}

	fmt.Println("Server exited")
}
