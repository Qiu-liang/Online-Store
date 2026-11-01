package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"online-store/server/internal/auth"
	"online-store/server/internal/models"
	"online-store/server/internal/store"
)

type adminLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// handleAdminLogin 管理员登录：密码正确且角色为 admin 才签发会话（管理员 TTL）。
func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var req adminLoginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeFieldError(w, err)
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "请输入邮箱和密码")
		return
	}

	user, passwordHash, err := s.users.GetByEmail(r.Context(), email)
	if errors.Is(err, store.ErrNotFound) {
		auth.DummyCompare(req.Password)
		s.logger.Warn("管理员登录失败：邮箱不存在", "email", email)
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "邮箱或密码不正确")
		return
	}
	if err != nil {
		s.logger.Error("查询用户失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}

	if !auth.CheckPassword(passwordHash, req.Password) {
		s.logger.Warn("管理员登录失败：密码错误", "email", email)
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "邮箱或密码不正确")
		return
	}
	if user.Role != models.RoleAdmin {
		// 密码正确但不是管理员：拒绝且不创建会话，不影响浏览器中已有的客户会话。
		s.logger.Warn("管理员登录被拒绝：非管理员账号", "email", email)
		writeError(w, http.StatusForbidden, "not_admin", "当前账号没有管理员权限")
		return
	}

	if err := s.startSession(w, r, user.ID, s.cfg.AdminSessionTTL); err != nil {
		s.logger.Error("创建管理员会话失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	s.logger.Info("管理员登录成功", "adminId", user.ID, "email", user.Email)
	writeJSON(w, http.StatusOK, userResponse{User: user})
}

// handleAdminMe 返回当前管理员信息。
func (s *Server) handleAdminMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, userResponse{User: currentUser(r)})
}

// handleAdminStats 后台概览：统计卡片 + 最近 5 笔订单。
func (s *Server) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.stats.Overview(r.Context())
	if err != nil {
		s.logger.Error("统计概览失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}

	recent, _, err := s.orders.AdminList(r.Context(), store.AdminOrderFilter{Limit: 5, Offset: 0})
	if err != nil {
		s.logger.Error("查询最近订单失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stats": stats, "recentOrders": recent})
}

// logAdminAction 记录后台关键写操作（不含密码、哈希、完整收货地址）。
func (s *Server) logAdminAction(r *http.Request, action, target string) {
	admin := currentUser(r)
	s.logger.Info("admin action",
		"adminId", admin.ID,
		"adminEmail", admin.Email,
		"action", action,
		"target", target,
		"time", time.Now().UTC().Format(time.RFC3339),
	)
}
