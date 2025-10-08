package mark

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"sort"
	"testing"
	"w2w.io/cmn"
	"w2w.io/null"
)

func TestCompareSlices(t *testing.T) {
	tests := []struct {
		name           string
		a              []string
		b              []string
		expectedResult bool
	}{
		{
			name:           "内容相同切片相等",
			a:              []string{"A", "B", "C"},
			b:              []string{"A", "B", "C"},
			expectedResult: true,
		},
		{
			name:           "不同顺序但数据相同的切片应该相等",
			a:              []string{"A", "B", "C"},
			b:              []string{"C", "A", "B"},
			expectedResult: true,
		},
		{
			name:           "包含不同元素的切片不等",
			a:              []string{"A", "B", "C"},
			b:              []string{"A", "B", "D"},
			expectedResult: false,
		},
		{
			name:           "不同长度的切片不等",
			a:              []string{"A", "B"},
			b:              []string{"A", "B", "C"},
			expectedResult: false,
		},
		{
			name:           "空切片相等",
			a:              []string{},
			b:              []string{},
			expectedResult: true,
		},
		{
			name:           "一个空切片和一个非空切片不等",
			a:              []string{},
			b:              []string{"A"},
			expectedResult: false,
		},
		{
			name:           "有重复元素但内容相同切片相等",
			a:              []string{"A", "A", "B"},
			b:              []string{"A", "B", "A"},
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareSlices(tt.a, tt.b)

			assert.Equal(t, tt.expectedResult, result, "期待结果 %v，但得到 %v", tt.expectedResult, result)
		})
	}
}

func TestShuffleSlice(t *testing.T) {
	tests := []struct {
		name        string
		input       []int
		expectedLen int
	}{
		{
			name:        "一般切片",
			input:       []int{1, 2, 3, 4, 5, 6, 7, 8},
			expectedLen: 8,
		},
		{
			name:        "只有一个元素的切片",
			input:       []int{1},
			expectedLen: 1,
		},
		{
			name:        "空切片",
			input:       []int{},
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 复制输入切片以避免修改原始数据
			inputCopy := make([]int, len(tt.input))
			copy(inputCopy, tt.input)

			// 执行 shuffle
			shuffleSlice(inputCopy)

			t.Logf("inputCopy: %v", inputCopy)

			// 单元测试不做随机性测试

			assert.Equal(t, tt.expectedLen, len(inputCopy), "切片长度应为 %d，实际为 %d", tt.expectedLen, len(inputCopy))
			assert.ElementsMatch(t, tt.input, inputCopy, "切片内容（不考虑顺序）应该相同，实际不同, 原 %v， 现 %v", tt.input, inputCopy)
		})
	}
}

func TestRandomSplit(t *testing.T) {
	tests := []struct {
		name          string
		input         []int
		n             int
		expectedLen   int
		expectedError string
	}{
		{
			name:        "一般切片",
			input:       []int{1, 2, 3, 4, 5},
			n:           2,
			expectedLen: 2,
		},
		{
			name:        "不均匀分割",
			input:       []int{1, 2, 3, 4, 5},
			n:           3,
			expectedLen: 3,
		},
		{
			name:        "单元素切片",
			input:       []int{1},
			n:           2,
			expectedLen: 2,
		},
		{
			name:          "n = 0",
			input:         []int{1, 2, 3},
			n:             0,
			expectedLen:   0,
			expectedError: "分片数量 n 必须大于 0",
		},
		{
			name:        "空切片",
			input:       []int{},
			n:           2,
			expectedLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 复制输入切片
			inputCopy := make([]int, len(tt.input))
			copy(inputCopy, tt.input)

			// 执行 randomSplit
			result, err := randomSplit(inputCopy, tt.n)

			if tt.expectedError != "" {
				assert.Error(t, err, "期待错误，但是没有获取到错误")
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				// 验证结果长度
				assert.Equal(t, tt.expectedLen, len(result), "结果切片数量应为 %d，实际为 %d", tt.expectedLen, len(result))

				// 验证元素总数
				totalElements := 0
				for _, subSlice := range result {
					totalElements += len(subSlice)
				}
				assert.Equal(t, len(tt.input), totalElements, "总元素数应为 %d，实际为 %d", len(tt.input), totalElements)

				// 验证元素集合未变
				if len(tt.input) > 0 {
					originalSorted := make([]int, len(tt.input))
					copy(originalSorted, tt.input)
					sort.Ints(originalSorted)
					var resultElements []int
					for _, subSlice := range result {
						resultElements = append(resultElements, subSlice...)
					}
					sort.Ints(resultElements)
					assert.True(t, reflect.DeepEqual(originalSorted, resultElements), "分割后的元素集合应与原集合相同")
				}

				// 验证分配均匀性
				if len(tt.input) > 0 && tt.n > 1 {
					maxSize := (len(tt.input) + tt.n - 1) / tt.n // 最大子切片长度
					minSize := len(tt.input) / tt.n              // 最小子切片长度
					for _, subSlice := range result {
						assert.True(t, len(subSlice) >= minSize && len(subSlice) <= maxSize, "子切片长度应在 %d 和 %d 之间，实际为 %d", minSize, maxSize, len(subSlice))
					}
				}

				z.Sugar().Infof("randomSplit: input=%v, n=%d, result=%v", tt.input, tt.n, result)
			}

		})
	}
}

