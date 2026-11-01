// 命令 server 启动绿源农业后端服务。
package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"online-store/server/internal/api"
	"online-store/server/internal/auth"
	"online-store/server/internal/config"
	"online-store/server/internal/database"
	"online-store/server/internal/models"
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

	bootstrapAdmin(logger, cfg, store.NewUserStore(db))

	server := api.NewServer(
		cfg,
		logger,
		store.NewUserStore(db),
		sessionStore,
		store.NewProductStore(db),
		store.NewCartStore(db),
		store.NewOrderStore(db),
		store.NewStatsStore(db),
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
			"admin", "http://localhost:"+cfg.Port+"/admin",
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

// bootstrapAdmin 确保初始管理员存在：
//   - 邮箱不存在：创建管理员；密码取 ADMIN_PASSWORD，为空则随机生成并只在本次日志打印。
//   - 已存在且是管理员：跳过，不修改密码。
//   - 已存在但是普通用户：告警并不自动提权，避免误把普通用户变成管理员。
//
// 初始化失败只记录错误，不阻断商城服务启动。
func bootstrapAdmin(logger *slog.Logger, cfg *config.Config, users *store.UserStore) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	user, _, err := users.GetByEmail(ctx, cfg.AdminEmail)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		logger.Error("查询初始管理员失败", "error", err)
		return
	}
	if user != nil {
		if user.Role == models.RoleAdmin {
			logger.Info("初始管理员已存在", "email", cfg.AdminEmail)
			return
		}
		logger.Warn("ADMIN_EMAIL 对应账号已存在但不是管理员，未自动提权；请更换 ADMIN_EMAIL 或在数据库中调整角色",
			"email", cfg.AdminEmail)
		return
	}

	password := cfg.AdminPassword
	randomPassword := password == ""
	if randomPassword {
		generated, err := generateRandomPassword(16)
		if err != nil {
			logger.Error("生成随机管理员密码失败", "error", err)
			return
		}
		password = generated
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		logger.Error("管理员密码哈希失败", "error", err)
		return
	}
	if _, err := users.CreateAdmin(ctx, "管理员", cfg.AdminEmail, passwordHash); err != nil {
		logger.Error("创建初始管理员失败", "error", err)
		return
	}
	logger.Info("已创建初始管理员", "email", cfg.AdminEmail)
	if randomPassword {
		logger.Info("初始管理员随机密码（仅此一次打印，请立即保存并尽快修改）", "password", password)
	}
}

// generateRandomPassword 生成不带易混淆字符的随机密码。
func generateRandomPassword(length int) (string, error) {
	const charset = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	password := make([]byte, length)
	max := big.NewInt(int64(len(charset)))
	for i := range password {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("生成随机数: %w", err)
		}
		password[i] = charset[n.Int64()]
	}
	return string(password), nil
}
