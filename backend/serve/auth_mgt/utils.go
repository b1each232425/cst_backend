package auth_mgt

import (
	"regexp"
	"strings"
)

// IsValidDomain 检查传入的域是否符合合法格式：
// - 至少包含一个 label
// - 多个 label 用 '.' 分隔
// - 每个 label 由字母、数字、短横线组成，不能以 '-' 开头或结尾
func IsValidDomain(domain string) bool {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return false
	}

	// 拆分
	labels := strings.Split(domain, ".")
	labelRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
	for _, label := range labels {
		if !labelRegex.MatchString(label) {
			return false
		}
	}

	return true
}
