package api

import (
	"errors"
	"net/http"
	"strconv"

	"online-store/server/internal/models"
	"online-store/server/internal/store"
)

type adminOrderListResponse struct {
	Orders   []models.AdminOrder `json:"orders"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"pageSize"`
}

type adminUpdateStatusRequest struct {
	Status string `json:"status"`
}

// handleAdminListOrders 后台订单列表：status/query/userId + 分页。
func (s *Server) handleAdminListOrders(w http.ResponseWriter, r *http.Request) {
	page := parsePagination(r)

	userID := int64(0)
	if raw := r.URL.Query().Get("userId"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "invalid_user_id", "用户ID不合法")
			return
		}
		userID = parsed
	}

	filter := store.AdminOrderFilter{
		Status: r.URL.Query().Get("status"),
		Query:  r.URL.Query().Get("query"),
		UserID: userID,
		Limit:  page.PageSize,
		Offset: page.offset(),
	}

	orders, total, err := s.orders.AdminList(r.Context(), filter)
	if err != nil {
		s.logger.Error("后台查询订单失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusOK, adminOrderListResponse{
		Orders: orders, Total: total, Page: page.Page, PageSize: page.PageSize,
	})
}

// handleAdminGetOrder 后台订单详情（不校验归属，含明细）。
func (s *Server) handleAdminGetOrder(w http.ResponseWriter, r *http.Request) {
	order, err := s.orders.AdminGetByID(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "order_not_found", "订单不存在")
		return
	}
	if err != nil {
		s.logger.Error("后台查询订单失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"order": order})
}

// handleAdminUpdateOrderStatus 修改订单状态；五种状态间自由切换，便于人工纠错。
func (s *Server) handleAdminUpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	var req adminUpdateStatusRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeFieldError(w, err)
		return
	}
	if !models.IsValidOrderStatus(req.Status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "订单状态不合法")
		return
	}

	order, err := s.orders.UpdateStatus(r.Context(), r.PathValue("id"), req.Status)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "order_not_found", "订单不存在")
		return
	}
	if err != nil {
		s.logger.Error("更新订单状态失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	s.logAdminAction(r, "order.update_status", order.ID+"->"+req.Status)
	writeJSON(w, http.StatusOK, map[string]any{"order": order})
}

// handleAdminDeleteOrder 删除订单；order_items 级联删除。
func (s *Server) handleAdminDeleteOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.orders.Delete(r.Context(), id); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "order_not_found", "订单不存在")
		return
	} else if err != nil {
		s.logger.Error("删除订单失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	s.logAdminAction(r, "order.delete", id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
