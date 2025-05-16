// Package main 启动小红书签名 HTTP 服务。
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go_sign/internal/xhs"
)

func main() {
	// 设置 slog 日志格式和级别
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))

	// 解析配置
	stealthPath := flag.String("stealth", "./stealth.min.js", "stealth.min.js 文件路径")
	addr := flag.String("addr", ":5005", "HTTP 监听地址")
	flag.Parse()

	slog.Info("启动参数", "stealth_path", *stealthPath, "addr", *addr)

	// 初始化签名服务
	signer, err := xhs.NewSigner(context.Background(), *stealthPath)
	if err != nil {
		slog.Error("初始化签名服务失败", "err", err, "stealth_path", *stealthPath)
		os.Exit(1)
	}

	r := gin.New()
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 返回格式化字符串，便于日志采集
		return fmt.Sprintf("[GIN] %s %s %s %s\n", param.Method, param.Path, param.ClientIP, param.ErrorMessage)
	}))
	r.Use(gin.Recovery())

	xhs.RegisterRoutes(r, signer)

	// 用 http.Server 包裹 gin 实例，实现优雅关闭
	srv := &http.Server{
		Addr:    *addr,
		Handler: r,
	}

	// 启动 HTTP 服务（协程）
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("服务启动失败", "err", err)
			os.Exit(1)
		}
	}()

	slog.Info("服务启动", "addr", *addr)

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("收到退出信号，正在关闭服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("HTTP 服务优雅关闭失败", "err", err)
	}
	if err := signer.Close(); err != nil {
		slog.Error("关闭 Playwright 资源失败", "err", err)
	} else {
		slog.Info("Playwright 资源已成功关闭")
	}
}
