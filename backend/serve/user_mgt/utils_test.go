package user_mgt

import (
	"fmt"
	"strings"
	"testing"

	"w2w.io/null"
)

func TestNullableString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected null.String
	}{
		{
			name:     "非空字符串",
			input:    "test_value",
			expected: null.NewString("test_value", true),
		},
		{
			name:     "空字符串",
			input:    "",
			expected: null.NewString("", false),
		},
		{
			name:     "只包含空格的字符串",
			input:    "  ",
			expected: null.NewString("  ", false),
		},
		{
			name:     "包含特殊字符的字符串",
			input:    "test@example.com",
			expected: null.NewString("test@example.com", true),
		},
		{
			name:     "中文字符串",
			input:    "测试用户",
			expected: null.NewString("测试用户", true),
		},
		{
			name:     "数字字符串",
			input:    "12345",
			expected: null.NewString("12345", true),
		},
		{
			name:     "长字符串",
			input:    "这是一个很长的字符串用于测试NullableString函数的处理能力",
			expected: null.NewString("这是一个很长的字符串用于测试NullableString函数的处理能力", true),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			result := NullableString(tt.input)

			if result.Valid != tt.expected.Valid {
				t.Errorf("NullableString(%q).Valid = %v, want %v", tt.input, result.Valid, tt.expected.Valid)
			}

			if result.String != tt.expected.String {
				t.Errorf("NullableString(%q).String = %q, want %q", tt.input, result.String, tt.expected.String)
			}
		})
	}
}

func TestNullableIntFromStr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected null.Int
	}{
		{
			name:     "有效的正整数",
			input:    "123",
			expected: null.NewInt(123, true),
		},
		{
			name:     "有效的负整数",
			input:    "-456",
			expected: null.NewInt(-456, true),
		},
		{
			name:     "零值",
			input:    "0",
			expected: null.NewInt(0, true),
		},
		{
			name:     "空字符串",
			input:    "",
			expected: null.NewInt(0, false),
		},
		{
			name:     "无效的字符串",
			input:    "abc",
			expected: null.NewInt(0, false),
		},
		{
			name:     "包含字母的数字字符串",
			input:    "123abc",
			expected: null.NewInt(0, false),
		},
		{
			name:     "浮点数字符串",
			input:    "123.45",
			expected: null.NewInt(0, false),
		},
		{
			name:     "包含空格的数字",
			input:    " 123 ",
			expected: null.NewInt(0, false),
		},
		{
			name:     "大整数",
			input:    "9223372036854775807", // int64最大值
			expected: null.NewInt(9223372036854775807, true),
		},
		{
			name:     "超出int64范围的数字",
			input:    "9223372036854775808", // 超出int64最大值
			expected: null.NewInt(0, false),
		},
		{
			name:     "最小负整数",
			input:    "-9223372036854775808", // int64最小值
			expected: null.NewInt(-9223372036854775808, true),
		},
		{
			name:     "带加号的正数",
			input:    "+123",
			expected: null.NewInt(123, true),
		},
		{
			name:     "十六进制数字",
			input:    "0x123",
			expected: null.NewInt(0, false),
		},
		{
			name:     "八进制数字",
			input:    "0123",
			expected: null.NewInt(123, true), // strconv.ParseInt会将其解析为十进制
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			result := NullableIntFromStr(tt.input)

			if result.Valid != tt.expected.Valid {
				t.Errorf("NullableIntFromStr(%q).Valid = %v, want %v", tt.input, result.Valid, tt.expected.Valid)
			}

			if result.Int64 != tt.expected.Int64 {
				t.Errorf("NullableIntFromStr(%q).Int64 = %d, want %d", tt.input, result.Int64, tt.expected.Int64)
			}
		})
	}
}