func TestValidateExamSessionOrPractice(t *testing.T) {
	tests := []struct {
		name        string
		cond        QueryCondition
		expectedErr string
	}{
		{
			name: "success: 仅提供 ExamSessionID",
			cond: QueryCondition{
				ExamSessionID: 3,
			},
		},
		{
			name: "success: 仅提供 PracticeID",
			cond: QueryCondition{
				PracticeID: 1,
			},
		},
		{
			name:        "error: 两个ID都未提供",
			cond:        QueryCondition{},
			expectedErr: "无效的 cond 参数，必须包含 考试场次ID 或者 练习ID 中的一个",
		},
		{
			name: "error: 同时提供 PracticeID 和 ExamSessionID",
			cond: QueryCondition{
				PracticeID:    1,
				ExamSessionID: 2,
			},
			expectedErr: "无效的 cond 参数，不能同时包含 考试场次ID 和 练习ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExamSessionOrPractice(tt.cond)

			if tt.expectedErr == "" {
				assert.NoError(t, err, "期望无错误")
			} else {
				assert.Error(t, err, "期望错误但没有收到")
				assert.Contains(t, err.Error(), tt.expectedErr)
			}
		})
	}
}

func TestJoinWhereClause(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "success: 空数组",
			input:    []string{},
			expected: "",
		},
		{
			name:     "success: 单个元素",
			input:    []string{"mi.mark_teacher_id = $1"},
			expected: " AND mi.mark_teacher_id = $1",
		},
		{
			name:     "success: 多个元素",
			input:    []string{"mi.mark_teacher_id = $1", "es.start_time >= $2"},
			expected: " AND mi.mark_teacher_id = $1 AND es.start_time >= $2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinWhereClause(tt.input)
			if got != tt.expected {
				t.Errorf("joinWhereClause() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestValidateMarkingResult(t *testing.T) {
	tests := []struct {
		name           string
		input          cmn.TMark
		expectedErrStr string
	}{
		{
			name: "error: 缺少 QuestionID",
			input: cmn.TMark{
				QuestionID: null.IntFrom(-1),
			},
			expectedErrStr: "缺少 问题ID",
		},
		{
			name: "error: 缺少 TeacherID",
			input: cmn.TMark{
				QuestionID: null.IntFrom(100),
			},
			expectedErrStr: "缺少 老师ID",
		},
		{
			name: "error: 缺少 Creator",
			input: cmn.TMark{
				QuestionID: null.IntFrom(100),
				TeacherID:  null.IntFrom(20),
			},
			expectedErrStr: "缺少 创建者ID",
		},
		{
			name: "error: 考试模式但缺少 ExamineeID",
			input: cmn.TMark{
				QuestionID:    null.IntFrom(100),
				TeacherID:     null.IntFrom(20),
				Creator:       null.IntFrom(10),
				ExamSessionID: null.IntFrom(1),
			},
			expectedErrStr: "缺少 考生ID",
		},
		{
			name: "error: 练习模式但缺少 PracticeSubmissionID",
			input: cmn.TMark{
				QuestionID: null.IntFrom(100),
				TeacherID:  null.IntFrom(20),
				Creator:    null.IntFrom(10),
				PracticeID: null.IntFrom(1),
			},
			expectedErrStr: "缺少 练习提交ID",
		},
		{
			name: "error: 缺少 ExamSessionID 和 PracticeID",
			input: cmn.TMark{
				QuestionID: null.IntFrom(100),
				TeacherID:  null.IntFrom(20),
				Creator:    null.IntFrom(10),
			},
			expectedErrStr: "必须包含 考试场次ID 或者 练习ID 其中一个",
		},
		{
			name: "success: 考试模式正常",
			input: cmn.TMark{
				QuestionID:    null.IntFrom(100),
				TeacherID:     null.IntFrom(20),
				Creator:       null.IntFrom(10),
				ExamSessionID: null.IntFrom(10),
				ExamineeID:    null.IntFrom(200),
			},
		},
		{
			name: "success: 练习模式正常",
			input: cmn.TMark{
				QuestionID:           null.IntFrom(100),
				TeacherID:            null.IntFrom(20),
				Creator:              null.IntFrom(10),
				PracticeID:           null.IntFrom(10),
				PracticeSubmissionID: null.IntFrom(200),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMarkingResult(tt.input)

			if tt.expectedErrStr != "" {
				assert.Contains(t, err.Error(), tt.expectedErrStr)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

//// TestSplitSlice 测试 splitSlice 函数
//func TestSplitSlice(t *testing.T) {
//	// 初始化日志
//	cmn.PackageStarters[0]()
//	z.Info("TestSplitSlice started")
//
//	ctx := context.Background()
//
//	// 定义 shouldSplit 函数：如果切片长度 > 2，则继续分割
//	shouldSplit := func(slice []int) bool {
//		return len(slice) > 2
//	}
//
//	tests := []struct {
//		name          string
//		input         []int
//		shouldSplit   func([]int) bool
//		expectedLen   int
//		expectedError string
//	}{
//		{
//			name:        "normal split",
//			input:       []int{1, 2, 3, 4, 5},
//			shouldSplit: shouldSplit,
//			expectedLen: 3, // 应分割为 [1,2], [3,4], [5]
//		},
//		{
//			name:        "no split needed",
//			input:       []int{1, 2},
//			shouldSplit: shouldSplit,
//			expectedLen: 1,
//		},
//		{
//			name:        "empty slice",
//			input:       []int{},
//			shouldSplit: shouldSplit,
//			expectedLen: 0,
//		},
//		{
//			name:        "single element",
//			input:       []int{1},
//			shouldSplit: shouldSplit,
//			expectedLen: 1,
//		},
//		{
//			name:        "large slice",
//			input:       []int{1, 2, 3, 4, 5, 6, 7, 8},
//			shouldSplit: shouldSplit,
//			expectedLen: 4, // 应分割为 [1,2], [3,4], [5,6], [7,8]
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// 复制输入切片
//			inputCopy := make([]int, len(tt.input))
//			copy(inputCopy, tt.input)
//
//			// 执行 splitSlice
//			result := splitSlice(inputCopy, tt.shouldSplit)
//
//			// 验证结果长度
//			assert.Equal(t, tt.expectedLen, len(result), "结果切片数量应为 %d，实际为 %d", tt.expectedLen, len(result))
//
//			// 验证元素总数
//			totalElements := 0
//			for _, subSlice := range result {
//				totalElements += len(subSlice)
//			}
//			assert.Equal(t, len(tt.input), totalElements, "总元素数应为 %d，实际为 %d", len(tt.input), totalElements)
//
//			// 验证元素集合未变
//			if len(tt.input) > 0 {
//				originalSorted := make([]int, len(tt.input))
//				copy(originalSorted, tt.input)
//				sort.Ints(originalSorted)
//				var resultElements []int
//				for _, subSlice := range result {
//					resultElements = append(resultElements, subSlice...)
//				}
//				sort.Ints(resultElements)
//				assert.True(t, reflect.DeepEqual(originalSorted, resultElements), "分割后的元素集合应与原集合相同")
//			}
//
//			// 验证 shouldSplit 条件
//			for _, subSlice := range result {
//				assert.False(t, tt.shouldSplit(subSlice), "子切片应满足 shouldSplit 条件，实际为 %v", subSlice)
//			}
//
//			z.Sugar().Infof("splitSlice: input=%v, result=%v", tt.input, result)
//		})
//	}
//}
