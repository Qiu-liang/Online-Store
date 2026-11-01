package api

import (
	"net/mail"
	"strings"
	"unicode/utf8"
)

// validateUsername 校验用户名：2–30 个字符。
func validateUsername(username string) bool {
	length := utf8.RuneCountInString(username)
	return length >= 2 && length <= 30
}

// validateEmail 校验邮箱格式。
func validateEmail(email string) bool {
	if len(email) > 254 {
		return false
	}
	address, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	// 只接受纯邮箱地址，不允许 "Name <a@b.c>" 形式。
	return address.Address == email && !strings.Contains(email, " ")
}

// validatePassword 校验密码长度：8–72 字节（bcrypt 上限 72 字节）。
func validatePassword(password string) bool {
	return len(password) >= 8 && len(password) <= 72
}

// validatePhone 校验手机号：6–20 位数字，可含 + - 空格。
func validatePhone(phone string) bool {
	if length := len(phone); length < 6 || length > 20 {
		return false
	}
	for _, r := range phone {
		if !(r >= '0' && r <= '9') && r != '+' && r != '-' && r != ' ' {
			return false
		}
	}
	return strings.TrimSpace(phone) != ""
}

// validateShipping 校验收货人姓名与地址。
func validateShippingName(name string) bool {
	length := utf8.RuneCountInString(strings.TrimSpace(name))
	return length >= 2 && length <= 30
}

func validateShippingAddress(address string) bool {
	length := utf8.RuneCountInString(strings.TrimSpace(address))
	return length >= 5 && length <= 200
}