// TestNullableString_边界条件测试
func TestNullableString_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectValid bool
		desc        string
	}{
		{
			name:        "单个字符",
			input:       "a",
			expectValid: true,
			desc:        "单个字符应该被认为是有效的",
		},
		{
			name:        "只包含换行符",
			input:       "\n",
			expectValid: false,
			desc:        "换行符不应该被认为是有效的字符",
		},
		{
			name:        "只包含制表符",
			input:       "\t",
			expectValid: false,
			desc:        "制表符不应该被认为是有效的字符",
		},
		{
			name:        "Unicode字符",
			input:       "🚀",
			expectValid: true,
			desc:        "Unicode字符应该被正确处理",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			result := NullableString(tt.input)

			if result.Valid != tt.expectValid {
				t.Errorf("%s: NullableString(%q).Valid = %v, want %v", tt.desc, tt.input, result.Valid, tt.expectValid)
			}

			if result.Valid && result.String != tt.input {
				t.Errorf("%s: NullableString(%q).String = %q, want %q", tt.desc, tt.input, result.String, tt.input)
			}
		})
	}
}

// TestNullableIntFromStr_边界条件测试
func TestNullableIntFromStr_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectValid bool
		expectValue int64
		desc        string
	}{
		{
			name:        "前导零",
			input:       "000123",
			expectValid: true,
			expectValue: 123,
			desc:        "前导零应该被正确处理",
		},
		{
			name:        "只有零",
			input:       "000",
			expectValid: true,
			expectValue: 0,
			desc:        "多个零应该被解析为0",
		},
		{
			name:        "负零",
			input:       "-0",
			expectValid: true,
			expectValue: 0,
			desc:        "负零应该被解析为0",
		},
		{
			name:        "单独的负号",
			input:       "-",
			expectValid: false,
			expectValue: 0,
			desc:        "单独的负号应该被认为是无效的",
		},
		{
			name:        "单独的加号",
			input:       "+",
			expectValid: false,
			expectValue: 0,
			desc:        "单独的加号应该被认为是无效的",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			result := NullableIntFromStr(tt.input)

			if result.Valid != tt.expectValid {
				t.Errorf("%s: NullableIntFromStr(%q).Valid = %v, want %v", tt.desc, tt.input, result.Valid, tt.expectValid)
			}

			if result.Valid && result.Int64 != tt.expectValue {
				t.Errorf("%s: NullableIntFromStr(%q).Int64 = %d, want %d", tt.desc, tt.input, result.Int64, tt.expectValue)
			}
		})
	}
}

// Benchmark测试
func BenchmarkNullableString(b *testing.B) {
	testCases := []string{
		"",
		"short",
		"这是一个中等长度的测试字符串用于性能测试",
		"This is a very long string that is used for performance testing of the NullableString function to see how it performs with longer inputs",
	}

	for _, tc := range testCases {
		b.Run("len_"+string(rune(len(tc))), func(b *testing.B) {

			for i := 0; i < b.N; i++ {
				_ = NullableString(tc)
			}
		})
	}
}

func BenchmarkNullableIntFromStr(b *testing.B) {
	testCases := []string{
		"",
		"0",
		"123",
		"-456",
		"9223372036854775807",
		"invalid",
	}

	for _, tc := range testCases {
		b.Run("input_"+tc, func(b *testing.B) {

			for i := 0; i < b.N; i++ {
				_ = NullableIntFromStr(tc)
			}
		})
	}
}

