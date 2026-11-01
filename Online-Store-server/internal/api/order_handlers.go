package api

import (
	"errors"
	"net/http"
	"strings"

	"online-store/server/internal/models"
	"online-store/server/internal/store"
)

type createOrderRequest struct {
	ShippingName    string `json:"shippingName"`
	ShippingPhone   string `json:"shippingPhone"`
	ShippingAddress string `json:"shippingAddress"`
}

type orderResponse struct {
	Order *models.Order `json:"order"`
}

type orderListResponse struct {
	Orders []models.Order `json:"orders"`
}

// handleCreateOrder 用当前服务端购物车下单。
func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeFieldError(w, err)
		return
	}

	req.ShippingName = strings.TrimSpace(req.ShippingName)
	req.ShippingAddress = strings.TrimSpace(req.ShippingAddress)

	switch {
	case !validateShippingName(req.ShippingName):
		writeError(w, http.StatusBadRequest, "invalid_shipping_name", "请输入 2-30 个字符的收货人姓名")
		return
	case !validatePhone(strings.TrimSpace(req.ShippingPhone)):
		writeError(w, http.StatusBadRequest, "invalid_shipping_phone", "请输入有效的联系电话")
		return
	case !validateShippingAddress(req.ShippingAddress):
		writeError(w, http.StatusBadRequest, "invalid_shipping_address", "请输入 5-200 个字符的收货地址")
		return
	}

	shipping := models.Shipping{
		Name:    req.ShippingName,
		Phone:   strings.TrimSpace(req.ShippingPhone),
		Address: req.ShippingAddress,
	}

	order, err := s.orders.CreateFromCart(r.Context(), currentUser(r).ID, shipping)
	if errors.Is(err, store.ErrEmptyCart) {
		writeError(w, http.StatusBadRequest, "cart_empty", "购物车是空的，无法下单")
		return
	}
	if err != nil {
		s.logger.Error("创建订单失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusCreated, orderResponse{Order: order})
}

// handleListOrders 当前用户订单列表。
func (s *Server) handleListOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := s.orders.ListByUser(r.Context(), currentUser(r).ID)
	if err != nil {
		s.logger.Error("查询订单失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusOK, orderListResponse{Orders: orders})
}

// handleGetOrder 订单详情（校验归属）。
func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	order, err := s.orders.GetByID(r.Context(), currentUser(r).ID, r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "order_not_found", "订单不存在")
		return
	}
	if err != nil {
		s.logger.Error("查询订单失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusOK, orderResponse{Order: order})
}
