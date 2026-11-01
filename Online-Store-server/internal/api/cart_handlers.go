package api

import (
	"errors"
	"net/http"

	"online-store/server/internal/models"
	"online-store/server/internal/store"
)

type addCartItemRequest struct {
	ProductID string `json:"productId"`
	Quantity  int64  `json:"quantity"`
}

type updateCartItemRequest struct {
	Quantity int64 `json:"quantity"`
}

type mergeCartRequest struct {
	Items []models.MergeItem `json:"items"`
}

type cartResponse struct {
	Cart *models.Cart `json:"cart"`
}

// handleGetCart 当前用户购物车。
func (s *Server) handleGetCart(w http.ResponseWriter, r *http.Request) {
	cart, err := s.carts.GetCart(r.Context(), currentUser(r).ID)
	if err != nil {
		s.logger.Error("查询购物车失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusOK, cartResponse{Cart: cart})
}

// handleAddCartItem 添加商品到购物车。
func (s *Server) handleAddCartItem(w http.ResponseWriter, r *http.Request) {
	var req addCartItemRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeFieldError(w, err)
		return
	}
	if req.ProductID == "" {
		writeError(w, http.StatusBadRequest, "invalid_product", "缺少商品ID")
		return
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	cart, err := s.carts.AddItem(r.Context(), currentUser(r).ID, req.ProductID, req.Quantity)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "product_not_found", "商品不存在")
		return
	}
	if err != nil {
		s.logger.Error("写入购物车失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusOK, cartResponse{Cart: cart})
}

// handleUpdateCartItem 修改数量。
func (s *Server) handleUpdateCartItem(w http.ResponseWriter, r *http.Request) {
	var req updateCartItemRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeFieldError(w, err)
		return
	}
	if req.Quantity < 1 || req.Quantity > 99 {
		writeError(w, http.StatusBadRequest, "invalid_quantity", "数量需要在 1-99 之间")
		return
	}

	cart, err := s.carts.UpdateQuantity(r.Context(), currentUser(r).ID, r.PathValue("productId"), req.Quantity)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "cart_item_not_found", "购物车中没有该商品")
		return
	}
	if err != nil {
		s.logger.Error("更新购物车失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusOK, cartResponse{Cart: cart})
}

// handleRemoveCartItem 删除购物车商品。
func (s *Server) handleRemoveCartItem(w http.ResponseWriter, r *http.Request) {
	cart, err := s.carts.RemoveItem(r.Context(), currentUser(r).ID, r.PathValue("productId"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "cart_item_not_found", "购物车中没有该商品")
		return
	}
	if err != nil {
		s.logger.Error("删除购物车失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusOK, cartResponse{Cart: cart})
}

// handleMergeCart 合并游客购物车，忽略不存在的商品ID。
func (s *Server) handleMergeCart(w http.ResponseWriter, r *http.Request) {
	var req mergeCartRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeFieldError(w, err)
		return
	}

	cart, err := s.carts.MergeItems(r.Context(), currentUser(r).ID, req.Items)
	if err != nil {
		s.logger.Error("合并购物车失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusOK, cartResponse{Cart: cart})
}