// TestContains 测试Contains泛型函数
func TestContains(t *testing.T) {
	t.Run("字符串切片测试", func(t *testing.T) {
		tests := []struct {
			name     string
			item     string
			list     []string
			expected bool
		}{
			{
				name:     "存在的元素",
				item:     "apple",
				list:     []string{"apple", "banana", "cherry"},
				expected: true,
			},
			{
				name:     "不存在的元素",
				item:     "orange",
				list:     []string{"apple", "banana", "cherry"},
				expected: false,
			},
			{
				name:     "空切片",
				item:     "apple",
				list:     []string{},
				expected: false,
			},
			{
				name:     "空字符串元素",
				item:     "",
				list:     []string{"", "test"},
				expected: true,
			},
			{
				name:     "重复元素",
				item:     "test",
				list:     []string{"test", "test", "other"},
				expected: true,
			},
			{
				name:     "单个元素切片-存在",
				item:     "only",
				list:     []string{"only"},
				expected: true,
			},
			{
				name:     "单个元素切片-不存在",
				item:     "missing",
				list:     []string{"only"},
				expected: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := Contains(tt.item, tt.list)
				if result != tt.expected {
					t.Errorf("Contains(%q, %v) = %v, want %v", tt.item, tt.list, result, tt.expected)
				}
			})
		}
	})

	t.Run("整数切片测试", func(t *testing.T) {
		tests := []struct {
			name     string
			item     int
			list     []int
			expected bool
		}{
			{
				name:     "存在的正整数",
				item:     5,
				list:     []int{1, 3, 5, 7, 9},
				expected: true,
			},
			{
				name:     "不存在的整数",
				item:     4,
				list:     []int{1, 3, 5, 7, 9},
				expected: false,
			},
			{
				name:     "负数",
				item:     -1,
				list:     []int{-3, -1, 0, 1, 3},
				expected: true,
			},
			{
				name:     "零值",
				item:     0,
				list:     []int{-1, 0, 1},
				expected: true,
			},
			{
				name:     "大整数",
				item:     1000000,
				list:     []int{999999, 1000000, 1000001},
				expected: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := Contains(tt.item, tt.list)
				if result != tt.expected {
					t.Errorf("Contains(%d, %v) = %v, want %v", tt.item, tt.list, result, tt.expected)
				}
			})
		}
	})

	t.Run("浮点数切片测试", func(t *testing.T) {
		tests := []struct {
			name     string
			item     float64
			list     []float64
			expected bool
		}{
			{
				name:     "存在的浮点数",
				item:     3.14,
				list:     []float64{1.0, 2.5, 3.14, 4.2},
				expected: true,
			},
			{
				name:     "不存在的浮点数",
				item:     2.71,
				list:     []float64{1.0, 2.5, 3.14, 4.2},
				expected: false,
			},
			{
				name:     "零值浮点数",
				item:     0.0,
				list:     []float64{-1.0, 0.0, 1.0},
				expected: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := Contains(tt.item, tt.list)
				if result != tt.expected {
					t.Errorf("Contains(%f, %v) = %v, want %v", tt.item, tt.list, result, tt.expected)
				}
			})
		}
	})

	t.Run("布尔值切片测试", func(t *testing.T) {
		tests := []struct {
			name     string
			item     bool
			list     []bool
			expected bool
		}{
			{
				name:     "存在true",
				item:     true,
				list:     []bool{false, true},
				expected: true,
			},
			{
				name:     "存在false",
				item:     false,
				list:     []bool{false, true},
				expected: true,
			},
			{
				name:     "不存在true",
				item:     true,
				list:     []bool{false, false},
				expected: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := Contains(tt.item, tt.list)
				if result != tt.expected {
					t.Errorf("Contains(%t, %v) = %t, want %t", tt.item, tt.list, result, tt.expected)
				}
			})
		}
	})
}

// TestContains_EdgeCases 测试Contains函数的边界情况
func TestContains_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		description string
		test        func(t *testing.T)
	}{
		{
			name:        "nil切片",
			description: "测试nil切片的处理",
			test: func(t *testing.T) {
				var nilSlice []string
				result := Contains("test", nilSlice)
				if result != false {
					t.Errorf("Contains with nil slice should return false, got %v", result)
				}
			},
		},
		{
			name:        "大切片性能",
			description: "测试大切片的查找性能",
			test: func(t *testing.T) {
				// 创建一个大切片
				largeSlice := make([]int, 10000)
				for i := 0; i < 10000; i++ {
					largeSlice[i] = i
				}

				// 测试查找第一个元素
				result := Contains(0, largeSlice)
				if !result {
					t.Error("Should find first element")
				}

				// 测试查找最后一个元素
				result = Contains(9999, largeSlice)
				if !result {
					t.Error("Should find last element")
				}

				// 测试查找不存在的元素
				result = Contains(10000, largeSlice)
				if result {
					t.Error("Should not find non-existent element")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)
			tt.test(t)
		})
	}
}

