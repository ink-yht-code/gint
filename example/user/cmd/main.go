package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example/user/internal/wiring"
)

func main() {
	// 加载配置
	cfg, err := wiring.LoadConfig("configs/user.yaml")
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

	fmt.Printf("Service %s started\n", cfg.Service.Name)

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
