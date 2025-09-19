package registration

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx/types"
	"reflect"
	"strings"
	"w2w.io/cmn"
	"w2w.io/null"
	"w2w.io/serve/auth_mgt"
)

type Map map[string]interface{}

func GetAuthAPIAccessible(ctx context.Context, authority *auth_mgt.Authority, apiPath string) (bool, bool, bool, bool, error) {
	full, err := auth_mgt.CheckUserAPIAccessible(ctx, authority, apiPath, "full")
	if err != nil {
		z.Error(err.Error())
		return false, false, false, false, err
	}
	if full {
		return true, true, true, true, nil
	}
	read, err := auth_mgt.CheckUserAPIAccessible(ctx, authority, apiPath, "read")
	if err != nil {
		z.Error(err.Error())
		return false, false, false, false, err
	}
	create, err := auth_mgt.CheckUserAPIAccessible(ctx, authority, apiPath, "create")
	if err != nil {
		z.Error(err.Error())
		return false, false, false, false, err
	}
	update, err := auth_mgt.CheckUserAPIAccessible(ctx, authority, apiPath, "update")
	if err != nil {
		z.Error(err.Error())
		return false, false, false, false, err
	}
	deleteble, err := auth_mgt.CheckUserAPIAccessible(ctx, authority, apiPath, "delete")
	if err != nil {
		z.Error(err.Error())
		return false, false, false, false, err
	}
	return read, create, update, deleteble, nil
}

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

// CheckReviewerIDs 检测前端传过来的审查人员ids是不是int64切片
func CheckReviewerIDs(i interface{}) ([]int64, error) {
	t := reflect.TypeOf(i)
	v := reflect.ValueOf(i)

	switch t.Kind() {
	case reflect.Slice:
		// 转换为[]int64
		int64Slice := make([]int64, v.Len())
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			elemValue := elem.Interface()

			// 处理不同的数值类型
			switch val := elemValue.(type) {
			case int64:
				int64Slice[i] = val
			case int:
				int64Slice[i] = int64(val)
			case float64:
				int64Slice[i] = int64(val)
			case float32:
				int64Slice[i] = int64(val)
			default:
				// 尝试通过反射获取数值
				if elem.Kind() >= reflect.Int && elem.Kind() <= reflect.Int64 {
					int64Slice[i] = elem.Int()
				} else if elem.Kind() >= reflect.Uint && elem.Kind() <= reflect.Uint64 {
					int64Slice[i] = int64(elem.Uint())
				} else if elem.Kind() == reflect.Float32 || elem.Kind() == reflect.Float64 {
					int64Slice[i] = int64(elem.Float())
				} else {
					return nil, fmt.Errorf("invalid reviewerIDs element type at index %d: %T", i, elemValue)
				}
			}
		}
		return int64Slice, nil

	case reflect.Array:
		// 转换为[]int64
		int64Slice := make([]int64, v.Len())
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			elemValue := elem.Interface()

			// 处理不同的数值类型
			switch val := elemValue.(type) {
			case int64:
				int64Slice[i] = val
			case int:
				int64Slice[i] = int64(val)
			case float64:
				int64Slice[i] = int64(val)
			case float32:
				int64Slice[i] = int64(val)
			default:
				// 尝试通过反射获取数值
				if elem.Kind() >= reflect.Int && elem.Kind() <= reflect.Int64 {
					int64Slice[i] = elem.Int()
				} else if elem.Kind() >= reflect.Uint && elem.Kind() <= reflect.Uint64 {
					int64Slice[i] = int64(elem.Uint())
				} else if elem.Kind() == reflect.Float32 || elem.Kind() == reflect.Float64 {
					int64Slice[i] = int64(elem.Float())
				} else {
					return nil, fmt.Errorf("invalid reviewerIDs element type at index %d: %T", i, elemValue)
				}
			}
		}
		return int64Slice, nil

	default:
		return nil, fmt.Errorf("invalid reviewerIDs: not a slice or array, got %s", t.Kind().String())
	}
}
