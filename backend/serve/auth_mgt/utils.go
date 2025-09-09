package auth_mgt

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"w2w.io/cmn"
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

// IsValidDomainOrRole 检查字符串是域还是带角色的域：
// - "cst.school"  → 合法域
// - "cst.school^user" → 合法域+合法角色
func IsValidDomainOrRole(s string) (isDomain bool, isRole bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return false, false
	}

	parts := strings.Split(s, "^")
	if len(parts) == 1 {
		// 只有域
		return IsValidDomain(parts[0]), false
	} else if len(parts) == 2 {
		// 域 + 角色
		if !IsValidDomain(parts[0]) {
			return false, false
		}
		role := parts[1]
		// 角色名规则：字母数字下划线，且至少一个字符
		roleRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
		if !roleRegex.MatchString(role) {
			return false, false
		}
		return false, true
	}
	// 出现多个 ^ 不合法
	return false, false
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

// ParseFirstDomain 解析第一个域
// 规则：
// - "cst.school.class"       -> "cst"
// - "cst.school.class^admin" -> "cst"
// - "cst"                    -> ""
// - "cst^admin"              -> ""
// - ""                       -> ""
func ParseFirstDomain(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// 先去掉角色部分
	if idx := strings.Index(s, "^"); idx != -1 {
		s = s[:idx]
	}

	// 拆分域
	parts := strings.Split(s, ".")
	if len(parts) <= 1 {
		return "" // 只有一级域时返回空字符串
	}

	return parts[0]
}

// validateSelectedAPIs 校验选择的API是否合法
// 检查用户选择的API是否在可选的API列表中
func validateSelectedAPIs(ctx context.Context, selectedAPIs []*cmn.TAPI, parentDomain string) error {
	// 查询可选的API列表
	selectableAPIs, err := querySelectableAPIs(ctx, parentDomain)
	if err != nil {
		return fmt.Errorf("failed to query selectable APIs: %w", err)
	}

	// 创建可选API的ID映射，用于快速查找
	selectableAPIMap := make(map[int64]bool)
	for _, api := range selectableAPIs {
		if api.ID.Valid {
			selectableAPIMap[api.ID.Int64] = true
		}
	}

	// 检查每个选择的API是否在可选列表中
	for _, selectedAPI := range selectedAPIs {
		if selectedAPI.ID.Valid && !selectableAPIMap[selectedAPI.ID.Int64] {
			return fmt.Errorf("API ID %d is not in the selectable API list", selectedAPI.ID.Int64)
		}
	}

	return nil
}
