package api

import (
	"errors"
	"net/http"
	"strconv"

	"online-store/server/internal/auth"
	"online-store/server/internal/models"
	"online-store/server/internal/store"
)

type adminUserListResponse struct {
	Users    []models.AdminUser `json:"users"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
}

type adminUpdateRoleRequest struct {
	Role string `json:"role"`
}

type adminResetPasswordRequest struct {
	Password string `json:"password"`
}

// parseUserPathID 解析用户路径参数，非法返回 0（调用方写 400 响应）。
func parseUserPathID(r *http.Request) int64 {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

// handleAdminListUsers 后台用户列表：query/role + 分页，附订单统计。
func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	page := parsePagination(r)
	filter := store.UserFilter{
		Query:  r.URL.Query().Get("query"),
		Role:   r.URL.Query().Get("role"),
		Limit:  page.PageSize,
		Offset: page.offset(),
	}

	users, total, err := s.users.ListAdmin(r.Context(), filter)
	if err != nil {
		s.logger.Error("后台查询用户失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusOK, adminUserListResponse{
		Users: users, Total: total, Page: page.Page, PageSize: page.PageSize,
	})
}

// handleAdminUpdateRole 调整用户角色；不能改自己，不能降级最后一名管理员。
func (s *Server) handleAdminUpdateRole(w http.ResponseWriter, r *http.Request) {
	id := parseUserPathID(r)
	if id == 0 {
		writeError(w, http.StatusBadRequest, "invalid_user_id", "用户ID不合法")
		return
	}

	var req adminUpdateRoleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeFieldError(w, err)
		return
	}
	if !models.IsValidRole(req.Role) {
		writeError(w, http.StatusBadRequest, "invalid_role", "角色只支持 user / admin")
		return
	}

	target, err := s.users.GetByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user_not_found", "用户不存在")
		return
	}
	if err != nil {
		s.logger.Error("查询用户失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}

	// 最后管理员保护先于自助操作保护：唯一管理员降级（含降级自己）都返回 409。
	if target.Role == models.RoleAdmin && req.Role == models.RoleUser {
		admins, err := s.users.CountAdmins(r.Context())
		if err != nil {
			s.logger.Error("统计管理员失败", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
			return
		}
		if admins <= 1 {
			writeError(w, http.StatusConflict, "last_admin", "系统至少需要保留一名管理员")
			return
		}
	}
	if id == currentUser(r).ID {
		writeError(w, http.StatusBadRequest, "self_operation_forbidden", "不能修改自己的角色")
		return
	}

	user, err := s.users.UpdateRole(r.Context(), id, req.Role)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user_not_found", "用户不存在")
		return
	}
	if err != nil {
		s.logger.Error("更新用户角色失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	s.logAdminAction(r, "user.update_role", strconv.FormatInt(id, 10)+"->"+req.Role)
	writeJSON(w, http.StatusOK, userResponse{User: user})
}

// handleAdminResetPassword 重置用户密码；该用户全部会话立即失效。
func (s *Server) handleAdminResetPassword(w http.ResponseWriter, r *http.Request) {
	id := parseUserPathID(r)
	if id == 0 {
		writeError(w, http.StatusBadRequest, "invalid_user_id", "用户ID不合法")
		return
	}

	var req adminResetPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeFieldError(w, err)
		return
	}
	if !validatePassword(req.Password) {
		writeError(w, http.StatusBadRequest, "invalid_password", "密码长度需要 8-72 个字符")
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.logger.Error("密码哈希失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	if err := s.users.ResetPassword(r.Context(), id, passwordHash); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user_not_found", "用户不存在")
		return
	} else if err != nil {
		s.logger.Error("重置密码失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	s.logAdminAction(r, "user.reset_password", strconv.FormatInt(id, 10))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleAdminDeleteUser 删除用户：级联删除会话/购物车/订单；不能删自己与最后一名管理员。
func (s *Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := parseUserPathID(r)
	if id == 0 {
		writeError(w, http.StatusBadRequest, "invalid_user_id", "用户ID不合法")
		return
	}

	target, err := s.users.GetByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user_not_found", "用户不存在")
		return
	}
	if err != nil {
		s.logger.Error("查询用户失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}

	// 最后管理员保护先于自助操作保护：唯一管理员（含自己）都返回 409。
	if target.Role == models.RoleAdmin {
		admins, err := s.users.CountAdmins(r.Context())
		if err != nil {
			s.logger.Error("统计管理员失败", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
			return
		}
		if admins <= 1 {
			writeError(w, http.StatusConflict, "last_admin", "系统至少需要保留一名管理员")
			return
		}
	}
	if id == currentUser(r).ID {
		writeError(w, http.StatusBadRequest, "self_operation_forbidden", "不能删除自己")
		return
	}

	if err := s.users.Delete(r.Context(), id); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user_not_found", "用户不存在")
		return
	} else if err != nil {
		s.logger.Error("删除用户失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	s.logAdminAction(r, "user.delete", strconv.FormatInt(id, 10))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
