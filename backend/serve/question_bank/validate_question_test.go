package question_bank

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"w2w.io/cmn"
	"w2w.io/null"
)

func TestValidateQuestion(t *testing.T) {
	cmn.ConfigureForTest()

	tests := []struct {
		name        string
		question    *cmn.TQuestion
		wantValid   bool
		expectedErr string
	}{
		// 基础验证测试用例
		{
			name:        "question为nil",
			question:    nil,
			wantValid:   false,
			expectedErr: "question cannot be nil",
		},
		{
			name: "不支持的题目类型",
			question: &cmn.TQuestion{
				Type:       "99",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(2.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("测试题目"),
				Tags:       []byte(`["测试"]`),
			},
			wantValid:   false,
			expectedErr: "unsupported question type: 99",
		},
		{
			name: "没有校验方法的题目类型",
			question: &cmn.TQuestion{
				Type:       "test",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(2.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("测试题目"),
				Tags:       []byte(`["测试"]`),
			},
			wantValid:   false,
			expectedErr: "unsupported question type for validation: test",
		},
		{
			name: "不支持的难度值",
			question: &cmn.TQuestion{
				Type:       "00",
				Difficulty: null.IntFrom(99),
				Score:      null.FloatFrom(2.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("测试题目"),
				Tags:       []byte(`["测试"]`),
			},
			wantValid:   false,
			expectedErr: "unsupported question difficulty: {{99 true}}",
		},
		{
			name: "分数小于等于0",
			question: &cmn.TQuestion{
				Type:       "00",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("测试题目"),
				Tags:       []byte(`["测试"]`),
			},
			wantValid:   false,
			expectedErr: "question score must be greater than zero",
		},
		{
			name: "BelongTo小于等于0",
			question: &cmn.TQuestion{
				Type:       "00",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(2.0),
				BelongTo:   null.IntFrom(0),
				Content:    null.StringFrom("测试题目"),
				Tags:       []byte(`["测试"]`),
			},
			wantValid:   false,
			expectedErr: "question belongTo must be greater than zero",
		},
		{
			name: "题目内容为空",
			question: &cmn.TQuestion{
				Type:       "00",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(2.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom(""),
				Tags:       []byte(`["测试"]`),
			},
			wantValid:   false,
			expectedErr: "question content cannot be empty",
		},
		{
			name: "题目内容只有空白字符",
			question: &cmn.TQuestion{
				Type:       "00",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(2.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("   \n\t  "),
				Tags:       []byte(`["测试"]`),
			},
			wantValid:   false,
			expectedErr: "question content cannot be empty",
		},
		{
			name: "标签为空数组",
			question: &cmn.TQuestion{
				Type:       "00",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(2.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("测试题目"),
				Tags:       []byte(`[]`),
			},
			wantValid:   false,
			expectedErr: "question must have at least one tag",
		},
		{
			name: "标签包含空值",
			question: &cmn.TQuestion{
				Type:       "00",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(2.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("测试题目"),
				Tags:       []byte(`["测试", "", "标签"]`),
			},
			wantValid:   false,
			expectedErr: "question tags cannot contain empty values",
		},
		{
			name: "标签格式无效",
			question: &cmn.TQuestion{
				Type:       "00",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(2.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("测试题目"),
				Tags:       []byte(`invalid json`),
			},
			wantValid:   false,
			expectedErr: "question tags format invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := validateQuestion(tt.question)

			require.Equal(t, tt.wantValid, valid, "validation result should match expected")

			if tt.expectedErr != "" {
				require.Error(t, err, "should return error")
				require.Contains(t, err.Error(), tt.expectedErr, "error message should contain expected text")
			} else {
				require.NoError(t, err, "should not return error")
			}
		})
	}
}

func TestValidateSingleChoiceQuestion(t *testing.T) {
	cmn.ConfigureForTest()

	validOptions, _ := json.Marshal([]QuestionOption{
		{Label: "A", Value: "选项A"},
		{Label: "B", Value: "选项B"},
		{Label: "C", Value: "选项C"},
		{Label: "D", Value: "选项D"},
	})
	validAnswers, _ := json.Marshal([]string{"A"})

	tests := []struct {
		name        string
		question    *cmn.TQuestion
		wantValid   bool
		expectedErr string
	}{
		{
			name: "有效的单选题",
			question: &cmn.TQuestion{
				Type:       "00",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(2.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("这是一个单选题"),
				Tags:       []byte(`["测试"]`),
				Options:    validOptions,
				Answers:    validAnswers,
			},
			wantValid:   true,
			expectedErr: "",
		},
		{
			name: "选项少于2个",
			question: &cmn.TQuestion{
				Type:       "00",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(2.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("这是一个单选题"),
				Tags:       []byte(`["测试"]`),
				Options:    func() []byte { opts, _ := json.Marshal([]QuestionOption{{Label: "A", Value: "选项A"}}); return opts }(),
				Answers:    validAnswers,
			},
			wantValid:   false,
			expectedErr: "single choice question must have at least 2 options",
		},
		{
			name: "选项标签重复",
			question: &cmn.TQuestion{
				Type:       "00",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(2.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("这是一个单选题"),
				Tags:       []byte(`["测试"]`),
				Options: func() []byte {
					opts, _ := json.Marshal([]QuestionOption{
						{Label: "A", Value: "选项A"},
						{Label: "A", Value: "选项B"},
						{Label: "C", Value: "选项C"},
					})
					return opts
				}(),
				Answers: validAnswers,
			},
			wantValid:   false,
			expectedErr: "single choice question option labels must be unique: A",
		},
		{
			name: "选项标签为空",
			question: &cmn.TQuestion{
				Type:       "00",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(2.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("这是一个单选题"),
				Tags:       []byte(`["测试"]`),
				Options: func() []byte {
					opts, _ := json.Marshal([]QuestionOption{
						{Label: "", Value: "选项A"},
						{Label: "B", Value: "选项B"},
					})
					return opts
				}(),
				Answers: validAnswers,
			},
			wantValid:   false,
			expectedErr: "single choice question option label cannot be empty",
		},
		{
			name: "答案不在选项中",
			question: &cmn.TQuestion{
				Type:       "00",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(2.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("这是一个单选题"),
				Tags:       []byte(`["测试"]`),
				Options:    validOptions,
				Answers:    func() []byte { ans, _ := json.Marshal([]string{"X"}); return ans }(),
			},
			wantValid:   false,
			expectedErr: "single choice question answer 'X' not found in options",
		},
		{
			name: "答案数量不为1",
			question: &cmn.TQuestion{
				Type:       "00",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(2.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("这是一个单选题"),
				Tags:       []byte(`["测试"]`),
				Options:    validOptions,
				Answers:    func() []byte { ans, _ := json.Marshal([]string{"A", "B"}); return ans }(),
			},
			wantValid:   false,
			expectedErr: "single choice question must have exactly one answer, got 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := validateQuestion(tt.question)

			require.Equal(t, tt.wantValid, valid, "validation result should match expected")

			if tt.expectedErr != "" {
				require.Error(t, err, "should return error")
				require.Contains(t, err.Error(), tt.expectedErr, "error message should contain expected text")
			} else {
				require.NoError(t, err, "should not return error")
			}
		})
	}
}

func TestValidateMultipleChoiceQuestion(t *testing.T) {
	cmn.ConfigureForTest()

	validOptions, _ := json.Marshal([]QuestionOption{
		{Label: "A", Value: "选项A"},
		{Label: "B", Value: "选项B"},
		{Label: "C", Value: "选项C"},
		{Label: "D", Value: "选项D"},
	})
	validAnswers, _ := json.Marshal([]string{"A", "C"})

	tests := []struct {
		name        string
		question    *cmn.TQuestion
		wantValid   bool
		expectedErr string
	}{
		{
			name: "有效的多选题",
			question: &cmn.TQuestion{
				Type:       "02",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(3.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("这是一个多选题"),
				Tags:       []byte(`["测试"]`),
				Options:    validOptions,
				Answers:    validAnswers,
			},
			wantValid:   true,
			expectedErr: "",
		},
		{
			name: "选项少于3个",
			question: &cmn.TQuestion{
				Type:       "02",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(3.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("这是一个多选题"),
				Tags:       []byte(`["测试"]`),
				Options: func() []byte {
					opts, _ := json.Marshal([]QuestionOption{
						{Label: "A", Value: "选项A"},
						{Label: "B", Value: "选项B"},
					})
					return opts
				}(),
				Answers: validAnswers,
			},
			wantValid:   false,
			expectedErr: "multiple choice question must have at least 3 options",
		},
		{
			name: "答案少于2个",
			question: &cmn.TQuestion{
				Type:       "02",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(3.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("这是一个多选题"),
				Tags:       []byte(`["测试"]`),
				Options:    validOptions,
				Answers:    func() []byte { ans, _ := json.Marshal([]string{"A"}); return ans }(),
			},
			wantValid:   false,
			expectedErr: "multiple choice question must have at least 2 correct answers, got 1",
		},
		{
			name: "所有选项都是正确答案",
			question: &cmn.TQuestion{
				Type:       "02",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(3.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("这是一个多选题"),
				Tags:       []byte(`["测试"]`),
				Options:    validOptions,
				Answers:    func() []byte { ans, _ := json.Marshal([]string{"A", "B", "C", "D"}); return ans }(),
			},
			wantValid:   false,
			expectedErr: "multiple choice question cannot have all options as correct answers",
		},
		{
			name: "答案有重复",
			question: &cmn.TQuestion{
				Type:       "02",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(3.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("这是一个多选题"),
				Tags:       []byte(`["测试"]`),
				Options:    validOptions,
				Answers:    func() []byte { ans, _ := json.Marshal([]string{"A", "A", "C"}); return ans }(),
			},
			wantValid:   false,
			expectedErr: "multiple choice question has duplicate answer: A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := validateQuestion(tt.question)

			require.Equal(t, tt.wantValid, valid, "validation result should match expected")

			if tt.expectedErr != "" {
				require.Error(t, err, "should return error")
				require.Contains(t, err.Error(), tt.expectedErr, "error message should contain expected text")
			} else {
				require.NoError(t, err, "should not return error")
			}
		})
	}
}

func TestValidateTrueFalseQuestion(t *testing.T) {
	cmn.ConfigureForTest()

	validOptions, _ := json.Marshal([]QuestionOption{
		{Label: "A", Value: "正确"},
		{Label: "B", Value: "错误"},
	})
	validAnswers, _ := json.Marshal([]string{"A"})

	tests := []struct {
		name        string
		question    *cmn.TQuestion
		wantValid   bool
		expectedErr string
	}{
		{
			name: "有效的判断题",
			question: &cmn.TQuestion{
				Type:       "04",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(1.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("这是一个判断题"),
				Tags:       []byte(`["测试"]`),
				Options:    validOptions,
				Answers:    validAnswers,
			},
			wantValid:   true,
			expectedErr: "",
		},
		{
			name: "选项数量不为2",
			question: &cmn.TQuestion{
				Type:       "04",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(1.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("这是一个判断题"),
				Tags:       []byte(`["测试"]`),
				Options: func() []byte {
					opts, _ := json.Marshal([]QuestionOption{
						{Label: "A", Value: "正确"},
						{Label: "B", Value: "错误"},
						{Label: "C", Value: "不确定"},
					})
					return opts
				}(),
				Answers: validAnswers,
			},
			wantValid:   false,
			expectedErr: "true/false question must have exactly 2 options, got 3",
		},
		{
			name: "选项标签不是A和B",
			question: &cmn.TQuestion{
				Type:       "04",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(1.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("这是一个判断题"),
				Tags:       []byte(`["测试"]`),
				Options: func() []byte {
					opts, _ := json.Marshal([]QuestionOption{
						{Label: "X", Value: "正确"},
						{Label: "Y", Value: "错误"},
					})
					return opts
				}(),
				Answers: validAnswers,
			},
			wantValid:   false,
			expectedErr: "true/false question option labels must be A and B, got: X",
		},
		{
			name: "答案不是A或B",
			question: &cmn.TQuestion{
				Type:       "04",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(1.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("这是一个判断题"),
				Tags:       []byte(`["测试"]`),
				Options:    validOptions,
				Answers:    func() []byte { ans, _ := json.Marshal([]string{"C"}); return ans }(),
			},
			wantValid:   false,
			expectedErr: "true/false question answer must be A or B, got: C",
		},
		{
			name: "答案数量不为1",
			question: &cmn.TQuestion{
				Type:       "04",
				Difficulty: null.IntFrom(1),
				Score:      null.FloatFrom(1.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("这是一个判断题"),
				Tags:       []byte(`["测试"]`),
				Options:    validOptions,
				Answers:    func() []byte { ans, _ := json.Marshal([]string{"A", "B"}); return ans }(),
			},
			wantValid:   false,
			expectedErr: "true/false question must have exactly one answer, got 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := validateQuestion(tt.question)

			require.Equal(t, tt.wantValid, valid, "validation result should match expected")

			if tt.expectedErr != "" {
				require.Error(t, err, "should return error")
				require.Contains(t, err.Error(), tt.expectedErr, "error message should contain expected text")
			} else {
				require.NoError(t, err, "should not return error")
			}
		})
	}
}

func TestValidateFillInBlankQuestion(t *testing.T) {
	cmn.ConfigureForTest()

	validAnswers, _ := json.Marshal([]SubjectiveAnswer{
		{Index: 1, Score: 2.0, Answer: "TCP", GradingRule: "必须完全匹配"},
		{Index: 2, Score: 2.0, Answer: "HTTP", GradingRule: "必须完全匹配"},
	})

	tests := []struct {
		name        string
		question    *cmn.TQuestion
		wantValid   bool
		expectedErr string
	}{
		{
			name: "有效的填空题-使用span标签",
			question: &cmn.TQuestion{
				Type:       "06",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(4.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom(`<p><span class="blank-item">____</span>是传输层协议，<span class="blank-item">____</span>是应用层协议</p>`),
				Tags:       []byte(`["测试"]`),
				Answers:    validAnswers,
			},
			wantValid:   true,
			expectedErr: "",
		},
		{
			name: "无效的填空题-使用其他class命名",
			question: &cmn.TQuestion{
				Type:       "06",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(4.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom(`<p><span class="input-blank-item">____</span>是传输层协议，<span class="other-blank">____</span>是应用层协议</p>`),
				Tags:       []byte(`["测试"]`),
				Answers:    validAnswers,
			},
			wantValid:   false,
			expectedErr: "fill-in-blank question content must contain blank markers with span tags",
		},
		{
			name: "题目内容没有填空标记",
			question: &cmn.TQuestion{
				Type:       "06",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(4.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("这是传输层协议，这是应用层协议"),
				Tags:       []byte(`["测试"]`),
				Answers:    validAnswers,
			},
			wantValid:   false,
			expectedErr: "fill-in-blank question content must contain blank markers with span tags",
		},
		{
			name: "填空数量与答案数量不匹配",
			question: &cmn.TQuestion{
				Type:       "06",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(4.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom(`<p><span class="blank-item">____</span>是传输层协议，<span class="blank-item">____</span>是应用层协议，<span class="blank-item">____</span>是网络层协议</p>`),
				Tags:       []byte(`["测试"]`),
				Answers:    validAnswers,
			},
			wantValid:   false,
			expectedErr: "fill-in-blank question blank count (3) does not match answer count (2)",
		},
		{
			name: "答案索引重复",
			question: &cmn.TQuestion{
				Type:       "06",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(4.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom(`<p><span class="blank-item">____</span>是传输层协议，<span class="blank-item">____</span>是应用层协议</p>`),
				Tags:       []byte(`["测试"]`),
				Answers: func() []byte {
					ans, _ := json.Marshal([]SubjectiveAnswer{
						{Index: 1, Score: 2.0, Answer: "TCP", GradingRule: "必须完全匹配"},
						{Index: 1, Score: 2.0, Answer: "HTTP", GradingRule: "必须完全匹配"},
					})
					return ans
				}(),
			},
			wantValid:   false,
			expectedErr: "fill-in-blank question has duplicate answer index: 1",
		},
		{
			name: "答案索引超出范围",
			question: &cmn.TQuestion{
				Type:       "06",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(4.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom(`<p><span class="blank-item">____</span>是传输层协议，<span class="blank-item">____</span>是应用层协议</p>`),
				Tags:       []byte(`["测试"]`),
				Answers: func() []byte {
					ans, _ := json.Marshal([]SubjectiveAnswer{
						{Index: 1, Score: 2.0, Answer: "TCP", GradingRule: "必须完全匹配"},
						{Index: 5, Score: 2.0, Answer: "HTTP", GradingRule: "必须完全匹配"},
					})
					return ans
				}(),
			},
			wantValid:   false,
			expectedErr: "fill-in-blank question answer index (5) exceeds answer count (2)",
		},
		{
			name: "答案分数小于等于0",
			question: &cmn.TQuestion{
				Type:       "06",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(4.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom(`<p><span class="blank-item">____</span>是传输层协议，<span class="blank-item">____</span>是应用层协议</p>`),
				Tags:       []byte(`["测试"]`),
				Answers: func() []byte {
					ans, _ := json.Marshal([]SubjectiveAnswer{
						{Index: 1, Score: 0, Answer: "TCP", GradingRule: "必须完全匹配"},
						{Index: 2, Score: 2.0, Answer: "HTTP", GradingRule: "必须完全匹配"},
					})
					return ans
				}(),
			},
			wantValid:   false,
			expectedErr: "fill-in-blank question answer score must be greater than 0, got: 0.000000",
		},
		{
			name: "答案内容为空",
			question: &cmn.TQuestion{
				Type:       "06",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(4.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom(`<p><span class="blank-item">____</span>是传输层协议，<span class="blank-item">____</span>是应用层协议</p>`),
				Tags:       []byte(`["测试"]`),
				Answers: func() []byte {
					ans, _ := json.Marshal([]SubjectiveAnswer{
						{Index: 1, Score: 2.0, Answer: "", GradingRule: "必须完全匹配"},
						{Index: 2, Score: 2.0, Answer: "HTTP", GradingRule: "必须完全匹配"},
					})
					return ans
				}(),
			},
			wantValid:   false,
			expectedErr: "fill-in-blank question answer content cannot be empty for index 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := validateQuestion(tt.question)

			require.Equal(t, tt.wantValid, valid, "validation result should match expected")

			if tt.expectedErr != "" {
				require.Error(t, err, "should return error")
				require.Contains(t, err.Error(), tt.expectedErr, "error message should contain expected text")
			} else {
				require.NoError(t, err, "should not return error")
			}
		})
	}
}

func TestValidateEssayQuestion(t *testing.T) {
	cmn.ConfigureForTest()

	// 单个小问的有效答案
	singleValidAnswers, _ := json.Marshal([]SubjectiveAnswer{
		{Index: 1, Score: 5.0, Answer: "应从传播途径、预防措施等方面阐述", GradingRule: "答出3个要点即可得满分"},
	})

	// 多个小问的有效答案
	multipleValidAnswers, _ := json.Marshal([]SubjectiveAnswer{
		{Index: 1, Score: 3.0, Answer: "阐述计算机病毒的定义和特点", GradingRule: "答出定义得2分，特点得1分"},
		{Index: 2, Score: 4.0, Answer: "说明病毒的传播途径", GradingRule: "每说出一种途径得1分，最多4分"},
		{Index: 3, Score: 3.0, Answer: "提出预防措施", GradingRule: "答出3个要点即可得满分"},
	})

	tests := []struct {
		name        string
		question    *cmn.TQuestion
		wantValid   bool
		expectedErr string
	}{
		{
			name: "有效的简答题-单个小问",
			question: &cmn.TQuestion{
				Type:       "08",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(5.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("简述计算机病毒的传播途径"),
				Tags:       []byte(`["测试"]`),
				Answers:    singleValidAnswers,
				Analysis:   null.StringFrom("这是一道关于计算机安全的题目"),
			},
			wantValid:   true,
			expectedErr: "",
		},
		{
			name: "有效的简答题-多个小问",
			question: &cmn.TQuestion{
				Type:       "08",
				Difficulty: null.IntFrom(3),
				Score:      null.FloatFrom(10.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("关于计算机病毒：(1)请阐述计算机病毒的定义和特点；(2)说明病毒的传播途径；(3)提出预防措施"),
				Tags:       []byte(`["测试", "综合"]`),
				Answers:    multipleValidAnswers,
				Analysis:   null.StringFrom("这是一道综合性的计算机安全题目，考查学生对病毒的全面理解"),
			},
			wantValid:   true,
			expectedErr: "",
		},
		{
			name: "简答题没有答案",
			question: &cmn.TQuestion{
				Type:       "08",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(5.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("简述计算机病毒的传播途径"),
				Tags:       []byte(`["测试"]`),
				Answers: func() []byte {
					ans, _ := json.Marshal([]SubjectiveAnswer{})
					return ans
				}(),
				Analysis: null.StringFrom("这是一道关于计算机安全的题目"),
			},
			wantValid:   false,
			expectedErr: "essay question must have at least one answer",
		},
		{
			name: "答案索引重复",
			question: &cmn.TQuestion{
				Type:       "08",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(10.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("多小问简答题"),
				Tags:       []byte(`["测试"]`),
				Answers: func() []byte {
					ans, _ := json.Marshal([]SubjectiveAnswer{
						{Index: 1, Score: 3.0, Answer: "答案1", GradingRule: "规则1"},
						{Index: 1, Score: 4.0, Answer: "答案2", GradingRule: "规则2"}, // 重复索引
						{Index: 3, Score: 3.0, Answer: "答案3", GradingRule: "规则3"},
					})
					return ans
				}(),
				Analysis: null.StringFrom("这是一道关于计算机安全的题目"),
			},
			wantValid:   false,
			expectedErr: "essay question has duplicate answer index: 1",
		},
		{
			name: "答案索引超出范围",
			question: &cmn.TQuestion{
				Type:       "08",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(10.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("多小问简答题"),
				Tags:       []byte(`["测试"]`),
				Answers: func() []byte {
					ans, _ := json.Marshal([]SubjectiveAnswer{
						{Index: 1, Score: 3.0, Answer: "答案1", GradingRule: "规则1"},
						{Index: 2, Score: 4.0, Answer: "答案2", GradingRule: "规则2"},
						{Index: 5, Score: 3.0, Answer: "答案3", GradingRule: "规则3"}, // 索引超出范围
					})
					return ans
				}(),
				Analysis: null.StringFrom("这是一道关于计算机安全的题目"),
			},
			wantValid:   false,
			expectedErr: "essay question answer index (5) exceeds answer count (3)",
		},
		{
			name: "总分数与题目分数不匹配",
			question: &cmn.TQuestion{
				Type:       "08",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(15.0), // 题目总分15分
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("多小问简答题"),
				Tags:       []byte(`["测试"]`),
				Answers: func() []byte {
					ans, _ := json.Marshal([]SubjectiveAnswer{
						{Index: 1, Score: 3.0, Answer: "答案1", GradingRule: "规则1"},
						{Index: 2, Score: 4.0, Answer: "答案2", GradingRule: "规则2"},
						{Index: 3, Score: 3.0, Answer: "答案3", GradingRule: "规则3"},
					}) // 总分10分，与题目分数15分不匹配
					return ans
				}(),
				Analysis: null.StringFrom("这是一道关于计算机安全的题目"),
			},
			wantValid:   false,
			expectedErr: "essay question total answer score (10.000000) must match question score (15.000000)",
		},
		{
			name: "缺少分析",
			question: &cmn.TQuestion{
				Type:       "08",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(5.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("简述计算机病毒的传播途径"),
				Tags:       []byte(`["测试"]`),
				Answers:    singleValidAnswers,
				Analysis:   null.StringFrom(""),
			},
			wantValid:   false,
			expectedErr: "essay question must have analysis",
		},
		{
			name: "某个小问的答案模板为空",
			question: &cmn.TQuestion{
				Type:       "08",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(10.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("多小问简答题"),
				Tags:       []byte(`["测试"]`),
				Answers: func() []byte {
					ans, _ := json.Marshal([]SubjectiveAnswer{
						{Index: 1, Score: 3.0, Answer: "答案1", GradingRule: "规则1"},
						{Index: 2, Score: 4.0, Answer: "", GradingRule: "规则2"}, // 空答案
						{Index: 3, Score: 3.0, Answer: "答案3", GradingRule: "规则3"},
					})
					return ans
				}(),
				Analysis: null.StringFrom("这是一道关于计算机安全的题目"),
			},
			wantValid:   false,
			expectedErr: "essay question must have answer template for index 2",
		},
		{
			name: "某个小问的评分规则为空",
			question: &cmn.TQuestion{
				Type:       "08",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(10.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("多小问简答题"),
				Tags:       []byte(`["测试"]`),
				Answers: func() []byte {
					ans, _ := json.Marshal([]SubjectiveAnswer{
						{Index: 1, Score: 3.0, Answer: "答案1", GradingRule: "规则1"},
						{Index: 2, Score: 4.0, Answer: "答案2", GradingRule: ""}, // 空评分规则
						{Index: 3, Score: 3.0, Answer: "答案3", GradingRule: "规则3"},
					})
					return ans
				}(),
				Analysis: null.StringFrom("这是一道关于计算机安全的题目"),
			},
			wantValid:   false,
			expectedErr: "essay question must have grading rule for index 2",
		},
		{
			name: "某个小问的分数为0",
			question: &cmn.TQuestion{
				Type:       "08",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(7.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("多小问简答题"),
				Tags:       []byte(`["测试"]`),
				Answers: func() []byte {
					ans, _ := json.Marshal([]SubjectiveAnswer{
						{Index: 1, Score: 3.0, Answer: "答案1", GradingRule: "规则1"},
						{Index: 2, Score: 0, Answer: "答案2", GradingRule: "规则2"}, // 分数为0
						{Index: 3, Score: 4.0, Answer: "答案3", GradingRule: "规则3"},
					})
					return ans
				}(),
				Analysis: null.StringFrom("这是一道关于计算机安全的题目"),
			},
			wantValid:   false,
			expectedErr: "essay question answer score must be greater than 0, got: 0.000000",
		},
		{
			name: "答案索引小于1",
			question: &cmn.TQuestion{
				Type:       "08",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(7.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("多小问简答题"),
				Tags:       []byte(`["测试"]`),
				Answers: func() []byte {
					ans, _ := json.Marshal([]SubjectiveAnswer{
						{Index: 0, Score: 3.0, Answer: "答案1", GradingRule: "规则1"}, // 索引为0
						{Index: 2, Score: 4.0, Answer: "答案2", GradingRule: "规则2"},
					})
					return ans
				}(),
				Analysis: null.StringFrom("这是一道关于计算机安全的题目"),
			},
			wantValid:   false,
			expectedErr: "essay question answer index must be greater than 0, got: 0",
		},
		{
			name: "答案JSON格式无效",
			question: &cmn.TQuestion{
				Type:       "08",
				Difficulty: null.IntFrom(2),
				Score:      null.FloatFrom(5.0),
				BelongTo:   null.IntFrom(1001),
				Content:    null.StringFrom("简述计算机病毒的传播途径"),
				Tags:       []byte(`["测试"]`),
				Answers:    []byte(`invalid json`),
				Analysis:   null.StringFrom("这是一道关于计算机安全的题目"),
			},
			wantValid:   false,
			expectedErr: "essay question answers format invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := validateQuestion(tt.question)

			require.Equal(t, tt.wantValid, valid, "validation result should match expected")

			if tt.expectedErr != "" {
				require.Error(t, err, "should return error")
				require.Contains(t, err.Error(), tt.expectedErr, "error message should contain expected text")
			} else {
				require.NoError(t, err, "should not return error")
			}
		})
	}
}
