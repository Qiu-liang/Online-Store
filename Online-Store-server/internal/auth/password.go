// Package auth 提供密码哈希与会话 Cookie 工具。
package auth

import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12

// HashPassword 生成 bcrypt 哈希。
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword 校验明文密码与哈希是否匹配。
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// dummyHash 是进程启动时生成的一次性哈希，用于抹平“用户不存在”分支的响应时间。
var dummyHash = func() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing-equalization"), bcryptCost)
	if err != nil {
		panic("生成哑哈希失败: " + err.Error())
	}
	return hash
}()

// DummyCompare 对哑哈希做一次等价成本的校验。登录时邮箱不存在也调用它，
// 避免攻击者通过响应时间差枚举已注册邮箱。
func DummyCompare(password string) {
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
}
