// Package config 负责读取环境变量与可选的 .env 文件。
package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 是服务端运行所需的全部配置。
type Config struct {
	Port          string        // HTTP 监听端口
	FrontendDir   string        // 静态前端目录（Online-Store-main）
	DBPath        string        // SQLite 数据库文件路径
	SessionTTL    time.Duration // 会话有效期
	CookieSecure  bool          // 是否为会话 Cookie 开启 Secure（生产环境 HTTPS 下开启）
	AllowedOrigin string        // 可选：允许跨域携带 Cookie 的来源，如 http://localhost:5500
	Env           string        // development / production
}

// Load 读取配置：先加载 .env（已存在的环境变量优先），再应用默认值。
func Load() (*Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return nil, err
	}

	hours, err := strconv.Atoi(strings.TrimSpace(getEnv("SESSION_TTL_HOURS", "168")))
	if err != nil || hours <= 0 {
		hours = 168
	}

	return &Config{
		Port:          getEnv("PORT", "8080"),
		FrontendDir:   getEnv("FRONTEND_DIR", "../Online-Store-main"),
		DBPath:        getEnv("DB_PATH", "./data/store.db"),
		SessionTTL:    time.Duration(hours) * time.Hour,
		CookieSecure:  strings.EqualFold(getEnv("COOKIE_SECURE", "false"), "true"),
		AllowedOrigin: strings.TrimSpace(getEnv("ALLOWED_ORIGIN", "")),
		Env:           getEnv("APP_ENV", "development"),
	}, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// loadDotEnv 逐行读取 KEY=VALUE，不覆盖已存在的环境变量。
func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
	return scanner.Err()
}
