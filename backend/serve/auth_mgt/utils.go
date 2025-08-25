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

// DomainLevel 解析传入的域并返回层级数
// 例如：cst.school -> 2，cst.school.class -> 3
// 如果域不合法则返回0
func DomainLevel(domain string) int {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return 0
	}

	// 先校验合法性
	if !IsValidDomain(domain) {
		return 0
	}

	labels := strings.Split(domain, ".")
	return len(labels)
}
