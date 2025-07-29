package cmn

import (
	"encoding/json"
)

// removeNullAndEmpty 递归地去除JSON对象中值为null或空字符串的键
func removeNullAndEmpty(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		// 创建新的map来存储清理后的数据
		cleaned := make(map[string]interface{})

		for key, value := range v {
			// 跳过null值和空字符串
			if value == nil || value == "" {
				continue
			}

			// 递归处理嵌套对象
			cleanedValue := removeNullAndEmpty(value)

			// 如果递归处理后的值不为nil，则保留该键值对
			if cleanedValue != nil {
				cleaned[key] = cleanedValue
			}
		}

		return cleaned

	case []interface{}:
		// 处理数组，递归清理数组中的每个元素
		var cleaned []interface{}

		for _, item := range v {
			cleanedItem := removeNullAndEmpty(item)
			if cleanedItem != nil {
				cleaned = append(cleaned, cleanedItem)
			}
		}

		// 如果数组不为空，返回清理后的数组
		if len(cleaned) > 0 {
			return cleaned
		}
		// 如果数组为空，可以选择返回空数组或nil
		return []interface{}{}

	default:
		// 对于其他类型（字符串、数字、布尔值等），直接返回
		return v
	}
}

// CleanJSONString 清理JSON字符串，去除null和空字符串值的键
func CleanJSONString(jsonStr string) (string, error) {
	var data interface{}

	// 解析JSON字符串
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		z.Error(err.Error())
		return "", err
	}

	// 清理数据
	cleaned := removeNullAndEmpty(data)

	// 将清理后的数据转换回JSON字符串
	result, err := json.Marshal(cleaned)
	if err != nil {
		z.Error(err.Error())
		return "", err
	}

	return string(result), nil
}

// CleanJSONStringPretty 清理JSON字符串并格式化输出
func CleanJSONStringPretty(jsonStr string) (string, error) {
	var data interface{}

	// 解析JSON字符串
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		z.Error(err.Error())
		return "", err
	}

	// 清理数据
	cleaned := removeNullAndEmpty(data)

	// 将清理后的数据转换回格式化的JSON字符串
	result, err := json.MarshalIndent(cleaned, "", "  ")
	if err != nil {
		z.Error(err.Error())
		return "", err
	}

	return string(result), nil
}
