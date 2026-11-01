package api

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"online-store/server/internal/auth"
	"online-store/server/internal/models"
)

type contextKey string

const userContextKey contextKey = "user"

// currentUser 从请求上下文取出已认证用户。
func currentUser(r *http.Request) *models.User {
	user, _ := r.Context().Value(userContextKey).(*models.User)
	return user
}

// withRecover 捕获处理器 panic，返回 500。
func (s *Server) withRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("panic recovered", "error", recovered, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// withLogging 记录请求日志。
func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote", clientIP(r),
			"duration", time.Since(start).Round(time.Millisecond).String(),
		)
	})
}

// contentSecurityPolicy 覆盖前端现有资源需求：Font Awesome 走 cdnjs（样式+字体），
// 其余脚本/样式/接口均为同源。前端不得再引入内联脚本或其他外部源。
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self' https://cdnjs.cloudflare.com; " +
	"style-src 'self' https://cdnjs.cloudflare.com; " +
	"font-src https://cdnjs.cloudflare.com; " +
	"img-src 'self' data:; " +
	"connect-src 'self'; " +
	"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

// withSecureHeaders 设置基础安全响应头。
func (s *Server) withSecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
		next.ServeHTTP(w, r)
	})
}

// withSameOrigin 对写请求校验 Origin/Referer 同源，缓解 CSRF。
// 两者都缺失时（如 curl）放行，由 Cookie SameSite 兜底。
func (s *Server) withSameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")
		referer := r.Header.Get("Referer")
		if origin == "" && referer == "" {
			next.ServeHTTP(w, r)
			return
		}

		check := func(raw string) bool {
			parsed, err := url.Parse(raw)
			if err != nil || parsed.Host == "" {
				return false
			}
			if parsed.Host == r.Host {
				return true
			}
			if s.cfg.AllowedOrigin == "" {
				return false
			}
			allowed, err := url.Parse(s.cfg.AllowedOrigin)
			if err != nil {
				return false
			}
			return parsed.Host == allowed.Host
		}

		if (origin != "" && check(origin)) || (origin == "" && referer != "" && check(referer)) {
			next.ServeHTTP(w, r)
			return
		}
		writeError(w, http.StatusForbidden, "forbidden", "跨站请求被拒绝")
	})
}

// withAuth 要求有效会话，并把用户放入请求上下文。
func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(auth.CookieName)
		if err != nil || cookie.Value == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "请先登录")
			return
		}

		user, err := s.sessions.Validate(r.Context(), auth.HashToken(cookie.Value))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "登录已过期，请重新登录")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userContextKey, user)))
	}
}

// rateLimiter 是简单的固定窗口内存限流器，用于登录/注册接口。
type rateLimiter struct {
	mu        sync.Mutex
	hits      map[string][]time.Time
	limit     int
	window    time.Duration
	lastSweep time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{hits: make(map[string][]time.Time), limit: limit, window: window, lastSweep: time.Now()}
}

func (l *rateLimiter) allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	// 每过一个窗口整体清扫一次全过期条目，避免海量独立 IP 撑大内存。
	if now.Sub(l.lastSweep) >= l.window {
		for k, hits := range l.hits {
			if len(hits) == 0 || now.Sub(hits[len(hits)-1]) >= l.window {
				delete(l.hits, k)
			}
		}
		l.lastSweep = now
	}

	recent := l.hits[key][:0]
	for _, hit := range l.hits[key] {
		if now.Sub(hit) < l.window {
			recent = append(recent, hit)
		}
	}
	if len(recent) >= l.limit {
		l.hits[key] = recent
		return false
	}
	l.hits[key] = append(recent, now)
	return true
}

// withRateLimit 对登录/注册做按 IP 限流。
func (s *Server) withRateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.authLimiter.allow(clientIP(r)) {
			writeError(w, http.StatusTooManyRequests, "rate_limited", "尝试过于频繁，请稍后再试")
			return
		}
		next(w, r)
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}
