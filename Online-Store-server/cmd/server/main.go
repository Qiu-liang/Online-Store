// 命令 server 启动绿源农业后端服务。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"online-store/server/internal/api"
	"online-store/server/internal/config"
	"online-store/server/internal/database"
	"online-store/server/internal/store"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("读取配置失败", "error", err)
		os.Exit(1)
	}

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		logger.Error("初始化数据库失败", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 启动时清理一次过期会话，之后每小时定时清理。
	sessionStore := store.NewSessionStore(db)
	if err := sessionStore.DeleteExpired(context.Background()); err != nil {
		logger.Warn("清理过期会话失败", "error", err)
	}

	server := api.NewServer(
		cfg,
		logger,
		store.NewUserStore(db),
		sessionStore,
		store.NewProductStore(db),
		store.NewCartStore(db),
		store.NewOrderStore(db),
	)

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("服务已启动",
			"addr", "http://localhost:"+cfg.Port,
			"frontend", cfg.FrontendDir,
			"db", cfg.DBPath,
			"env", cfg.Env,
		)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("服务异常退出", "error", err)
			stop()
		}
	}()

	// 每小时清理一次过期会话。
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := sessionStore.DeleteExpired(context.Background()); err != nil {
					logger.Warn("清理过期会话失败", "error", err)
				}
			}
		}
	}()

	<-ctx.Done()
	logger.Info("正在关闭服务…")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("关闭服务失败", "error", err)
	}
	logger.Info("服务已停止")
}
