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

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	User *models.User `json:"user"`
}

// handleRegister 注册新用户并自动登录（签发会话）。
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeFieldError(w, err)
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	switch {
	case !validateUsername(req.Username):
		writeError(w, http.StatusBadRequest, "invalid_username", "用户名需要 2-30 个字符")
		return
	case !validateEmail(req.Email):
		writeError(w, http.StatusBadRequest, "invalid_email", "邮箱格式不正确")
		return
	case !validatePassword(req.Password):
		writeError(w, http.StatusBadRequest, "invalid_password", "密码长度需要 8-72 个字符")
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.logger.Error("密码哈希失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}

	user, err := s.users.Create(r.Context(), req.Username, req.Email, passwordHash)
	if errors.Is(err, store.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "email_taken", "该邮箱已注册，请直接登录")
		return
	}
	if err != nil {
		s.logger.Error("创建用户失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}

	if err := s.startSession(w, r, user.ID, s.cfg.SessionTTL); err != nil {
		s.logger.Error("创建会话失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusCreated, userResponse{User: user})
}

// handleLogin 邮箱 + 密码登录。
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
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
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "邮箱或密码不正确")
		return
	}
	if err != nil {
		s.logger.Error("查询用户失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}

	if !auth.CheckPassword(passwordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "邮箱或密码不正确")
		return
	}

	if err := s.startSession(w, r, user.ID, s.cfg.SessionTTL); err != nil {
		s.logger.Error("创建会话失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusOK, userResponse{User: user})
}

// handleLogout 删除当前会话并清除 Cookie。
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(auth.CookieName); err == nil && cookie.Value != "" {
		if err := s.sessions.Delete(r.Context(), auth.HashToken(cookie.Value)); err != nil {
			s.logger.Error("删除会话失败", "error", err)
		}
	}
	auth.ClearCookie(w, s.cfg.CookieSecure)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleMe 返回当前登录用户。
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, userResponse{User: currentUser(r)})
}

// startSession 创建会话记录并写入 Cookie；管理员登录使用更短的 TTL。
func (s *Server) startSession(w http.ResponseWriter, r *http.Request, userID int64, ttl time.Duration) error {
	token, tokenHash, err := auth.NewToken()
	if err != nil {
		return err
	}
	if err := s.sessions.Create(r.Context(), userID, tokenHash, time.Now().Add(ttl)); err != nil {
		return err
	}
	auth.SetCookie(w, token, ttl, s.cfg.CookieSecure)
	return nil
}
