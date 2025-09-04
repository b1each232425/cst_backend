package user_mgt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx/types"
	"go.uber.org/zap"
	"w2w.io/cmn"
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

// IsDomainExist 判断 domain 是否存在于 t_domain 表，如果存在则返回对应记录
func IsDomainExist(ctx context.Context, tx pgx.Tx, domain string) (bool, *cmn.TDomain, error) {
	query := `SELECT id, name, domain, priority 
	          FROM t_domain WHERE domain = $1 LIMIT 1`

	var row pgx.Row
	if tx != nil {
		row = tx.QueryRow(ctx, query, domain)
	} else {
		pgConn := cmn.GetPgxConn()
		row = pgConn.QueryRow(ctx, query, domain)
	}

	d := &cmn.TDomain{}
	err := row.Scan(&d.ID, &d.Name, &d.Domain, &d.Priority)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// 没查到，返回不存在
			return false, nil, nil
		}
		z.Error("failed to query domain:", zap.Error(err))
		return false, nil, err
	}

	return true, d, nil
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

// NormalizeAndValidateCNID 接受 15 或 18 位中国居民身份证号。
// 1) 自动去除空白（含中间空格）
// 2) 若为 18 位：校验日期与校验位，返回统一大写 X 的 18 位号码
// 3) 若为 15 位：转为 18 位（补 "19" 年份并重算校验位），同样校验日期后返回
func NormalizeAndValidateCNID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", errors.New("empty id")
	}

	// 移除所有空白字符（若不希望自动去掉中间空格，可去除此段）
	id = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, id)

	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	checkMap := []byte("10X98765432")
	dateMin, _ := time.Parse("20060102", "19000101")

	// 小工具：校验 YYYYMMDD 合法且在 [1900-01-01, 今天] 区间
	validDate := func(yyyymmdd string) bool {
		if len(yyyymmdd) != 8 {
			return false
		}
		t, err := time.Parse("20060102", yyyymmdd)
		if err != nil {
			return false
		}
		if t.Before(dateMin) || t.After(time.Now()) {
			return false
		}
		return true
	}

	switch len(id) {
	case 18:
		// 前17位必须数字
		sum := 0
		for i := 0; i < 17; i++ {
			c := id[i]
			if c < '0' || c > '9' {
				return "", errors.New("invalid char in first 17 digits")
			}
			sum += int(c-'0') * weights[i]
		}
		// 生日位 7-14
		birth := id[6:14]
		if !validDate(birth) {
			return "", errors.New("invalid birth date")
		}
		// 校验位
		mod := sum % 11
		exp := checkMap[mod]
		last := id[17]
		if !((last >= '0' && last <= '9') || last == 'X' || last == 'x') {
			return "", errors.New("invalid last char")
		}
		got := byte(last)
		if got == 'x' {
			got = 'X'
		}
		if got != exp {
			return "", errors.New("checksum mismatch")
		}
		return strings.ToUpper(id), nil

	case 15:
		// 15位必须全数字
		for i := 0; i < 15; i++ {
			if id[i] < '0' || id[i] > '9' {
				return "", errors.New("invalid char in 15-digit id")
			}
		}
		prefix6 := id[:6]
		birth6 := id[6:12] // yyMMdd
		rest3 := id[12:15] // 顺序码
		birth := "19" + birth6
		if !validDate(birth) {
			return "", errors.New("invalid birth date in 15-digit id")
		}
		base17 := prefix6 + birth + rest3
		sum := 0
		for i := 0; i < 17; i++ {
			sum += int(base17[i]-'0') * weights[i]
		}
		mod := sum % 11
		checkChar := checkMap[mod]
		return base17 + string(checkChar), nil

	default:
		return "", errors.New("id length must be 15 or 18")
	}
}

// ParseAndValidateIDCardFile 解析并校验 IDCardFile JSON 字段
func ParseAndValidateIDCardFile(raw types.JSONText) (IDCardFile, error) {
	var v IDCardFile
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" || s == "{}" || s == "[]" {
		return v, fmt.Errorf("id_card_file is empty")
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields() // 发现未知字段名直接报错
	if err := dec.Decode(&v); err != nil {
		return v, fmt.Errorf("invalid id_card_file json: %w", err)
	}
	// 额外防御：确保没有多余数据
	if dec.More() {
		return v, fmt.Errorf("invalid id_card_file: trailing data")
	}

	// 必填校验（根据你的业务改）
	if v.FrontImgID == "" {
		return v, fmt.Errorf("missing frontImgID")
	}
	if v.BackImgID == "" {
		return v, fmt.Errorf("missing backImgID")
	}
	return v, nil
}