// BenchmarkContains 性能基准测试
func BenchmarkContains(b *testing.B) {
	// 测试不同大小的切片
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			// 创建测试切片
			slice := make([]int, size)
			for i := 0; i < size; i++ {
				slice[i] = i
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// 查找中间的元素
				_ = Contains(size/2, slice)
			}
		})
	}
}

// TestNormalizeAndValidateCNID 测试中国身份证号码验证和标准化函数
func TestNormalizeAndValidateCNID(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
		errorMsg    string
	}{
		// 18位身份证号码测试
		{
			name:        "有效的18位身份证号码",
			input:       "11010519491231002X",
			expected:    "11010519491231002X",
			expectError: false,
		},
		{
			name:        "有效的18位身份证号码-小写x",
			input:       "11010519491231002x",
			expected:    "11010519491231002X",
			expectError: false,
		},
		{
			name:        "有效的18位身份证号码-数字校验位",
			input:       "310107196503251267",
			expected:    "310107196503251267",
			expectError: false,
		},
		{
			name:        "18位身份证号码-包含空格",
			input:       "110105 1949 1231 002X",
			expected:    "11010519491231002X",
			expectError: false,
		},
		{
			name:        "18位身份证号码-前后有空格",
			input:       "  11010519491231002X  ",
			expected:    "11010519491231002X",
			expectError: false,
		},
		{
			name:        "18位身份证号码-校验位错误",
			input:       "110105194912310021",
			expected:    "",
			expectError: true,
			errorMsg:    "checksum mismatch",
		},
		{
			name:        "18位身份证号码-前17位包含非数字",
			input:       "1101051949123100aX",
			expected:    "",
			expectError: true,
			errorMsg:    "invalid char in first 17 digits",
		},
		{
			name:        "18位身份证号码-无效日期",
			input:       "11010519491331002X",
			expected:    "",
			expectError: true,
			errorMsg:    "invalid birth date",
		},
		{
			name:        "18位身份证号码-未来日期",
			input:       "11010520501231002X",
			expected:    "",
			expectError: true,
			errorMsg:    "invalid birth date",
		},
		{
			name:        "18位身份证号码-过早日期",
			input:       "11010518991231002X",
			expected:    "",
			expectError: true,
			errorMsg:    "invalid birth date",
		},
		{
			name:        "18位身份证号码-无效校验位字符",
			input:       "11010519491231002Z",
			expected:    "",
			expectError: true,
			errorMsg:    "invalid last char",
		},
		// 15位身份证号码测试
		{
			name:        "有效的15位身份证号码",
			input:       "110105491231002",
			expected:    "11010519491231002X",
			expectError: false,
		},
		{
			name:        "15位身份证号码-包含空格",
			input:       "110105 491231 002",
			expected:    "11010519491231002X",
			expectError: false,
		},
		{
			name:        "15位身份证号码-前后有空格",
			input:       "  110105491231002  ",
			expected:    "11010519491231002X",
			expectError: false,
		},
		{
			name:        "15位身份证号码-包含非数字",
			input:       "11010549123100a",
			expected:    "",
			expectError: true,
			errorMsg:    "invalid char in 15-digit id",
		},
		{
			name:        "15位身份证号码-无效日期",
			input:       "110105491331002",
			expected:    "",
			expectError: true,
			errorMsg:    "invalid birth date in 15-digit id",
		},
		// 边界情况测试
		{
			name:        "空字符串",
			input:       "",
			expected:    "",
			expectError: true,
			errorMsg:    "empty id",
		},
		{
			name:        "只有空格",
			input:       "   ",
			expected:    "",
			expectError: true,
			errorMsg:    "empty id",
		},
		{
			name:        "长度不正确-16位",
			input:       "1101051949123100",
			expected:    "",
			expectError: true,
			errorMsg:    "id length must be 15 or 18",
		},
		{
			name:        "长度不正确-17位",
			input:       "11010519491231002",
			expected:    "",
			expectError: true,
			errorMsg:    "id length must be 15 or 18",
		},
		{
			name:        "长度不正确-19位",
			input:       "11010519491231002XX",
			expected:    "",
			expectError: true,
			errorMsg:    "id length must be 15 or 18",
		},
		{
			name:        "长度不正确-14位",
			input:       "11010549123100",
			expected:    "",
			expectError: true,
			errorMsg:    "id length must be 15 or 18",
		},
		// 特殊字符测试
		{
			name:        "包含制表符和换行符",
			input:       "110105\t1949\n1231002X",
			expected:    "11010519491231002X",
			expectError: false,
		},
		// 真实有效身份证号码测试（使用公开的测试用身份证号码）
		{
			name:        "真实格式18位身份证-1",
			input:       "310107196503251267",
			expected:    "310107196503251267",
			expectError: false,
		},
		{
			name:        "科学计数法表示的18位身份证",
			input:       "4.41322E+17",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeAndValidateCNID(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("NormalizeAndValidateCNID(%q) expected error but got none", tt.input)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("NormalizeAndValidateCNID(%q) error = %v, want error containing %q", tt.input, err, tt.errorMsg)
				}
				if result != tt.expected {
					t.Errorf("NormalizeAndValidateCNID(%q) result = %q, want %q", tt.input, result, tt.expected)
				}
			} else {
				if err != nil {
					t.Errorf("NormalizeAndValidateCNID(%q) unexpected error: %v", tt.input, err)
					return
				}
				if result != tt.expected {
					t.Errorf("NormalizeAndValidateCNID(%q) = %q, want %q", tt.input, result, tt.expected)
				}
			}
		})
	}
}

