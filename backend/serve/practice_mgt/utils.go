/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 请在此填写文件描述
 * @Date: 2025-07-15 16:40:38
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-08-27 13:46:06
 */
package practice_mgt

import (
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx/types"
	"reflect"
	"strings"
	"w2w.io/cmn"
	"w2w.io/null"
)

type JSONText = types.JSONText
type Map map[string]interface{}

func S2Map(in interface{}) Map {
	data := make(Map)
	v := reflect.ValueOf(in).Elem() // 获取结构体指针指向的值
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// 跳过零值字段（如未设置的 null.Int）
		if reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface()) {
			continue
		}

		// 解析 db 标签获取列名（如 "name,false,character varying" -> "name"）
		dbTag := field.Tag.Get("db")
		if dbTag == "" || strings.Contains(dbTag, "true") { // 跳过主键或忽略字段
			continue
		}
		columnName := strings.Split(dbTag, ",")[0]

		// 处理特殊类型（如 null.Int, types.JSONText）
		switch v := value.Interface().(type) {
		case null.Int:
			if v.Valid {
				data[columnName] = v.Int64
			}
		case null.String:
			if v.Valid {
				data[columnName] = v.String
			}
		case types.JSONText:
			if len(v) > 0 { // JSONText 非空
				data[columnName] = v
			}
		default:
			data[columnName] = value.Interface()
		}
	}
	return data
}

// RemoveFields 清除不需要更新的字段
func RemoveFields(m Map, fields ...string) Map {
	for _, field := range fields {
		delete(m, field)
	}
	return m
}

func Json(v interface{}) string {
	buf, _ := json.Marshal(v)
	return string(buf)
}

// ValidatePractice 对前端传来的Practice结构体进行参数校验
func ValidatePractice(p *cmn.TPractice, ps []int64) error {
	var err error
	if !p.Name.Valid || p.Name.String == "" {
		err = fmt.Errorf("invalid practice Name")
		return err
	}
	if !p.CorrectMode.Valid || p.CorrectMode.String == "" || (p.CorrectMode.String != MarkMode.Normal && p.CorrectMode.String != MarkMode.AI) {
		err = fmt.Errorf("invalid practice CorrectMode")
		return err
	}
	if !p.PaperID.Valid || p.PaperID.Int64 <= 0 {
		err = fmt.Errorf("invalid practice PaperID")
		return err
	}
	if !p.Type.Valid || p.Type.String == "" || (p.Type.String != PracticeType.PracticeNew && p.Type.String != PracticeType.Classical && p.Type.String != PracticeType.Intelligent) {
		err = fmt.Errorf("invalid practice Type")
		return err
	}
	if !p.AllowedAttempts.Valid || p.AllowedAttempts.Int64 < 0 {
		err = fmt.Errorf("invalid practice AllowedAttempts")
		return err
	}
	if ps != nil && len(ps) > 0 {
		for _, id := range ps {
			if id <= 0 {
				err = fmt.Errorf("invalid practice studentID")
				return err
			}
		}
	}
	return nil

}
