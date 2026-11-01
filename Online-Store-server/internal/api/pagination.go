package api

import (
	"net/http"
	"strconv"
	"strings"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

// pagination 是夹取后的分页参数。
type pagination struct {
	Page     int
	PageSize int
}

// offset 计算 SQL 偏移量。
func (p pagination) offset() int {
	return (p.Page - 1) * p.PageSize
}

// parsePagination 解析 page / pageSize 查询参数：
// page 默认 1，pageSize 默认 20、最大 100；非法或越界一律夹取，不报错。
func parsePagination(r *http.Request) pagination {
	page := clampInt(r.URL.Query().Get("page"), defaultPage, 1, 1_000_000)
	pageSize := clampInt(r.URL.Query().Get("pageSize"), defaultPageSize, 1, maxPageSize)
	return pagination{Page: page, PageSize: pageSize}
}

// clampInt 解析整数并夹取到 [min, max]；空值用 fallback，非法值视为 fallback。
func clampInt(raw string, fallback, min, max int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