// TestNormalizeAndValidateCNID_EdgeCases 测试边界情况
func TestNormalizeAndValidateCNID_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
		test        func(t *testing.T, input string)
	}{
		{
			name:        "15位转18位-边界年份",
			input:       "110105000101001",
			description: "测试15位身份证转换为18位时的边界年份处理",
			test: func(t *testing.T, input string) {
				result, err := NormalizeAndValidateCNID(input)
				if err != nil {
					t.Errorf("15位转18位应该成功，但得到错误: %v", err)
				}
				if len(result) != 18 {
					t.Errorf("转换后应该是18位，但得到 %d 位: %q", len(result), result)
				}
				if !strings.HasPrefix(result, "11010519000101001") {
					t.Errorf("转换结果不正确: %q", result)
				}
			},
		},
		{
			name:        "复杂空白字符处理",
			input:       "\t\n 110105\r\n1949\t1231\u00A0002X \n\t",
			description: "测试各种空白字符的处理",
			test: func(t *testing.T, input string) {
				result, err := NormalizeAndValidateCNID(input)
				if err != nil {
					t.Errorf("复杂空白字符应该被正确处理，但得到错误: %v", err)
				}
				expected := "11010519491231002X"
				if result != expected {
					t.Errorf("空白字符处理结果不正确: got %q, want %q", result, expected)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)
			tt.test(t, tt.input)
		})
	}
}

// BenchmarkNormalizeAndValidateCNID 性能基准测试
func BenchmarkNormalizeAndValidateCNID(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{"18位有效身份证", "11010519491231002X"},
		{"15位有效身份证", "110105491231002"},
		{"包含空格的18位身份证", "110105 1949 1231 002X"},
		{"无效身份证", "invalid_id_number"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = NormalizeAndValidateCNID(tc.input)
			}
		})
	}
}
