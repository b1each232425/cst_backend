package registration

import (
	"fmt"
	"github.com/jmoiron/sqlx/types"
	"reflect"
	"strings"
	"w2w.io/cmn"
	"w2w.io/null"
)

type Map map[string]interface{}

func ValidateRegisterInfo(R *cmn.TRegisterPlan, Rs []int64) error {
	var err error
	if !R.Name.Valid || R.Name.String == "" {
		err = fmt.Errorf("invalid register Name")
		return err
	}
	if !R.StartTime.Valid || R.StartTime.Int64 <= 0 {
		err = fmt.Errorf("invalid register StartTime")
		return err
	}
	if !R.EndTime.Valid || R.EndTime.Int64 <= 0 {
		err = fmt.Errorf("invalid register EndTime")
		return err
	}
	if !R.ReviewEndTime.Valid || R.ReviewEndTime.Int64 <= 0 {
		err = fmt.Errorf("invalid register ReviewEndTime")
		return err
	}
	if !R.MaxNumber.Valid || R.MaxNumber.Int64 < 0 {
		err = fmt.Errorf("invalid register MaxNumber")
		return err
	}
	if !R.Course.Valid || R.Course.String == "" || (R.Course.String != "00" && R.Course.String != "02" && R.Course.String != "04") {
		err = fmt.Errorf("invalid register Course")
		return err
	}
	if !R.ExamPlanLocation.Valid || R.ExamPlanLocation.String == "" {
		err = fmt.Errorf("invalid register ExamPlanLocation")
		return err
	}
	if Rs != nil && len(Rs) > 0 {
		for _, id := range Rs {
			if id <= 0 {
				err = fmt.Errorf("invalid register practiceIDs")
				return err
			}
		}
	}
	return nil
}
func s2Map(in interface{}) Map {
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

// 检测前端传过来的审查人员ids是不是切片
func CheckReviewerIDs(i interface{}) error {
	t := reflect.TypeOf(i)
	switch t.Kind() {
	case reflect.Slice:
		return nil
	default:
		return fmt.Errorf("invalid reviewerIDs")
	}
}
