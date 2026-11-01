package api

import (
	"errors"
	"net/http"

	"online-store/server/internal/store"
)

// handleHealth 健康检查。
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleListCategories 分类列表。
func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := s.products.ListCategories(r.Context())
	if err != nil {
		s.logger.Error("查询分类失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": categories})
}

// handleListProducts 商品列表，可按 category 查询参数过滤。
func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	products, err := s.products.List(r.Context(), category)
	if err != nil {
		s.logger.Error("查询商品失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": products})
}

// handleGetProduct 商品详情。下架商品视为不存在，提示“已下架”。
func (s *Server) handleGetProduct(w http.ResponseWriter, r *http.Request) {
	product, err := s.products.GetByID(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "product_not_found", "商品不存在或已下架")
		return
	}
	if err != nil {
		s.logger.Error("查询商品失败", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"product": product})
}
