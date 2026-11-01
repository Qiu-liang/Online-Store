package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// decodeJSON 解析并限制请求体大小。
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("请求体解析失败: %w", err)
	}
	return nil
}

// errInvalidField 提示某个字段非法。
func errInvalidField(field, reason string) error {
	return &fieldError{Field: field, Reason: reason}
}

type fieldError struct {
	Field  string
	Reason string
}

func (e *fieldError) Error() string { return e.Field + ": " + e.Reason }

// writeFieldError 把字段错误转成 400 响应。
func writeFieldError(w http.ResponseWriter, err error) {
	var fe *fieldError
	if errors.As(err, &fe) {
		writeError(w, http.StatusBadRequest, "invalid_"+fe.Field, fe.Reason)
		return
	}
	writeError(w, http.StatusBadRequest, "invalid_request", "请求参数不合法")
}

// writeJSON 输出 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}

type errorPayload struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// writeError 输出统一格式的错误响应。
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorPayload{Error: errorBody{Code: code, Message: message}})
}
