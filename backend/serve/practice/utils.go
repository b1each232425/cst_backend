/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 请在此填写文件描述
 * @Date: 2025-07-15 16:40:38
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-07-24 12:11:55
 */
package practice_mgt

import (
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx/types"
	"reflect"
	"sort"
	"strings"
	"time"
	"w2w.io/null"
)

type JSONText = types.JSONText
type Map map[string]interface{}

func S2Map(in interface{}) (Map, error) {
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
	return data, nil
}

// BuildUpdateSQL 构建动态更新练习信息的SQL语句
func BuildUpdateSQL(table string, filters Map, id int64) (string, []interface{}) {
	var clauses []string
	var args []interface{}
	idx := 1
	for field, value := range filters {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", field, idx))
		args = append(args, value)
		idx++
	}
	args = append(args, id)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d", table, strings.Join(clauses, ", "), idx)

	return query, args
}

// BuildQuerySQL 构建动态查询练习SQL语句
func BuildQuerySQL(table string, filters Map, orderBy []string, offset, limit int) (string, []interface{}) {
	var clauses []string
	var args []interface{}
	idx := 1

	// 处理WHERE条件
	if len(filters) > 0 {
		// 对字段名排序保证生成的SQL稳定
		keys := make([]string, 0, len(filters))
		for k := range filters {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, field := range keys {
			value := filters[field]
			// 处理NULL值情况
			if value == nil {
				clauses = append(clauses, fmt.Sprintf("%s IS NULL", field))
			} else {
				clauses = append(clauses, fmt.Sprintf("%s = $%d", field, idx))
				args = append(args, value)
				idx++
			}
		}
	}

	// 构建基础查询
	query := fmt.Sprintf("SELECT * FROM %s", table)
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}

	// 添加ORDER BY子句
	if len(orderBy) > 0 {
		query += " ORDER BY " + strings.Join(orderBy, ", ")
	}

	// 添加分页参数
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", idx)
		args = append(args, limit)
		idx++
	}

	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", idx)
		args = append(args, offset)
	}

	return query, args
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

func Timestamp(t time.Time) int64 {
	return t.Local().UnixNano() / 1e6
}
