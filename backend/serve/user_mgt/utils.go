package user_mgt

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"w2w.io/null"
)

// NullableString 创建一个 null.String 对象
// 自动根据字符串是否为空来设置 Valid 字段
func NullableString(s string) null.String {
	return null.NewString(s, strings.TrimSpace(s) != "")
}

// NullableIntFromStr 由 string 类型创建一个 null.Int 对象
// 自动根据字符串是否为空或能否合法转换为 int64 来设置 Valid 字段
func NullableIntFromStr(s string) null.Int {
	// 检查是否包含空格（包含半角/全角空格）
	for _, r := range s {
		if unicode.IsSpace(r) {
			return null.Int{}
		}
	}

	// 去除首尾空格（安全措施，可选）
	s = strings.TrimSpace(s)
	if s == "" {
		return null.Int{}
	}

	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return null.Int{}
	}

	return null.NewInt(i, true)
}

// IsValidEmail 判断邮箱地址是否合法
func IsValidEmail(email string) bool {
	// RFC 5322 标准的简化版正则表达式
	var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// IsDomainExist 判断角色是否合法
func IsDomainExist(domain string) bool {
	_, ok := domainSet[domain]
	return ok
}

// Contains 泛型函数，判断元素是否存在于切片中
func Contains[T comparable](item T, list []T) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}
