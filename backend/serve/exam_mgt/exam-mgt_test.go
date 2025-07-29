package exam_mgt

//annotation:exam_mgt
//author:{"name":"Ma Yuxin","tel":"13824087366", "email":"dbs45412@163.com"}

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"w2w.io/cmn"
	"w2w.io/null"
)

// createMockContextWithRole 创建带用户角色的模拟上下文
func createMockContextWithRole(method, path string, queryParams url.Values, forceError string, userID, userRole int64) context.Context {
	// 创建mock HTTP请求
	req := httptest.NewRequest(method, path, nil)
	req.URL.RawQuery = queryParams.Encode()

	// 创建mock HTTP响应
	w := httptest.NewRecorder()

	// 创建ServiceCtx
	serviceCtx := &cmn.ServiceCtx{
		R: req,
		W: w,
		Msg: &cmn.ReplyProto{
			API:    path,
			Method: method,
		},
		BeginTime: time.Now(),
		Tag:       make(map[string]interface{}),
		SysUser: &cmn.TUser{
			ID:   null.NewInt(userID, true),
			Role: null.NewInt(userRole, true),
		},
	}

	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)

	// 设置强制错误
	if forceError != "" {
		ctx = context.WithValue(ctx, "force-error", forceError)
	}

	return ctx
}

func createMockContextWithBody(method, path string, data string, forceError string, userID int64) context.Context {
	var req *http.Request

	if data != "" {
		// 创建ReqProto结构体，Data字段使用json.RawMessage类型
		body := &cmn.ReqProto{
			Data: json.RawMessage(data),
		}

		// 将请求体转换为JSON字符串
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			e := fmt.Sprintf("Failed to marshal request data: %v", err)
			z.Fatal(e)
		}

		// 创建mock HTTP请求
		req = httptest.NewRequest(method, path, strings.NewReader(string(bodyBytes)))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	req.Header.Set("Content-Type", "application/json")

	// 创建mock HTTP响应
	w := httptest.NewRecorder()

	// 创建ServiceCtx
	serviceCtx := &cmn.ServiceCtx{
		R: req,
		W: w,
		Msg: &cmn.ReplyProto{
			API:    path,
			Method: method,
		},
		BeginTime: time.Now(),
		Tag:       make(map[string]interface{}),
		SysUser: &cmn.TUser{
			ID: null.NewInt(userID, true), // 请求用户ID
		},
	}

	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)

	// 使用QNearKey将ServiceCtx设置到context中
	return context.WithValue(ctx, "force-error", forceError)
}

// 辅助函数：检查字符串是否包含子字符串
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestGenerateExamineeNumber(t *testing.T) {
	tests := []struct {
		name         string
		serialNumber int64
		examInfo     cmn.TExamInfo
		examSessions []cmn.TExamSession
		expected     string
		description  string
	}{
		{
			name:         "线上考试模式-返回空字符串",
			serialNumber: 1,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(123),
				Mode: null.StringFrom("00"), // 线上考试
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC).UnixMilli()),
				},
			},
			expected:    "",
			description: "线上考试不生成准考证号",
		},
		{
			name:         "线下考试模式-正常生成准考证号",
			serialNumber: 1,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(123),
				Mode: null.StringFrom("02"), // 线下考试
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC).UnixMilli()),
				},
			},
			expected:    "24123000001",
			description: "24(年份) + 123(考试ID) + 000001(序号)",
		},
		{
			name:         "大序号测试-超过6位数",
			serialNumber: 1234567,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(789),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()),
				},
			},
			expected:    "257891234567",
			description: "序号超过6位时正常显示",
		},
		{
			name:         "小序号测试-需要补零",
			serialNumber: 5,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(1),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(time.Date(2023, 7, 20, 14, 30, 0, 0, time.UTC).UnixMilli()),
				},
			},
			expected:    "231000005",
			description: "序号5 -> 000005",
		},
		{
			name:         "序号为0的情况",
			serialNumber: 0,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(999),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(time.Date(2025, 12, 25, 9, 0, 0, 0, time.UTC).UnixMilli()),
				},
			},
			expected:    "25999000000",
			description: "序号0 -> 000000",
		},
		{
			name:         "大考试ID测试",
			serialNumber: 100,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(999999),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(time.Date(2026, 3, 15, 16, 45, 0, 0, time.UTC).UnixMilli()),
				},
			},
			expected:    "26999999000100",
			description: "大考试ID正常处理",
		},
		{
			name:         "空考试场次-使用当前时间",
			serialNumber: 50,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(777),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{},
			expected:     "",
			description:  "空场次时使用当前年份",
		},
		{
			name:         "无效的开始时间-使用当前时间",
			serialNumber: 25,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(888),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.IntFrom(0), // 无效时间戳
				},
			},
			expected:    "",
			description: "无效时间戳时使用当前年份",
		},
		{
			name:         "StartTime不Valid的情况",
			serialNumber: 10,
			examInfo: cmn.TExamInfo{
				ID:   null.IntFrom(555),
				Mode: null.StringFrom("02"),
			},
			examSessions: []cmn.TExamSession{
				{
					StartTime: null.Int{}, // StartTime不Valid
				},
			},
			expected:    "",
			description: "StartTime不Valid时使用当前年份",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateExamineeNumber(tt.serialNumber, tt.examInfo, tt.examSessions)

			if tt.expected == "" {
				// 对于使用当前时间的情况，我们需要动态验证
				if tt.examInfo.Mode.String == "00" {
					// 线上考试应该返回空字符串
					if result != "" {
						t.Errorf("generateExamineeNumber() = %v, 期望空字符串", result)
					}
				} else if len(tt.examSessions) == 0 || !tt.examSessions[0].StartTime.Valid || tt.examSessions[0].StartTime.Int64 <= 0 {
					// 使用当前时间的情况，验证格式是否正确
					currentYear := time.Now().Year() % 100
					expectedPrefix := fmt.Sprintf("%02d%d", currentYear, tt.examInfo.ID.Int64)
					expectedSuffix := fmt.Sprintf("%06d", tt.serialNumber)
					expectedResult := expectedPrefix + expectedSuffix

					if result != expectedResult {
						t.Errorf("generateExamineeNumber() = %v, 期望 %v (使用当前年份 %d)", result, expectedResult, currentYear)
					}
				}
			} else {
				if result != tt.expected {
					t.Errorf("generateExamineeNumber() = %v, 期望 %v (%s)", result, tt.expected, tt.description)
				}
			}
		})
	}
}

func TestValidateExamData(t *testing.T) {
	// 确保logger已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	tests := []struct {
		name      string
		examData  ExamData
		isUpdate  bool
		wantError bool
		errorMsg  string
	}{
		{
			name: "有效的新建考试数据",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("期末考试"),
					Type: null.StringFrom("02"), // 期末成绩考试
					Mode: null.StringFrom("00"), // 线上考试
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",                  // 人工批卷
						PeriodMode:           null.StringFrom("00"), // 固定时段
						Duration:             null.IntFrom(120),     // 120分钟
						QuestionShuffledMode: null.StringFrom("00"), // 既有试题乱序也有选项乱序
						MarkMode:             null.StringFrom("00"), // 不需要手动批改
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: false,
		},
		{
			name: "有效的更新考试数据",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					ID:   null.IntFrom(123),
					Name: null.StringFrom("期中考试"),
					Type: null.StringFrom("00"), // 平时考试
					Mode: null.StringFrom("02"), // 线下考试
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(2),
						MarkMethod:           "02",                  // 自动批卷
						PeriodMode:           null.StringFrom("02"), // 灵活时段
						Duration:             null.IntFrom(90),      // 90分钟
						QuestionShuffledMode: null.StringFrom("02"), // 选项乱序
						MarkMode:             null.StringFrom("02"), // 全卷多评
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(90 * time.Minute).UnixMilli()),
					},
				},
			},
			isUpdate:  true,
			wantError: false,
		},
		{
			name: "更新时考试ID无效",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					ID:   null.IntFrom(0), // 无效ID
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  true,
			wantError: true,
			errorMsg:  "更新考试时传入的考试ID无效",
		},
		{
			name: "考试名称为空",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom(""), // 空名称
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试名称不能为空",
		},
		{
			name: "无效的考试类型",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("99"), // 无效类型
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "无效的考试类型",
		},
		{
			name: "无效的考试方式",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("99"), // 无效方式
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "无效的考试方式",
		},
		{
			name: "考试场次为空",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{}, // 空场次
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次不能为空",
		},
		{
			name: "无效的试卷ID",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(0), // 无效试卷ID
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的试卷ID无效",
		},
		{
			name: "无效的批卷方式",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "99", // 无效批卷方式
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的批卷方式无效",
		},
		{
			name: "考试时长无效",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(0), // 无效时长
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Unix()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).Unix()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的时长无效",
		},
		{
			name: "开始时间晚于结束时间",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
						EndTime:              null.IntFrom(time.Now().UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的开始时间晚于或等于结束时间",
		},
		{
			name: "开始时间等于结束时间",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().UnixMilli()), // 开始时间等于结束时间
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的开始时间晚于或等于结束时间",
		},
		{
			name: "考试类型为空字符串",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom(""), // 空字符串
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "无效的考试类型",
		},
		{
			name: "考试方式为空字符串",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom(""), // 空字符串
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "无效的考试方式",
		},
		{
			name: "无效的时段模式",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("99"), // 无效时段模式
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的考试时段模式无效",
		},
		{
			name: "时段模式为空字符串",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom(""), // 空字符串
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的考试时段模式无效",
		},
		{
			name: "无效的乱序方式",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("99"), // 无效乱序方式
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的乱序方式无效",
		},
		{
			name: "乱序方式为空字符串",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom(""), // 空字符串
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的乱序方式无效",
		},
		{
			name: "无效的批改配置",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("99"), // 无效批改配置
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的批改配置无效",
		},
		{
			name: "批改配置为空字符串",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom(""), // 空字符串
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的批改配置无效",
		},
		{
			name: "批卷方式为空字符串",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "", // 空字符串
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的批卷方式无效",
		},
		{
			name: "设定的考试时长大于总时长",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(180), // 180分钟，但总时长只有60分钟
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(1 * time.Hour).UnixMilli()), // 只有1小时
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "设定的考试时长",
		},
		{
			name: "负数的考试ID",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					ID:   null.IntFrom(-1), // 负数ID
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  true,
			wantError: true,
			errorMsg:  "更新考试时传入的考试ID无效",
		},
		{
			name: "负数的试卷ID",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(-1), // 负数试卷ID
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(120),
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的试卷ID无效",
		},
		{
			name: "负数的考试时长",
			examData: ExamData{
				ExamInfo: cmn.TExamInfo{
					Name: null.StringFrom("考试"),
					Type: null.StringFrom("00"),
					Mode: null.StringFrom("00"),
				},
				ExamSessions: []cmn.TExamSession{
					{
						PaperID:              null.IntFrom(1),
						MarkMethod:           "00",
						PeriodMode:           null.StringFrom("00"),
						Duration:             null.IntFrom(-10), // 负数时长
						QuestionShuffledMode: null.StringFrom("00"),
						MarkMode:             null.StringFrom("00"),
						StartTime:            null.IntFrom(time.Now().UnixMilli()),
						EndTime:              null.IntFrom(time.Now().Add(2 * time.Hour).UnixMilli()),
					},
				},
			},
			isUpdate:  false,
			wantError: true,
			errorMsg:  "考试场次的时长无效",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExamData(tt.examData, tt.isUpdate)

			if tt.wantError {
				if err == nil {
					t.Errorf("validateExamData() 期望返回错误，但实际返回 nil")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("validateExamData() 错误信息 = %v, 期望包含 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateExamData() 期望返回 nil，但实际返回错误 = %v", err)
				}
			}
		})
	}
}

// cleanupTestData 清理测试过程中插入的数据
func cleanupTestData(t *testing.T, creators []int64) {
	if len(creators) == 0 {
		return
	}

	conn := cmn.GetPgxConn()

	ctx := context.Background()

	// 开始事务
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Logf("开始清理事务失败: %v", err)
		return
	}
	defer tx.Rollback(ctx)

	// 删除考生记录
	for _, creator := range creators {
		_, err = tx.Exec(ctx, `
			DELETE FROM t_examinee 
			WHERE exam_session_id IN (
				SELECT id FROM t_exam_session WHERE creator = $1
			)`, creator)
		if err != nil {
			t.Logf("删除考生记录失败 (creator=%d): %v", creator, err)
		}

		// 删除考试场次记录
		_, err = tx.Exec(ctx, `DELETE FROM t_exam_session WHERE creator = $1`, creator)
		if err != nil {
			t.Logf("删除考试场次记录失败 (creator=%d): %v", creator, err)
		}

		// 删除考试信息记录
		_, err = tx.Exec(ctx, `DELETE FROM t_exam_info WHERE creator = $1`, creator)
		if err != nil {
			t.Logf("删除考试信息记录失败 (creator=%d): %v", creator, err)
		}
	}

	// 提交事务
	err = tx.Commit(ctx)
	if err != nil {
		t.Logf("提交清理事务失败: %v", err)
	} else {
		t.Logf("成功清理 %d 个测试考试的数据", len(creators))
	}
}

func TestExamPostMethod(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	var creators []int64
	creators = append(creators, 99999)
	t.Cleanup(func() {
		cleanupTestData(t, creators)
	})

	tests := []struct {
		name          string
		requestBody   string
		forceError    string
		expectedError bool
		errorContains string
		description   string
		userID        int64
	}{
		{
			name: "有效的考试创建请求",
			requestBody: `{
				"examInfo": {
					"Name": "期末考试",
					"Type": "02",
					"Mode": "00",
					"Rules": "考试规则",
					"Status": "00"
				},
				"examSessions": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"NameVisibilityIn": true,
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00",
					"LateEntryTime": 0,
					"EarliestSubmissionTime": 0
				}],
				"examinee": [1001, 1002, 1003],
				"examRooms": [],
				"invigilators": []
			}`,
			forceError:    "",
			expectedError: false,
			description:   "正常的考试创建请求",
			userID:        99999,
		},
		{
			name:          "空请求体",
			requestBody:   "",
			forceError:    "",
			expectedError: true,
			errorContains: "empty body",
			description:   "请求体为空时应该返回错误",
			userID:        99999,
		},
		{
			name: "考试名称为空",
			requestBody: `{
				"examInfo": {
					"Name": "",
					"Type": "00",
					"Mode": "00"
				},
				"examSessions": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00"
				}],
				"examinee": [1001]
			}`,
			forceError:    "",
			expectedError: true,
			errorContains: "考试名称不能为空",
			description:   "考试名称为空时应该返回验证错误",
			userID:        99999,
		},
		{
			name: "强制JSON解析错误",
			requestBody: `{
				"examInfo": {
					"Name": "测试考试",
					"Type": "00",
					"Mode": "00"
				},
				"examSessions": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00"
				}],
				"examinee": [1001]
			}`,
			forceError:    "json.Unmarshal",
			expectedError: true,
			description:   "模拟JSON解析失败",
			userID:        99999,
		},
		{
			name: "强制JSON解析错误",
			requestBody: `{
				"examInfo": {
					"Name": "测试考试",
					"Type": "00",
					"Mode": "00"
				},
				"examSessions": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00"
				}],
				"examinee": [1001]
			}`,
			forceError:    "json.Unmarshal2",
			expectedError: true,
			description:   "模拟JSON解析失败2",
			userID:        99999,
		},
		{
			name: "强制事务开始失败",
			requestBody: `{
				"examInfo": {
					"Name": "期末考试",
					"Type": "02",
					"Mode": "00",
					"Rules": "考试规则",
					"Status": "00"
				},
				"examSessions": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"NameVisibilityIn": true,
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00",
					"LateEntryTime": 0,
					"EarliestSubmissionTime": 0
				}],
				"examinee": [1001, 1002, 1003],
				"examRooms": [],
				"invigilators": []
			}`,
			forceError:    "tx.Begin",
			expectedError: true,
			description:   "模拟事务开始失败",
			userID:        99999,
		},
		{
			name: "强制事务回滚失败",
			requestBody: `{
				"examInfo": {
					"Name": "期末考试",
					"Type": "02",
					"Mode": "00",
					"Rules": "考试规则",
					"Status": "00"
				},
				"examSessions": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"NameVisibilityIn": true,
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00",
					"LateEntryTime": 0,
					"EarliestSubmissionTime": 0
				}],
				"examinee": [1001, 1002, 1003],
				"examRooms": [],
				"invigilators": []
			}`,
			forceError:    "tx.Rollback",
			expectedError: false,
			description:   "模拟事务回滚失败（通过验证失败触发回滚）",
			userID:        99999,
		},
		{
			name: "强制事务提交失败",
			requestBody: `{
				"examInfo": {
					"Name": "期末考试",
					"Type": "02",
					"Mode": "00",
					"Rules": "考试规则",
					"Status": "00"
				},
				"examSessions": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"NameVisibilityIn": true,
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00",
					"LateEntryTime": 0,
					"EarliestSubmissionTime": 0
				}],
				"examinee": [1001, 1002, 1003],
				"examRooms": [],
				"invigilators": []
			}`,
			forceError:    "tx.Commit",
			expectedError: false,
			description:   "模拟事务提交失败",
			userID:        99999,
		},
		{
			name: "无效的UserID",
			requestBody: `{
				"examInfo": {
					"Name": "",
					"Type": "00",
					"Mode": "00"
				},
				"examSessions": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00"
				}],
				"examinee": [1001]
			}`,
			forceError:    "",
			expectedError: true,
			description:   "无效的UserID",
			userID:        0,
		},
		{
			name: "读取请求体错误",
			requestBody: `{
				"examInfo": {
					"Name": "",
					"Type": "00",
					"Mode": "00"
				},
				"examSessions": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00"
				}],
				"examinee": [1001]
			}`,
			forceError:    "io.ReadAll",
			expectedError: true,
			description:   "读取请求体错误",
			userID:        99999,
		},
		{
			name: "关闭IO错误",
			requestBody: `{
				"examInfo": {
					"Name": "期末考试",
					"Type": "02",
					"Mode": "00",
					"Rules": "考试规则",
					"Status": "00"
				},
				"examSessions": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"NameVisibilityIn": true,
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00",
					"LateEntryTime": 0,
					"EarliestSubmissionTime": 0
				}],
				"examinee": [1001, 1002, 1003],
				"examRooms": [],
				"invigilators": []
			}`,
			forceError:    "io.Close",
			expectedError: false,
			description:   "关闭IO错误",
			userID:        99999,
		},
		{
			name: "QueryRow错误1",
			requestBody: `{
				"examInfo": {
					"Name": "期末考试",
					"Type": "02",
					"Mode": "00",
					"Rules": "考试规则",
					"Status": "00"
				},
				"examSessions": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"NameVisibilityIn": true,
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00",
					"LateEntryTime": 0,
					"EarliestSubmissionTime": 0
				}],
				"examinee": [1001, 1002, 1003],
				"examRooms": [],
				"invigilators": []
			}`,
			forceError:    "tx.QueryRow1",
			expectedError: true,
			description:   "tx.QueryRow错误1",
			userID:        99999,
		},
		{
			name: "QueryRow错误2",
			requestBody: `{
				"examInfo": {
					"Name": "期末考试",
					"Type": "02",
					"Mode": "00",
					"Rules": "考试规则",
					"Status": "00"
				},
				"examSessions": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"NameVisibilityIn": true,
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00",
					"LateEntryTime": 0,
					"EarliestSubmissionTime": 0
				}],
				"examinee": [1001, 1002, 1003],
				"examRooms": [],
				"invigilators": []
			}`,
			forceError:    "tx.QueryRow2",
			expectedError: true,
			description:   "tx.QueryRow错误2",
			userID:        99999,
		},
		{
			name: "Exec错误",
			requestBody: `{
				"examInfo": {
					"Name": "期末考试",
					"Type": "02",
					"Mode": "00",
					"Rules": "考试规则",
					"Status": "00"
				},
				"examSessions": [{
					"SessionNum": 1,
					"PaperID": 123,
					"StartTime": ` + strconv.FormatInt(time.Now().Add(24*time.Hour).UnixMilli(), 10) + `,
					"EndTime": ` + strconv.FormatInt(time.Now().Add(26*time.Hour).UnixMilli(), 10) + `,
					"Duration": 120,
					"QuestionShuffledMode": "00",
					"NameVisibilityIn": true,
					"MarkMethod": "00",
					"MarkMode": "00",
					"PeriodMode": "00",
					"LateEntryTime": 0,
					"EarliestSubmissionTime": 0
				}],
				"examinee": [1001, 1002, 1003],
				"examRooms": [],
				"invigilators": []
			}`,
			forceError:    "tx.Exec",
			expectedError: true,
			description:   "tx.Exec错误",
			userID:        99999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟上下文
			ctx := createMockContextWithBody("POST", "/api/exam", tt.requestBody, tt.forceError, tt.userID)

			func() {
				defer func() {
					if r := recover(); r != nil {
						// 如果有panic，检查是否是预期的
						if !tt.expectedError {
							t.Errorf("exam() 意外panic: %v", r)
						}
					}
				}()

				exam(ctx)
			}()

			// 获取ServiceCtx以检查结果
			serviceCtx := ctx.Value(cmn.QNearKey).(*cmn.ServiceCtx)

			if tt.expectedError {
				// 期望有错误
				if serviceCtx.Err == nil {
					t.Errorf("exam() 期望返回错误，但实际成功")
					return
				}

				if tt.errorContains != "" && !containsString(serviceCtx.Err.Error(), tt.errorContains) {
					t.Errorf("exam() 错误信息 = %v, 期望包含 %v", serviceCtx.Err.Error(), tt.errorContains)
				}
			} else {
				// 期望成功
				if serviceCtx.Err != nil {
					t.Errorf("exam() 期望成功，但返回错误: %v", serviceCtx.Err)
				}
			}
		})
	}
}

func TestExamGetMethod(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 准备测试数据 - 插入一个测试考试和场次
	conn := cmn.GetPgxConn()
	ctx := context.Background()

	// 测试数据ID
	testExamID := int64(999005)
	testSessionID1 := int64(999501)
	testSessionID2 := int64(999502)
	testUserID := int64(1)
	testStudentID := int64(101)

	// 设置测试数据
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("创建事务失败: %v", err)
	}

	// 插入考试信息
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_info (id, name, status, creator, create_time, updated_by, update_time)
		VALUES ($1, '测试考试_GET', '01', $2, $3, $2, $3)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, status = EXCLUDED.status
	`, testExamID, testUserID, time.Now().UnixMilli())
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试考试数据失败: %v", err)
	}

	// 插入考试场次
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_session (id, exam_id, paper_id, mark_mode, mark_method, session_num, status, creator, create_time, updated_by, update_time)
		VALUES ($1, $2, 1, '00', '00', 1, '01', $3, $4, $3, $4), ($5, $2, 1, '00', '00', 2, '01', $3, $4, $3, $4)
	`, testSessionID1, testExamID, testUserID, time.Now().UnixMilli(), testSessionID2)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试场次数据失败: %v", err)
	}

	// 插入考生数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_examinee (exam_session_id, student_id, status, creator, create_time)
		VALUES ($1, $2, '01', $3, $4)
	`, testSessionID1, testStudentID, testUserID, time.Now().UnixMilli())
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试考生数据失败: %v", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		t.Fatalf("提交测试数据失败: %v", err)
	}

	// 清理函数
	t.Cleanup(func() {
		tx, err := conn.Begin(ctx)
		if err != nil {
			return
		}
		defer tx.Rollback(ctx)

		tx.Exec(ctx, "DELETE FROM t_examinee WHERE exam_session_id IN ($1, $2)", testSessionID1, testSessionID2)
		tx.Exec(ctx, "DELETE FROM t_exam_session WHERE id IN ($1, $2)", testSessionID1, testSessionID2)
		tx.Exec(ctx, "DELETE FROM t_exam_info WHERE id = $1", testExamID)

		tx.Commit(ctx)
	})

	tests := []struct {
		name          string
		queryParams   string
		forceError    string
		expectedError bool
		errorContains string
		description   string
		userID        int64
		userRole      int64
		mockValues    map[string]string
	}{
		{
			name:          "正常获取考试信息-教师角色",
			queryParams:   fmt.Sprintf("exam_id=%d", testExamID),
			forceError:    "",
			expectedError: false,
			description:   "教师角色正常获取考试信息",
			userID:        testUserID,
			userRole:      2,
		},
		{
			name:          "正常获取考试信息-教师角色 Query错误",
			queryParams:   fmt.Sprintf("exam_id=%d", testExamID),
			forceError:    "conn.Query",
			expectedError: true,
			description:   "教师角色正常获取考试信息 Query错误",
			userID:        testUserID,
			userRole:      2,
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "正常获取考试信息-教师角色 Scan错误",
			queryParams:   fmt.Sprintf("exam_id=%d", testExamID),
			forceError:    "examinee_rows.Scan",
			expectedError: true,
			description:   "教师角色正常获取考试信息 Scan错误",
			userID:        testUserID,
			userRole:      2,
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "正常获取考试信息-学生角色",
			queryParams:   fmt.Sprintf("exam_id=%d", testExamID),
			forceError:    "",
			expectedError: false,
			description:   "学生角色正常获取考试信息",
			userID:        testStudentID,
			userRole:      1, // 学生角色
		},
		{
			name:          "正常获取考试信息-学生角色",
			queryParams:   fmt.Sprintf("exam_id=%d", testExamID),
			forceError:    "",
			expectedError: false,
			description:   "学生角色正常获取考试信息",
			userID:        testStudentID,
			userRole:      1, // 学生角色
		},
		{
			name:          "无权获取考试信息-教师角色",
			queryParams:   fmt.Sprintf("exam_id=%d", testExamID),
			forceError:    "",
			expectedError: true,
			description:   "无权获取考试信息-教师角色",
			userID:        2,
			userRole:      2,
		},
		{
			name:          "无效的考试ID-非数字",
			queryParams:   "exam_id=invalid",
			forceError:    "",
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "考试ID为非数字时应返回错误",
			userID:        testUserID,
			userRole:      2,
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "无效的考试ID-零值",
			queryParams:   "exam_id=0",
			forceError:    "",
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "考试ID为0时应返回错误",
			userID:        testUserID,
			userRole:      2,
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "无效的考试ID-负值",
			queryParams:   "exam_id=-1",
			forceError:    "",
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "考试ID为负值时应返回错误",
			userID:        testUserID,
			userRole:      2,
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "缺少考试ID参数",
			queryParams:   "",
			forceError:    "",
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "缺少exam_id参数时应返回错误",
			userID:        testUserID,
			userRole:      2,
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "无效的用户ID",
			queryParams:   fmt.Sprintf("exam_id=%d", testExamID),
			forceError:    "",
			expectedError: true,
			errorContains: "无效的用户ID",
			description:   "用户ID无效时应返回错误",
			userID:        0, // 无效用户ID
			userRole:      2,
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "无效的用户角色",
			queryParams:   fmt.Sprintf("exam_id=%d", testExamID),
			forceError:    "",
			expectedError: true,
			errorContains: "无效的用户角色",
			description:   "用户角色无效时应返回错误",
			userID:        testUserID,
			userRole:      0, // 无效角色
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "不存在的考试ID",
			queryParams:   "exam_id=999999",
			forceError:    "",
			expectedError: true,
			errorContains: "考试不存在",
			description:   "查询不存在的考试ID时应返回错误",
			userID:        testUserID,
			userRole:      2,
		},
		{
			name:          "模拟GetExamInfo错误",
			queryParams:   fmt.Sprintf("exam_id=%d", testExamID),
			forceError:    "",
			expectedError: true,
			description:   "模拟GetExamInfo函数返回错误",
			userID:        testUserID,
			userRole:      2,
			mockValues:    map[string]string{"test": "GetExamInfo-error"},
		},
		{
			name:          "模拟GetExamSessions错误",
			queryParams:   fmt.Sprintf("exam_id=%d", testExamID),
			forceError:    "",
			expectedError: true,
			description:   "模拟GetExamSessions函数返回错误",
			userID:        testUserID,
			userRole:      2,
			mockValues:    map[string]string{"test": "GetExamSessions-error"},
		},
		{
			name:          "模拟JSON编码错误1",
			queryParams:   fmt.Sprintf("exam_id=%d", testExamID),
			forceError:    "json.Marshal",
			expectedError: true,
			description:   "模拟JSON编码失败",
			userID:        testUserID,
			userRole:      2,
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "模拟JSON编码错误2",
			queryParams:   fmt.Sprintf("exam_id=%d", testExamID),
			forceError:    "json.Marshal",
			expectedError: true,
			description:   "模拟JSON编码失败",
			userID:        testUserID,
			userRole:      1,
			mockValues:    map[string]string{"test": "normal-resp"},
		},
		{
			name:          "模拟权限验证失败",
			queryParams:   fmt.Sprintf("exam_id=%d", testExamID),
			forceError:    "",
			expectedError: true,
			description:   "模拟用户无权限访问考试",
			userID:        999,
			userRole:      1,
			mockValues:    map[string]string{"test": "validateUserExamPermission-error"},
		},
		{
			name:          "验证考试是否存在失败",
			queryParams:   fmt.Sprintf("exam_id=%d", testExamID),
			forceError:    "",
			expectedError: true,
			description:   "模拟验证考试是否存在失败",
			userID:        999,
			userRole:      1,
			mockValues:    map[string]string{"test": "examExists-error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 解析查询参数
			queryParams := url.Values{}
			if tt.queryParams != "" {
				parts := strings.Split(tt.queryParams, "&")
				for _, part := range parts {
					if kv := strings.Split(part, "="); len(kv) == 2 {
						queryParams.Set(kv[0], kv[1])
					}
				}
			}

			// 创建模拟上下文
			ctx := createMockContextWithRole("GET", "/api/exam", queryParams, tt.forceError, tt.userID, tt.userRole)

			// 设置mock值
			for key, value := range tt.mockValues {
				ctx = context.WithValue(ctx, key, value)
			}

			func() {
				defer func() {
					if r := recover(); r != nil {
						// 如果有panic，检查是否是预期的
						if !tt.expectedError {
							t.Errorf("exam() 意外panic: %v", r)
						}
					}
				}()

				exam(ctx)
			}()

			// 获取ServiceCtx以检查结果
			serviceCtx := ctx.Value(cmn.QNearKey).(*cmn.ServiceCtx)

			if tt.expectedError {
				// 期望有错误
				if serviceCtx.Err == nil {
					t.Errorf("exam() 期望返回错误，但实际成功")
					return
				}

				if tt.errorContains != "" && !containsString(serviceCtx.Err.Error(), tt.errorContains) {
					t.Errorf("exam() 错误信息 = %v, 期望包含 %v", serviceCtx.Err.Error(), tt.errorContains)
				}
			} else {
				// 期望成功
				if serviceCtx.Err != nil {
					t.Errorf("exam() 期望成功，但返回错误: %v", serviceCtx.Err)
					return
				}

				// 验证返回的数据
				if serviceCtx.Msg.Data == nil {
					t.Errorf("exam() 期望返回数据，但数据为空")
					return
				}

				// 尝试解析返回的JSON数据
				var examData ExamData
				if err := json.Unmarshal(serviceCtx.Msg.Data, &examData); err != nil {
					t.Errorf("exam() 返回数据格式错误: %v", err)
					return
				}

				// 验证考试信息
				if examData.ExamInfo.ID.Int64 != testExamID {
					t.Logf("exam() 返回的信息 %v", examData.ExamInfo)
					t.Errorf("exam() 返回的考试ID错误，期望 %d, 实际 %d", testExamID, examData.ExamInfo.ID.Int64)

				}

				// 验证场次信息
				if len(examData.ExamSessions) == 0 {
					t.Errorf("exam() 期望返回场次信息，但为空")
				}
			}
		})
	}
}

// TestExamPutMethod 测试 exam 函数的 PUT 方法
func TestExamPutMethod(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	tests := []struct {
		name          string
		description   string
		userID        int64
		userRole      int64
		expectedError bool
		errorContains string
	}{
		{
			name:          "PUT方法-不支持",
			description:   "PUT方法应该返回不支持的错误",
			userID:        1,
			userRole:      2,
			expectedError: true,
			errorContains: "unsupported method: put",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟上下文
			queryParams := url.Values{}
			ctx := createMockContextWithRole("PUT", "/api/exam", queryParams, "", tt.userID, tt.userRole)

			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("exam() 意外panic: %v", r)
					}
				}()

				exam(ctx)
			}()

			// 获取ServiceCtx以检查结果
			serviceCtx := ctx.Value(cmn.QNearKey).(*cmn.ServiceCtx)

			if tt.expectedError {
				// 期望有错误
				if serviceCtx.Err == nil {
					t.Errorf("exam() 期望返回错误，但实际成功")
					return
				}

				if tt.errorContains != "" && !containsString(serviceCtx.Err.Error(), tt.errorContains) {
					t.Errorf("exam() 错误信息 = %v, 期望包含 %v", serviceCtx.Err.Error(), tt.errorContains)
				}
			} else {
				// 期望成功
				if serviceCtx.Err != nil {
					t.Errorf("exam() 期望成功，但返回错误: %v", serviceCtx.Err)
				}
			}
		})
	}
}

// TestExamUnsupportedMethod 测试 exam 函数的不支持方法（default case）
func TestExamUnsupportedMethod(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	tests := []struct {
		name          string
		method        string
		description   string
		userID        int64
		userRole      int64
		expectedError bool
		errorContains string
	}{
		{
			name:          "DELETE方法-不支持",
			method:        "DELETE",
			description:   "DELETE方法应该返回不支持的错误",
			userID:        1,
			userRole:      2,
			expectedError: true,
			errorContains: "unsupported method: delete",
		},
		{
			name:          "PATCH方法-不支持",
			method:        "PATCH",
			description:   "PATCH方法应该返回不支持的错误",
			userID:        1,
			userRole:      2,
			expectedError: true,
			errorContains: "unsupported method: patch",
		},
		{
			name:          "OPTIONS方法-不支持",
			method:        "OPTIONS",
			description:   "OPTIONS方法应该返回不支持的错误",
			userID:        1,
			userRole:      2,
			expectedError: true,
			errorContains: "unsupported method: options",
		},
		{
			name:          "HEAD方法-不支持",
			method:        "HEAD",
			description:   "HEAD方法应该返回不支持的错误",
			userID:        1,
			userRole:      2,
			expectedError: true,
			errorContains: "unsupported method: head",
		},
		{
			name:          "自定义方法-不支持",
			method:        "CUSTOM",
			description:   "自定义方法应该返回不支持的错误",
			userID:        1,
			userRole:      2,
			expectedError: true,
			errorContains: "unsupported method: custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟上下文
			queryParams := url.Values{}
			ctx := createMockContextWithRole(tt.method, "/api/exam", queryParams, "", tt.userID, tt.userRole)

			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("exam() 意外panic: %v", r)
					}
				}()

				exam(ctx)
			}()

			// 获取ServiceCtx以检查结果
			serviceCtx := ctx.Value(cmn.QNearKey).(*cmn.ServiceCtx)

			if tt.expectedError {
				// 期望有错误
				if serviceCtx.Err == nil {
					t.Errorf("exam() 期望返回错误，但实际成功")
					return
				}

				if tt.errorContains != "" && !containsString(serviceCtx.Err.Error(), tt.errorContains) {
					t.Errorf("exam() 错误信息 = %v, 期望包含 %v", serviceCtx.Err.Error(), tt.errorContains)
				}
			} else {
				// 期望成功
				if serviceCtx.Err != nil {
					t.Errorf("exam() 期望成功，但返回错误: %v", serviceCtx.Err)
				}
			}
		})
	}
}

func TestValidateUserExamPermission(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 准备测试数据 - 需要先创建测试用的考试和考生数据
	conn := cmn.GetPgxConn()
	ctx := context.Background()

	// 用于清理的考试ID列表
	var testExamIDs []int64
	var testUserIDs []int64
	var testSessionIDs []int64

	// 清理函数
	defer func() {
		tx, err := conn.Begin(ctx)
		if err != nil {
			t.Logf("开始清理事务失败: %v", err)
			return
		}
		defer tx.Rollback(ctx)

		// 清理考生记录
		for _, sessionID := range testSessionIDs {
			_, err = tx.Exec(ctx, `DELETE FROM t_examinee WHERE exam_session_id = $1`, sessionID)
			if err != nil {
				t.Logf("清理考生记录失败: %v", err)
			}
		}

		// 清理考试场次
		for _, examID := range testExamIDs {
			_, err = tx.Exec(ctx, `DELETE FROM t_exam_session WHERE exam_id = $1`, examID)
			if err != nil {
				t.Logf("清理考试场次失败: %v", err)
			}
		}

		// 清理考试记录
		for _, examID := range testExamIDs {
			_, err = tx.Exec(ctx, `DELETE FROM t_exam_info WHERE id = $1`, examID)
			if err != nil {
				t.Logf("清理考试记录失败: %v", err)
			}
		}

		// 清理用户记录
		for _, userID := range testUserIDs {
			_, err = tx.Exec(ctx, `DELETE FROM t_user WHERE id = $1`, userID)
			if err != nil {
				t.Logf("清理用户记录失败: %v", err)
			}
		}

		err = tx.Commit(ctx)
		if err != nil {
			t.Logf("提交清理事务失败: %v", err)
		}
	}()

	// 创建测试用户
	testTeacherID := int64(99001)
	testStudentID := int64(99002)
	testAdminID := int64(99003)
	testUserIDs = append(testUserIDs, testTeacherID, testStudentID, testAdminID)

	currentTime := time.Now().UnixMilli()

	// 插入测试用户
	_, err := conn.Exec(ctx, `
		INSERT INTO t_user (id, role, account, category, official_name, create_time, update_time, status) 
		VALUES 
			($1, 2, 'test_teacher', '00', '测试教师', $4, $4, '00'),
			($2, 1, 'test_student', '00', '测试学生', $4, $4, '00'),
			($3, 3, 'test_admin', '00', '测试管理员', $4, $4, '00')
		ON CONFLICT (id) DO NOTHING`,
		testTeacherID, testStudentID, testAdminID, currentTime)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 创建测试考试（由testTeacherID创建）
	var testExamID int64
	err = conn.QueryRow(ctx, `
		INSERT INTO t_exam_info (
			name, type, mode, creator, create_time, updated_by, update_time, status
		) VALUES (
			'测试权限考试', '00', '00', $1, $2, $1, $2, '00'
		) RETURNING id`,
		testTeacherID, currentTime).Scan(&testExamID)
	if err != nil {
		t.Fatalf("创建测试考试失败: %v", err)
	}
	testExamIDs = append(testExamIDs, testExamID)

	// 创建测试考试场次
	var testSessionID int64
	err = conn.QueryRow(ctx, `
		INSERT INTO t_exam_session (
			exam_id, session_num, paper_id, start_time, end_time, duration,
			question_shuffled_mode, mark_method, mark_mode, period_mode,
			status, creator, create_time, updated_by, update_time
		) VALUES (
			$1, 1, 123, $2, $3, 120, '00', '00', '00', '00', '00', $4, $2, $4, $2
		) RETURNING id`,
		testExamID, currentTime, currentTime+7200000, testTeacherID).Scan(&testSessionID)
	if err != nil {
		t.Fatalf("创建测试考试场次失败: %v", err)
	}
	testSessionIDs = append(testSessionIDs, testSessionID)

	// 创建考生记录（将testStudentID添加为考生）
	_, err = conn.Exec(ctx, `
		INSERT INTO t_examinee (
			student_id, exam_session_id, creator, create_time, updated_by, update_time, 
			status, addi, serial_number, examinee_number
		) VALUES (
			$1, $2, $3, $4, $3, $4, '00', '{}', 1, '24001000001'
		)`,
		testStudentID, testSessionID, testTeacherID, currentTime)
	if err != nil {
		t.Fatalf("创建考生记录失败: %v", err)
	}

	// 创建另一个考试（不属于testTeacherID）
	var otherExamID int64
	err = conn.QueryRow(ctx, `
		INSERT INTO t_exam_info (
			name, type, mode, creator, create_time, updated_by, update_time, status
		) VALUES (
			'其他人的考试', '00', '00', $1, $2, $1, $2, '00'
		) RETURNING id`,
		88888, currentTime).Scan(&otherExamID)
	if err != nil {
		t.Fatalf("创建其他考试失败: %v", err)
	}
	testExamIDs = append(testExamIDs, otherExamID)

	tests := []struct {
		name        string
		userID      int64
		examID      int64
		role        int64
		wantResult  bool
		wantError   bool
		errorMsg    string
		description string
		forceError  string
		mockValue   string
	}{
		{
			name:        "管理员权限-应该有权限",
			userID:      testAdminID,
			examID:      testExamID,
			role:        3, // 管理员角色
			wantResult:  true,
			wantError:   false,
			description: "管理员对所有考试都有权限",
			forceError:  "",
		},
		{
			name:        "教师权限-自己创建的考试,QueryRow 错误",
			userID:      testTeacherID,
			examID:      testExamID,
			role:        2, // 教师角色
			wantResult:  false,
			wantError:   true,
			description: "教师对自己创建的考试有权限",
			forceError:  "conn.QueryRow",
		},
		{
			name:        "教师权限-自己创建的考试",
			userID:      testTeacherID,
			examID:      testExamID,
			role:        2, // 教师角色
			wantResult:  true,
			wantError:   false,
			description: "教师对自己创建的考试有权限",
			forceError:  "",
		},
		{
			name:        "教师权限-不是自己创建的考试",
			userID:      testTeacherID,
			examID:      otherExamID,
			role:        2, // 教师角色
			wantResult:  false,
			wantError:   false,
			description: "教师对不是自己创建的考试没有权限",
			forceError:  "",
		},
		{
			name:        "学生权限-参加的考试",
			userID:      testStudentID,
			examID:      testExamID,
			role:        1, // 学生角色
			wantResult:  true,
			wantError:   false,
			description: "学生对自己参加的考试有权限",
			forceError:  "",
		},
		{
			name:        "学生权限-参加的考试 QueryRow 错误",
			userID:      testStudentID,
			examID:      testExamID,
			role:        1, // 学生角色
			wantResult:  false,
			wantError:   true,
			description: "学生对自己参加的考试有权限",
			forceError:  "conn.QueryRow",
		},
		{
			name:        "学生权限-未参加的考试",
			userID:      testStudentID,
			examID:      otherExamID,
			role:        1, // 学生角色
			wantResult:  false,
			wantError:   false,
			description: "学生对未参加的考试没有权限",
			forceError:  "",
		},
		{
			name:        "无效的用户ID",
			userID:      0,
			examID:      testExamID,
			role:        1,
			wantResult:  false,
			wantError:   true,
			errorMsg:    "无效的用户ID或考试ID",
			description: "用户ID为0时应该返回错误",
			forceError:  "",
		},
		{
			name:        "负数用户ID",
			userID:      -1,
			examID:      testExamID,
			role:        1,
			wantResult:  false,
			wantError:   true,
			errorMsg:    "无效的用户ID或考试ID",
			description: "负数用户ID应该返回错误",
			forceError:  "",
		},
		{
			name:        "无效的考试ID",
			userID:      testStudentID,
			examID:      0,
			role:        1,
			wantResult:  false,
			wantError:   true,
			errorMsg:    "无效的用户ID或考试ID",
			description: "考试ID为0时应该返回错误",
			forceError:  "",
		},
		{
			name:        "负数考试ID",
			userID:      testStudentID,
			examID:      -1,
			role:        1,
			wantResult:  false,
			wantError:   true,
			errorMsg:    "无效的用户ID或考试ID",
			description: "负数考试ID应该返回错误",
			forceError:  "",
		},
		// Mock测试用例
		{
			name:        "Mock-normal-resp",
			userID:      testStudentID,
			examID:      testExamID,
			role:        1,
			wantResult:  true,
			wantError:   false,
			description: "Mock normal-resp应该返回true, nil",
			forceError:  "",
			mockValue:   "normal-resp",
		},
		{
			name:        "Mock-validateUserExamPermission-false",
			userID:      testStudentID,
			examID:      testExamID,
			role:        1,
			wantResult:  false,
			wantError:   false,
			description: "Mock validateUserExamPermission-false应该返回false, nil",
			forceError:  "",
			mockValue:   "validateUserExamPermission-false",
		},
		{
			name:        "Mock-validateUserExamPermission-error",
			userID:      testStudentID,
			examID:      testExamID,
			role:        1,
			wantResult:  false,
			wantError:   true,
			errorMsg:    "validateUserExamPermission error",
			description: "Mock validateUserExamPermission-error应该返回false, error",
			forceError:  "",
			mockValue:   "validateUserExamPermission-error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			// 设置错误注入
			if tt.forceError != "" {
				ctx = context.WithValue(ctx, "force-error", tt.forceError)
			}
			// 设置mock值
			if tt.mockValue != "" {
				ctx = context.WithValue(ctx, "test", tt.mockValue)
			}

			result, err := validateUserExamPermission(ctx, tt.userID, tt.examID, tt.role)

			// 检查错误
			if tt.wantError {
				if err == nil {
					t.Errorf("validateUserExamPermission() 期望返回错误，但实际没有错误")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("validateUserExamPermission() 错误信息 = %v, 期望包含 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateUserExamPermission() 期望没有错误，但返回错误: %v", err)
					return
				}
			}

			// 检查结果
			if result != tt.wantResult {
				t.Errorf("validateUserExamPermission() = %v, 期望 %v (%s)", result, tt.wantResult, tt.description)
			}
		})
	}
}

func TestGetExamInfo(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 准备测试数据
	conn := cmn.GetPgxConn()
	ctx := context.Background()

	// 用于清理的数据ID列表
	var testExamIDs []int64

	// 清理函数
	defer func() {
		for _, examID := range testExamIDs {
			_, err := conn.Exec(ctx, `DELETE FROM t_exam_info WHERE id = $1`, examID)
			if err != nil {
				t.Logf("清理考试记录失败 (exam_id=%d): %v", examID, err)
			}
		}
	}()

	// 创建测试考试
	currentTime := time.Now().UnixMilli()
	var testExamID int64
	err := conn.QueryRow(ctx, `
		INSERT INTO t_exam_info (
			name, rules, type, mode, creator, create_time, updated_by, update_time, 
			status, exam_delivery_status
		) VALUES (
			'测试考试信息', '考试规则说明', '00', '02', 99001, $1, 99001, $1, '00', '00'
		) RETURNING id`,
		currentTime).Scan(&testExamID)
	if err != nil {
		t.Fatalf("创建测试考试失败: %v", err)
	}
	testExamIDs = append(testExamIDs, testExamID)

	// 创建已删除状态的考试
	var deletedExamID int64
	err = conn.QueryRow(ctx, `
		INSERT INTO t_exam_info (
			name, rules, type, mode, creator, create_time, updated_by, update_time, 
			status, exam_delivery_status
		) VALUES (
			'已删除考试', '考试规则说明', '00', '02', 99001, $1, 99001, $1, '12', '00'
		) RETURNING id`,
		currentTime).Scan(&deletedExamID)
	if err != nil {
		t.Fatalf("创建已删除考试失败: %v", err)
	}
	testExamIDs = append(testExamIDs, deletedExamID)

	tests := []struct {
		name        string
		examID      int64
		role        int64
		mockValue   string
		wantError   bool
		errorMsg    string
		description string
		forceError  string
		checkResult func(*testing.T, cmn.TExamInfo, error)
	}{
		{
			name:        "管理员角色-正常获取考试信息",
			examID:      testExamID,
			role:        3, // 管理员
			wantError:   false,
			description: "管理员获取完整考试信息",
			checkResult: func(t *testing.T, examInfo cmn.TExamInfo, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if !examInfo.ID.Valid || examInfo.ID.Int64 != testExamID {
					t.Errorf("考试ID不匹配: got %v, want %d", examInfo.ID, testExamID)
				}
				if !examInfo.Name.Valid || examInfo.Name.String != "测试考试信息" {
					t.Errorf("考试名称不匹配: got %v, want '测试考试信息'", examInfo.Name)
				}
				// 管理员应该能看到完整信息包括creator等字段
				if !examInfo.Creator.Valid {
					t.Errorf("管理员应该能看到创建者信息")
				}
			},
		},
		{
			name:        "教师角色-正常获取考试信息",
			examID:      testExamID,
			role:        2, // 教师
			wantError:   false,
			description: "教师获取完整考试信息",
			checkResult: func(t *testing.T, examInfo cmn.TExamInfo, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if !examInfo.ID.Valid || examInfo.ID.Int64 != testExamID {
					t.Errorf("考试ID不匹配: got %v, want %d", examInfo.ID, testExamID)
				}
				// 教师也应该能看到完整信息
				if !examInfo.Creator.Valid {
					t.Errorf("教师应该能看到创建者信息")
				}
			},
		},
		{
			name:        "教师角色-正常获取考试信息 Scan错误",
			examID:      testExamID,
			role:        2,
			wantError:   true,
			description: "教师获取完整考试信息",
			checkResult: func(t *testing.T, examInfo cmn.TExamInfo, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if !examInfo.ID.Valid || examInfo.ID.Int64 != testExamID {
					t.Errorf("考试ID不匹配: got %v, want %d", examInfo.ID, testExamID)
				}
				// 教师也应该能看到完整信息
				if !examInfo.Creator.Valid {
					t.Errorf("教师应该能看到创建者信息")
				}
			},
			forceError: "conn.Scan",
		},
		{
			name:        "学生角色-只获取部分考试信息",
			examID:      testExamID,
			role:        1, // 学生
			wantError:   false,
			description: "学生只能获取部分考试信息",
			checkResult: func(t *testing.T, examInfo cmn.TExamInfo, err error) {

			},
		},
		{
			name:        "学生角色-只获取部分考试信息 Scan错误",
			examID:      testExamID,
			role:        1, // 学生
			wantError:   true,
			description: "学生只能获取部分考试信息",
			checkResult: func(t *testing.T, examInfo cmn.TExamInfo, err error) {

			},
			forceError: "conn.Scan",
		},
		{
			name:        "无效的考试ID-0",
			examID:      0,
			role:        3,
			wantError:   true,
			errorMsg:    "无效的考试ID",
			description: "考试ID为0时应该返回错误",
		},
		{
			name:        "无效的考试ID-负数",
			examID:      -1,
			role:        3,
			wantError:   true,
			errorMsg:    "无效的考试ID",
			description: "负数考试ID应该返回错误",
		},
		{
			name:        "不存在的考试ID",
			examID:      99999,
			role:        3,
			wantError:   true,
			description: "不存在的考试ID应该返回sql.ErrNoRows",
		},
		{
			name:        "已删除的考试",
			examID:      deletedExamID,
			role:        3,
			wantError:   true,
			description: "已删除的考试(status='12')应该查询不到",
		},
		{
			name:        "Mock测试-正常响应",
			examID:      testExamID,
			role:        3,
			mockValue:   "normal-resp",
			wantError:   false,
			description: "Mock正常响应",
			checkResult: func(t *testing.T, examInfo cmn.TExamInfo, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if !examInfo.ID.Valid || examInfo.ID.Int64 != 1 {
					t.Errorf("Mock考试ID不匹配: got %v, want 1", examInfo.ID)
				}
			},
		},
		{
			name:        "Mock测试-错误响应",
			examID:      testExamID,
			role:        3,
			mockValue:   "GetExamInfo-error",
			wantError:   true,
			errorMsg:    "GetExamInfo error",
			description: "Mock错误响应",
		},
		{
			name:        "Mock测试-bad-resp",
			examID:      testExamID,
			role:        3,
			mockValue:   "bad-resp",
			wantError:   false,
			errorMsg:    "",
			description: "Mock错误响应",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试上下文
			testCtx := ctx
			if tt.mockValue != "" {
				testCtx = context.WithValue(ctx, TEST, tt.mockValue)
			}

			if tt.forceError != "" {
				testCtx = context.WithValue(testCtx, "force-error", tt.forceError)
			}

			result, err := GetExamInfo(testCtx, tt.examID, tt.role)

			// 检查错误
			if tt.wantError {
				if err == nil {
					t.Errorf("GetExamInfo() 期望返回错误，但实际没有错误")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("GetExamInfo() 错误信息 = %v, 期望包含 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GetExamInfo() 期望没有错误，但返回错误: %v", err)
					return
				}

				// 使用自定义检查函数
				if tt.checkResult != nil {
					tt.checkResult(t, result, err)
				}
			}
		})
	}
}

func TestGetExamSessions(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 准备测试数据
	conn := cmn.GetPgxConn()
	ctx := context.Background()

	// 用于清理的数据ID列表
	var testExamIDs []int64
	var testSessionIDs []int64
	var testPaperIDs []int64

	// 清理函数
	defer func() {
		for _, sessionID := range testSessionIDs {
			_, err := conn.Exec(ctx, `DELETE FROM t_exam_session WHERE id = $1`, sessionID)
			if err != nil {
				t.Logf("清理考试场次失败: %v", err)
			}
		}
		for _, examID := range testExamIDs {
			_, err := conn.Exec(ctx, `DELETE FROM t_exam_info WHERE id = $1`, examID)
			if err != nil {
				t.Logf("清理考试记录失败: %v", err)
			}
		}
		for _, paperID := range testPaperIDs {
			_, err := conn.Exec(ctx, `DELETE FROM t_paper WHERE id = $1`, paperID)
			if err != nil {
				t.Logf("清理试卷记录失败: %v", err)
			}
		}
	}()

	// 创建测试试卷
	currentTime := time.Now().UnixMilli()
	var testPaperID int64
	err := conn.QueryRow(ctx, `
		INSERT INTO t_paper (name, category, creator, create_time, updated_by, update_time, status) 
		VALUES ('测试试卷', '00', 99001, $1, 99001, $1, '00') 
		RETURNING id`, currentTime).Scan(&testPaperID)
	if err != nil {
		t.Fatalf("创建测试试卷失败: %v", err)
	}
	testPaperIDs = append(testPaperIDs, testPaperID)

	// 创建测试考试
	var testExamID int64
	err = conn.QueryRow(ctx, `
		INSERT INTO t_exam_info (
			name, rules, type, mode, creator, create_time, updated_by, update_time, status
		) VALUES (
			'测试考试场次', '考试规则说明', '00', '02', 99001, $1, 99001, $1, '00'
		) RETURNING id`,
		currentTime).Scan(&testExamID)
	if err != nil {
		t.Fatalf("创建测试考试失败: %v", err)
	}
	testExamIDs = append(testExamIDs, testExamID)

	// 创建测试场次1
	var testSessionID1 int64
	err = conn.QueryRow(ctx, `
		INSERT INTO t_exam_session (
			exam_id, session_num, paper_id, start_time, end_time, duration,
			question_shuffled_mode, mark_method, mark_mode, period_mode,
			name_visibility_in, late_entry_time, early_submission_time,
			status, creator, create_time, updated_by, update_time
		) VALUES (
			$1, 1, $2, $3, $4, 120, '00', '00', '00', '02', 
			true, 5, 10, '00', 99001, $3, 99001, $3
		) RETURNING id`,
		testExamID, testPaperID, currentTime, currentTime+7200000).Scan(&testSessionID1)
	if err != nil {
		t.Fatalf("创建测试场次1失败: %v", err)
	}
	testSessionIDs = append(testSessionIDs, testSessionID1)

	// 创建测试场次2
	var testSessionID2 int64
	err = conn.QueryRow(ctx, `
		INSERT INTO t_exam_session (
			exam_id, session_num, paper_id, start_time, end_time, duration,
			question_shuffled_mode, mark_method, mark_mode, period_mode,
			name_visibility_in, late_entry_time, early_submission_time,
			status, creator, create_time, updated_by, update_time, reviewer_ids
		) VALUES (
			$1, 2, $2, $3, $4, 90, '02', '02', '01', '00', 
			false, 0, 0, '00', 99001, $3, 99001, $3, NULL
		) RETURNING id`,
		testExamID, testPaperID, currentTime+86400000, currentTime+86400000+5400000).Scan(&testSessionID2)
	if err != nil {
		t.Fatalf("创建测试场次2失败: %v", err)
	}
	testSessionIDs = append(testSessionIDs, testSessionID2)

	// 创建已删除的场次
	var deletedSessionID int64
	err = conn.QueryRow(ctx, `
		INSERT INTO t_exam_session (
			exam_id, session_num, paper_id, start_time, end_time, duration,
			question_shuffled_mode, mark_method, mark_mode, period_mode,
			status, creator, create_time, updated_by, update_time
		) VALUES (
			$1, 3, $2, $3, $4, 60, '00', '00', '00', '00', 
			'14', 99001, $3, 99001, $3
		) RETURNING id`,
		testExamID, testPaperID, currentTime, currentTime+3600000).Scan(&deletedSessionID)
	if err != nil {
		t.Fatalf("创建已删除场次失败: %v", err)
	}
	testSessionIDs = append(testSessionIDs, deletedSessionID)

	tests := []struct {
		name        string
		examID      int64
		role        int64
		mockValue   string
		wantError   bool
		errorMsg    string
		description string
		forceError  string
		checkResult func(*testing.T, []cmn.TExamSession, error)
	}{
		{
			name:        "教师角色-获取完整场次信息",
			examID:      testExamID,
			role:        2, // 教师
			wantError:   false,
			description: "教师获取完整场次信息，包括试卷信息",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if len(sessions) != 2 {
					t.Errorf("场次数量不匹配: got %d, want 2", len(sessions))
					return
				}
				// 教师也应该能看到完整信息
				session1 := sessions[0]
				if !session1.PaperName.Valid {
					t.Errorf("教师应该能看到试卷名称")
				}
			},
			forceError: "",
		},
		{
			name:        "教师角色-获取完整场次信息 Query错误",
			examID:      testExamID,
			role:        2, // 教师
			wantError:   true,
			description: "教师获取完整场次信息，包括试卷信息",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {

			},
			forceError: "conn.Query",
		},
		{
			name:        "教师角色-获取完整场次信息 Scan错误",
			examID:      testExamID,
			role:        2, // 教师
			wantError:   true,
			description: "教师获取完整场次信息，包括试卷信息",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {

			},
			forceError: "conn.Scan",
		},
		{
			name:        "学生角色-获取基本场次信息",
			examID:      testExamID,
			role:        1, // 学生
			wantError:   false,
			description: "学生只能获取基本场次信息",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if len(sessions) != 2 {
					t.Errorf("场次数量不匹配: got %d, want 2", len(sessions))
					return
				}
				// 学生不应该看到试卷名称等敏感信息
				session1 := sessions[0]
				if session1.PaperName.Valid {
					t.Errorf("学生不应该看到试卷名称，但获取到了: %v", session1.PaperName)
				}
				// 但应该能看到基本信息
				if !session1.StartTime.Valid {
					t.Errorf("学生应该能看到开始时间")
				}
			},
			forceError: "",
		},
		{
			name:        "学生角色-获取基本场次信息 Query错误",
			examID:      testExamID,
			role:        1, // 学生
			wantError:   true,
			description: "学生只能获取基本场次信息",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {

			},
			forceError: "conn.Query",
		},
		{
			name:        "学生角色-获取基本场次信息 Scan错误",
			examID:      testExamID,
			role:        1, // 学生
			wantError:   true,
			description: "学生只能获取基本场次信息",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {

			},
			forceError: "conn.Scan",
		},
		{
			name:        "无效的考试ID-0",
			examID:      0,
			role:        3,
			wantError:   true,
			errorMsg:    "无效的考试ID",
			description: "考试ID为0时应该返回错误",
		},
		{
			name:        "无效的考试ID-负数",
			examID:      -1,
			role:        3,
			wantError:   true,
			errorMsg:    "无效的考试ID",
			description: "负数考试ID应该返回错误",
		},
		{
			name:        "不存在的考试ID",
			examID:      99999,
			role:        3,
			wantError:   false,
			description: "不存在的考试ID应该返回空数组",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if len(sessions) != 0 {
					t.Errorf("不存在的考试应该返回空数组: got %d sessions", len(sessions))
				}
			},
		},
		{
			name:        "Mock测试-正常响应",
			examID:      testExamID,
			role:        3,
			mockValue:   "normal-resp",
			wantError:   false,
			description: "Mock正常响应",
			checkResult: func(t *testing.T, sessions []cmn.TExamSession, err error) {
				if err != nil {
					t.Errorf("意外错误: %v", err)
					return
				}
				if len(sessions) != 1 {
					t.Errorf("Mock应该返回1个场次: got %d", len(sessions))
					return
				}
				if !sessions[0].ID.Valid || sessions[0].ID.Int64 != 10001 {
					t.Errorf("Mock场次ID不匹配: got %v, want 10001", sessions[0].ID)
				}
			},
		},
		{
			name:        "Mock测试-错误响应",
			examID:      testExamID,
			role:        3,
			mockValue:   "GetExamSessions-error",
			wantError:   true,
			errorMsg:    "GetExamSessions error",
			description: "Mock错误响应",
		},
		{
			name:        "Mock测试-bad-resp",
			examID:      testExamID,
			role:        3,
			mockValue:   "bad-resp",
			wantError:   false,
			errorMsg:    "",
			description: "Mock错误响应",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试上下文
			testCtx := ctx
			if tt.mockValue != "" {
				testCtx = context.WithValue(ctx, TEST, tt.mockValue)
			}

			if tt.forceError != "" {
				testCtx = context.WithValue(testCtx, "force-error", tt.forceError)
			}

			result, err := GetExamSessions(testCtx, tt.examID, tt.role)

			// 检查错误
			if tt.wantError {
				if err == nil {
					t.Errorf("GetExamSessions() 期望返回错误，但实际没有错误")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("GetExamSessions() 错误信息 = %v, 期望包含 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GetExamSessions() 期望没有错误，但返回错误: %v", err)
					return
				}

				// 使用自定义检查函数
				if tt.checkResult != nil {
					tt.checkResult(t, result, err)
				}
			}
		})
	}
}

func TestUpdateExamStatus(t *testing.T) {
	// 初始化配置
	cmn.ConfigureForTest()

	// 创建基础上下文
	ctx := context.Background()

	// 创建测试用的事务
	conn := cmn.GetPgxConn()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("创建事务失败: %v", err)
	}
	defer tx.Rollback(ctx)

	// 准备测试数据 - 插入一个测试考试
	testExamID := int64(999001)
	testUserID := int64(1)
	testExamName := "测试考试_updateStatus"

	// 插入测试考试数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_info (id, name, status, creator, create_time, updated_by, update_time)
		VALUES ($1, $2, '00', $3, $4, $3, $4)
	`, testExamID, testExamName, testUserID, time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("插入测试考试数据失败: %v", err)
	}

	tests := []struct {
		name         string
		examID       int64
		newStatus    string
		userID       int64
		forceError   string
		wantError    bool
		errorMsg     string
		shouldVerify bool
	}{
		{
			name:         "正常更新考试状态-草稿到发布",
			examID:       testExamID,
			newStatus:    "01",
			userID:       testUserID,
			forceError:   "",
			wantError:    false,
			shouldVerify: true,
		},
		{
			name:         "正常更新考试状态-发布到进行中",
			examID:       testExamID,
			newStatus:    "02",
			userID:       testUserID,
			forceError:   "",
			wantError:    false,
			shouldVerify: true,
		},
		{
			name:      "无效的考试ID-零值",
			examID:    0,
			newStatus: "01",
			userID:    testUserID,
			wantError: true,
			errorMsg:  "无效的考试ID",
		},
		{
			name:      "无效的考试ID-负值",
			examID:    -1,
			newStatus: "01",
			userID:    testUserID,
			wantError: true,
			errorMsg:  "无效的考试ID",
		},
		{
			name:      "无效的用户ID-零值",
			examID:    testExamID,
			newStatus: "01",
			userID:    0,
			wantError: true,
			errorMsg:  "无效的用户ID",
		},
		{
			name:      "无效的用户ID-负值",
			examID:    testExamID,
			newStatus: "01",
			userID:    -1,
			wantError: true,
			errorMsg:  "无效的用户ID",
		},
		{
			name:      "空的状态值",
			examID:    testExamID,
			newStatus: "",
			userID:    testUserID,
			wantError: true,
			errorMsg:  "更新状态不能为空",
		},
		{
			name:       "数据库执行错误",
			examID:     testExamID,
			newStatus:  "01",
			userID:     testUserID,
			forceError: "tx.Exec",
			wantError:  true,
			errorMsg:   "force error",
		},
		{
			name:      "不存在的考试ID",
			examID:    999999,
			newStatus: "01",
			userID:    testUserID,
			wantError: false, // SQL执行成功但影响行数为0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 如果需要验证，先记录更新前的状态
			var originalStatus string
			if tt.shouldVerify && !tt.wantError {
				err := tx.QueryRow(ctx, "SELECT status FROM t_exam_info WHERE id = $1", tt.examID).Scan(&originalStatus)
				if err != nil {
					t.Fatalf("获取更新前状态失败: %v", err)
				}
			}

			// 创建测试上下文
			testCtx := ctx
			if tt.forceError != "" {
				testCtx = context.WithValue(ctx, "force-error", tt.forceError)
			}

			// 执行更新操作
			err := updateExamStatus(testCtx, tx, tt.examID, tt.newStatus, tt.userID)

			// 检查错误
			if tt.wantError {
				if err == nil {
					t.Errorf("updateExamStatus() 期望返回错误，但实际没有错误")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("updateExamStatus() 错误信息 = %v, 期望包含 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("updateExamStatus() 期望没有错误，但返回错误: %v", err)
					return
				}

				// 验证更新结果
				if tt.shouldVerify {
					var currentStatus string
					err := tx.QueryRow(ctx, "SELECT status FROM t_exam_info WHERE id = $1", tt.examID).Scan(&currentStatus)
					if err != nil {
						t.Errorf("验证更新结果失败: %v", err)
						return
					}

					if currentStatus != tt.newStatus {
						t.Errorf("updateExamStatus() 状态更新失败，期望状态 = %v, 实际状态 = %v", tt.newStatus, currentStatus)
					}

					// 验证 update_time 和 updated_by 字段
					var updatedBy int64
					var updateTime int64
					err = tx.QueryRow(ctx, "SELECT updated_by, update_time FROM t_exam_info WHERE id = $1", tt.examID).Scan(&updatedBy, &updateTime)
					if err != nil {
						t.Errorf("验证更新字段失败: %v", err)
					} else {
						if updatedBy != tt.userID {
							t.Errorf("updateExamStatus() updated_by 字段错误，期望 = %v, 实际 = %v", tt.userID, updatedBy)
						}
						// 验证 update_time 是最近更新的（容忍1分钟误差）
						if time.Since(time.UnixMilli(updateTime)) > time.Minute {
							t.Errorf("updateExamStatus() update_time 字段未正确更新，时间 = %v", updateTime)
						}
					}
				}
			}
		})
	}
}

// TestUpdateExamSessionStatus 测试 updateExamSessionStatus 函数
func TestUpdateExamSessionStatus(t *testing.T) {
	// 初始化配置
	cmn.ConfigureForTest()

	ctx := context.Background()

	// 创建测试用的事务
	conn := cmn.GetPgxConn()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("创建事务失败: %v", err)
	}
	defer tx.Rollback(ctx)

	// 准备测试数据 - 先插入考试信息，再插入考试场次
	testExamID := int64(999002)
	testSessionID := int64(999201)
	testUserID := int64(1)
	testExamName := "测试考试_updateSessionStatus"

	// 插入测试考试数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_info (id, name, status, creator, create_time, updated_by, update_time)
		VALUES ($1, $2, '01', $3, $4, $3, $4)
	`, testExamID, testExamName, testUserID, time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("插入测试考试数据失败: %v", err)
	}

	// 插入多个测试考试场次数据
	testSessionID2 := int64(999202)
	currentTime := time.Now().UnixMilli()

	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_session (id, exam_id, paper_id, mark_method, session_num, status, creator, create_time, updated_by, update_time)
		VALUES 
			($1, $2, 1, '00', 1, '00', $3, $4, $3, $4),
			($5, $2, 1, '00', 2, '00', $3, $4, $3, $4)
	`, testSessionID, testExamID, testUserID, currentTime, testSessionID2)
	if err != nil {
		t.Fatalf("插入测试考试场次数据失败: %v", err)
	}

	tests := []struct {
		name         string
		examID       int64
		newStatus    string
		userID       int64
		forceError   string
		wantError    bool
		errorMsg     string
		shouldVerify bool
	}{
		{
			name:         "正常更新考试场次状态-待开始到进行中",
			examID:       testExamID,
			newStatus:    "01",
			userID:       testUserID,
			forceError:   "",
			wantError:    false,
			shouldVerify: true,
		},
		{
			name:         "正常更新考试场次状态-进行中到已结束",
			examID:       testExamID,
			newStatus:    "02",
			userID:       testUserID,
			forceError:   "",
			wantError:    false,
			shouldVerify: true,
		},
		{
			name:      "无效的考试ID-零值",
			examID:    0,
			newStatus: "01",
			userID:    testUserID,
			wantError: true,
			errorMsg:  "无效的考试ID",
		},
		{
			name:      "无效的考试ID-负值",
			examID:    -1,
			newStatus: "01",
			userID:    testUserID,
			wantError: true,
			errorMsg:  "无效的考试ID",
		},
		{
			name:      "无效的用户ID-零值",
			examID:    testExamID,
			newStatus: "01",
			userID:    0,
			wantError: true,
			errorMsg:  "无效的用户ID",
		},
		{
			name:      "无效的用户ID-负值",
			examID:    testExamID,
			newStatus: "01",
			userID:    -1,
			wantError: true,
			errorMsg:  "无效的用户ID",
		},
		{
			name:      "空的状态值",
			examID:    testExamID,
			newStatus: "",
			userID:    testUserID,
			wantError: true,
			errorMsg:  "更新状态不能为空",
		},
		{
			name:       "数据库执行错误",
			examID:     testExamID,
			newStatus:  "01",
			userID:     testUserID,
			forceError: "tx.Exec",
			wantError:  true,
			errorMsg:   "force error",
		},
		{
			name:      "不存在的考试场次ID",
			examID:    999999,
			newStatus: "01",
			userID:    testUserID,
			wantError: false, // SQL执行成功但影响行数为0
		},
		{
			name:         "更新为暂停状态",
			examID:       testExamID,
			newStatus:    "03",
			userID:       testUserID,
			wantError:    false,
			shouldVerify: true,
		},
		{
			name:         "更新为取消状态",
			examID:       testExamID,
			newStatus:    "04",
			userID:       testUserID,
			wantError:    false,
			shouldVerify: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 如果需要验证，先记录更新前所有场次的状态
			var originalStatuses []string
			if tt.shouldVerify && !tt.wantError {
				rows, err := tx.Query(ctx, "SELECT status FROM t_exam_session WHERE exam_id = $1 ORDER BY session_num", tt.examID)
				if err != nil {
					t.Fatalf("获取更新前状态失败: %v", err)
				}
				defer rows.Close()

				for rows.Next() {
					var status string
					if err := rows.Scan(&status); err != nil {
						t.Fatalf("扫描原始状态失败: %v", err)
					}
					originalStatuses = append(originalStatuses, status)
				}
			}

			// 创建测试上下文
			testCtx := ctx
			if tt.forceError != "" {
				testCtx = context.WithValue(ctx, "force-error", tt.forceError)
			}

			// 执行更新操作
			err := updateExamSessionStatus(testCtx, tx, tt.examID, tt.newStatus, tt.userID)

			// 检查错误
			if tt.wantError {
				if err == nil {
					t.Errorf("updateExamSessionStatus() 期望返回错误，但实际没有错误")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("updateExamSessionStatus() 错误信息 = %v, 期望包含 %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("updateExamSessionStatus() 期望没有错误，但返回错误: %v", err)
					return
				}

				// 验证更新结果
				if tt.shouldVerify {
					// 检查所有场次的状态是否都被更新
					rows, err := tx.Query(ctx, "SELECT status, updated_by, update_time FROM t_exam_session WHERE exam_id = $1 ORDER BY session_num", tt.examID)
					if err != nil {
						t.Errorf("验证更新结果查询失败: %v", err)
						return
					}
					defer rows.Close()

					var sessionCount int
					for rows.Next() {
						var currentStatus string
						var updatedBy int64
						var updateTime int64

						if err := rows.Scan(&currentStatus, &updatedBy, &updateTime); err != nil {
							t.Errorf("验证更新结果扫描失败: %v", err)
							return
						}

						sessionCount++

						// 验证状态是否正确更新
						if currentStatus != tt.newStatus {
							t.Errorf("updateExamSessionStatus() 场次%d状态更新失败，期望状态 = %v, 实际状态 = %v", sessionCount, tt.newStatus, currentStatus)
						}

						// 验证 updated_by 字段
						if updatedBy != tt.userID {
							t.Errorf("updateExamSessionStatus() 场次%d updated_by 字段错误，期望 = %v, 实际 = %v", sessionCount, tt.userID, updatedBy)
						}

						// 验证 update_time 是最近更新的（容忍1分钟误差）
						if time.Since(time.UnixMilli(updateTime)) > time.Minute {
							t.Errorf("updateExamSessionStatus() 场次%d update_time 字段未正确更新，时间 = %v", sessionCount, updateTime)
						}
					}

					// 验证是否更新了所有场次（至少应该有2个场次）
					if sessionCount != len(originalStatuses) {
						t.Errorf("updateExamSessionStatus() 场次数量不匹配，期望更新 %d 个场次，实际更新了 %d 个", len(originalStatuses), sessionCount)
					}
				}
			}
		})
	}
}

// TestExamList 测试 examList 函数
func TestExamList(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 准备测试数据
	conn := cmn.GetPgxConn()
	ctx := context.Background()

	// 测试数据ID
	testExamID1 := int64(999901)
	testExamID2 := int64(999902)
	testSessionID1 := int64(999911)
	testSessionID2 := int64(999912)
	testUserID := int64(1001)
	testStudentID := int64(1002)

	// 设置测试数据
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("创建事务失败: %v", err)
	}

	currentTime := time.Now().UnixMilli()

	// 插入考试信息
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_info (id, name, type, mode, status, creator, create_time, updated_by, update_time)
		VALUES ($1, '测试考试1', '00', '00', '02', $2, $3, $2, $3),
		       ($4, '测试考试2', '02', '02', '04', $2, $3, $2, $3)
	`, testExamID1, testUserID, currentTime, testExamID2)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试考试数据失败: %v", err)
	}

	// 插入考试场次
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_session (id, exam_id, paper_id, mark_method, mark_mode, session_num, start_time, end_time, duration, status, creator, create_time, updated_by, update_time)
		VALUES ($1, $2, 1, '00', '00', 1, $5, $6, 120, '02', $3, $4, $3, $4),
		       ($7, $8, 1, '00', '00', 1, $5, $6, 90, '04', $3, $4, $3, $4)
	`, testSessionID1, testExamID1, testUserID, currentTime, currentTime+3600000, currentTime+7200000, testSessionID2, testExamID2)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试场次数据失败: %v", err)
	}

	// 插入考生数据（为学生角色测试准备）
	_, err = tx.Exec(ctx, `
		INSERT INTO t_examinee (exam_session_id, student_id, status, creator, create_time)
		VALUES ($1, $2, '01', $3, $4)
	`, testSessionID1, testStudentID, testUserID, currentTime)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试考生数据失败: %v", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		t.Fatalf("提交测试数据失败: %v", err)
	}

	// 清理函数
	t.Cleanup(func() {
		tx, err := conn.Begin(ctx)
		if err != nil {
			return
		}
		defer tx.Rollback(ctx)

		tx.Exec(ctx, "DELETE FROM t_examinee WHERE exam_session_id IN ($1, $2)", testSessionID1, testSessionID2)
		tx.Exec(ctx, "DELETE FROM t_exam_session WHERE id IN ($1, $2)", testSessionID1, testSessionID2)
		tx.Exec(ctx, "DELETE FROM t_exam_info WHERE id IN ($1, $2)", testExamID1, testExamID2)

		tx.Commit(ctx)
	})

	tests := []struct {
		name          string
		method        string
		queryParams   string
		userID        int64
		userRole      int64
		forceError    string
		expectedError bool
		errorContains string
		description   string
		checkResult   func(t *testing.T, serviceCtx *cmn.ServiceCtx)
	}{
		{
			name:          "教师角色-默认查询",
			method:        "GET",
			queryParams:   "",
			userID:        testUserID,
			userRole:      2, // 教师角色
			expectedError: false,
			description:   "教师角色使用默认查询参数获取考试列表",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var examList []ExamList
				if err := json.Unmarshal(serviceCtx.Msg.Data, &examList); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
				if len(examList) == 0 {
					t.Error("期望返回考试列表，但为空")
				}
				if serviceCtx.Msg.RowCount <= 0 {
					t.Error("期望返回行数大于0")
				}
			},
		},
		{
			name:          "教师角色-默认查询 Query错误",
			method:        "GET",
			queryParams:   "",
			userID:        testUserID,
			userRole:      2, // 教师角色
			expectedError: true,
			forceError:    "conn.Query",
			description:   "教师角色-默认查询 Query错误",
		},
		{
			name:          "教师角色-默认查询 Scan错误",
			method:        "GET",
			queryParams:   "",
			userID:        testUserID,
			userRole:      2, // 教师角色
			expectedError: true,
			forceError:    "rows.Scan",
			description:   "教师角色-默认查询 Scan错误",
		},
		{
			name:          "学生角色-查询自己的考试",
			method:        "GET",
			queryParams:   "",
			userID:        testStudentID,
			userRole:      1, // 学生角色
			expectedError: false,
			description:   "学生角色查询自己参与的考试",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var examList []ExamList
				if err := json.Unmarshal(serviceCtx.Msg.Data, &examList); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
				// 学生应该至少能看到一个自己参与的考试
				if len(examList) == 0 {
					t.Error("学生应该能看到自己参与的考试")
				}
			},
		},
		{
			name:          "学生角色-查询自己的考试 Query错误",
			method:        "GET",
			queryParams:   "",
			userID:        testStudentID,
			userRole:      1, // 学生角色
			expectedError: true,
			forceError:    "conn.Query",
			description:   "学生角色-查询自己的考试 Query错误",
		},
		{
			name:          "学生角色-查询自己的考试 Scan错误",
			method:        "GET",
			queryParams:   "",
			userID:        testStudentID,
			userRole:      1, // 学生角色
			expectedError: true,
			forceError:    "rows.Scan",
			description:   "学生角色-查询自己的考试 Scan错误",
		},
		{
			name:          "管理员角色-查询所有考试",
			method:        "GET",
			queryParams:   "",
			userID:        testUserID,
			userRole:      3, // 管理员角色
			expectedError: false,
			description:   "管理员角色查询所有考试",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
			},
		},
		{
			name:          "自定义查询-按名称过滤",
			method:        "GET",
			queryParams:   `q={"Action":"select","OrderBy":[{"ID":"DESC"}],"Filter":{"Name":"测试考试1","Status":"","StartTime":0,"EndTime":0},"Page":1,"PageSize":10}`,
			userID:        testUserID,
			userRole:      2,
			expectedError: false,
			description:   "按考试名称过滤查询",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var examList []ExamList
				if err := json.Unmarshal(serviceCtx.Msg.Data, &examList); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
				// 验证返回的考试名称包含过滤条件
				for _, exam := range examList {
					if !strings.Contains(exam.ExamName, "测试考试1") {
						t.Errorf("返回的考试名称不符合过滤条件: %s", exam.ExamName)
					}
				}
			},
		},
		{
			name:          "自定义查询-按时间过滤",
			method:        "GET",
			queryParams:   `q={"Action":"select","OrderBy":[{"ID":"DESC"}],"Filter":{"Name":"","Status":"","StartTime":` + strconv.FormatInt(currentTime+3600000, 10) + `,"EndTime":` + strconv.FormatInt(currentTime+7200000, 10) + `},"Page":1,"PageSize":10}`,
			userID:        testUserID,
			userRole:      2,
			expectedError: false,
			description:   "按考试时间过滤查询",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var examList []ExamList
				if err := json.Unmarshal(serviceCtx.Msg.Data, &examList); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
			},
		},
		{
			name:          "自定义查询-按状态过滤",
			method:        "GET",
			queryParams:   `q={"Action":"select","OrderBy":[{"ID":"DESC"}],"Filter":{"Name":"","Status":"02","StartTime":0,"EndTime":0},"Page":1,"PageSize":10}`,
			userID:        testUserID,
			userRole:      2,
			expectedError: false,
			description:   "按考试状态过滤查询",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var examList []ExamList
				if err := json.Unmarshal(serviceCtx.Msg.Data, &examList); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
				// 验证返回的考试状态符合过滤条件
				for _, exam := range examList {
					if exam.Status != "02" {
						t.Errorf("返回的考试状态不符合过滤条件: %s", exam.Status)
					}
				}
			},
		},
		{
			name:          "自定义查询-分页测试",
			method:        "GET",
			queryParams:   `q={"Action":"select","OrderBy":[{"ID":"DESC"}],"Filter":{"Name":"","Status":"","StartTime":0,"EndTime":0},"Page":1,"PageSize":1}`,
			userID:        testUserID,
			userRole:      2,
			expectedError: false,
			description:   "测试分页功能",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var examList []ExamList
				if err := json.Unmarshal(serviceCtx.Msg.Data, &examList); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
				// 由于PageSize设置为1，返回的考试数量应该不超过1
				if len(examList) > 1 {
					t.Errorf("分页测试失败，期望最多返回1个考试，实际返回%d个", len(examList))
				}
			},
		},
		{
			name:          "自定义查询-分页测试",
			method:        "GET",
			queryParams:   `q={"Action":"select","OrderBy":[{"ID":"DESC"}],"Filter":{"Name":"","Status":"","StartTime":0,"EndTime":0},"Page":-1,"PageSize":1}`,
			userID:        testUserID,
			userRole:      2,
			expectedError: false,
			description:   "测试分页功能",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var examList []ExamList
				if err := json.Unmarshal(serviceCtx.Msg.Data, &examList); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
				// 由于PageSize设置为1，返回的考试数量应该不超过1
				if len(examList) > 1 {
					t.Errorf("分页测试失败，期望最多返回1个考试，实际返回%d个", len(examList))
				}
			},
		},
		{
			name:          "自定义查询-分页测试",
			method:        "GET",
			queryParams:   `q={"Action":"select","OrderBy":[{"ID":"DESC"}],"Filter":{"Name":"","Status":"","StartTime":0,"EndTime":0},"Page":0,"PageSize":-1}`,
			userID:        testUserID,
			userRole:      2,
			expectedError: false,
			description:   "测试分页功能",
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回数据，但数据为空")
					return
				}
				var examList []ExamList
				if err := json.Unmarshal(serviceCtx.Msg.Data, &examList); err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}
				// 由于PageSize设置为1，返回的考试数量应该不超过1
				if len(examList) > 1 {
					t.Errorf("分页测试失败，期望最多返回1个考试，实际返回%d个", len(examList))
				}
			},
		},
		{
			name:          "无效JSON查询参数",
			method:        "GET",
			queryParams:   `q={"invalid json}`,
			userID:        testUserID,
			userRole:      2,
			expectedError: true,
			description:   "无效的JSON查询参数应返回错误",
		},
		{
			name:          "无效的用户ID-零值",
			method:        "GET",
			queryParams:   "",
			userID:        0,
			userRole:      2,
			expectedError: true,
			errorContains: "无效的用户ID",
			description:   "用户ID为0应返回错误",
		},
		{
			name:          "无效的用户ID-负值",
			method:        "GET",
			queryParams:   "",
			userID:        -1,
			userRole:      2,
			expectedError: true,
			errorContains: "无效的用户ID",
			description:   "用户ID为负值应返回错误",
		},
		{
			name:          "无效的用户角色-零值",
			method:        "GET",
			queryParams:   "",
			userID:        testUserID,
			userRole:      0,
			expectedError: true,
			errorContains: "无效的用户角色",
			description:   "用户角色为0应返回错误",
		},
		{
			name:          "无效的用户角色-负值",
			method:        "GET",
			queryParams:   "",
			userID:        testUserID,
			userRole:      -1,
			expectedError: true,
			errorContains: "无效的用户角色",
			description:   "用户角色为负值应返回错误",
		},
		{
			name:          "时间范围错误-开始时间晚于结束时间",
			method:        "GET",
			queryParams:   `q={"Action":"select","Filter":{"Name":"","Status":"","StartTime":2000000000000,"EndTime":1000000000000},"Page":1,"PageSize":10}`,
			userID:        testUserID,
			userRole:      2,
			expectedError: true,
			errorContains: "开始时间不能晚于结束时间",
			description:   "开始时间晚于结束时间应返回错误",
		},
		{
			name:          "不支持的HTTP方法",
			method:        "POST",
			queryParams:   "",
			userID:        testUserID,
			userRole:      2,
			expectedError: true,
			errorContains: "unsupported method",
			description:   "POST方法应返回不支持的错误",
		},
		{
			name:          "模拟数据库查询错误",
			method:        "GET",
			queryParams:   "",
			userID:        testUserID,
			userRole:      2,
			forceError:    "conn.QueryRow",
			expectedError: true,
			description:   "模拟数据库查询错误",
		},
		{
			name:          "模拟JSON序列化错误1",
			method:        "GET",
			queryParams:   "",
			userID:        testUserID,
			userRole:      2,
			forceError:    "json.Marshal1",
			expectedError: true,
			description:   "模拟JSON序列化错误1",
		},
		{
			name:          "模拟JSON序列化错误2",
			method:        "GET",
			queryParams:   "",
			userID:        testUserID,
			userRole:      2,
			forceError:    "json.Marshal2",
			expectedError: true,
			description:   "模拟JSON序列化错误2",
		},
		{
			name:          "模拟强制JSON解析错误1",
			method:        "GET",
			queryParams:   "",
			userID:        testUserID,
			userRole:      2,
			forceError:    "json.Unmarshal1",
			expectedError: true,
			description:   "模拟强制JSON解析错误1",
		},
		{
			name:          "模拟强制JSON解析错误2",
			method:        "GET",
			queryParams:   "",
			userID:        testUserID,
			userRole:      2,
			forceError:    "json.Unmarshal2",
			expectedError: true,
			description:   "模拟强制JSON解析错误1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 解析查询参数
			queryParams := url.Values{}
			if tt.queryParams != "" {
				if strings.HasPrefix(tt.queryParams, "q=") {
					queryParams.Set("q", tt.queryParams[2:])
				} else {
					parts := strings.Split(tt.queryParams, "&")
					for _, part := range parts {
						if kv := strings.Split(part, "="); len(kv) == 2 {
							queryParams.Set(kv[0], kv[1])
						}
					}
				}
			}

			// 创建模拟上下文
			ctx := createMockContextWithRole(tt.method, "/api/examList", queryParams, tt.forceError, tt.userID, tt.userRole)

			// 调用被测试的函数
			examList(ctx)

			// 获取ServiceCtx以检查结果
			serviceCtx := ctx.Value(cmn.QNearKey).(*cmn.ServiceCtx)

			if tt.expectedError {
				// 期望有错误
				if serviceCtx.Err == nil {
					t.Errorf("examList() 期望返回错误，但实际成功")
					return
				}

				if tt.errorContains != "" && !containsString(serviceCtx.Err.Error(), tt.errorContains) {
					t.Errorf("examList() 错误信息 = %v, 期望包含 %v", serviceCtx.Err.Error(), tt.errorContains)
				}
			} else {
				// 期望成功
				if serviceCtx.Err != nil {
					t.Errorf("examList() 期望成功，但返回错误: %v", serviceCtx.Err)
					return
				}

				// 使用自定义检查函数
				if tt.checkResult != nil {
					tt.checkResult(t, serviceCtx)
				}
			}
		})
	}
}

// TestExaminee 测试examinee函数
func TestExaminee(t *testing.T) {
	// 确保logger和数据库连接已初始化
	if z == nil {
		cmn.ConfigureForTest()
	}

	// 准备测试数据
	conn := cmn.GetPgxConn()
	ctx := context.Background()

	// 测试数据ID
	testExamID := int64(999801)
	testSessionID := int64(999811)
	testUserID := int64(1001)
	testStudentID := int64(1002)
	testExamineeID := int64(999821)

	// 设置测试数据
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("创建事务失败: %v", err)
	}

	defer func() {
		// 清理测试数据
		tx.Rollback(ctx)
		conn.Exec(ctx, "DELETE FROM t_examinee WHERE id = $1", testExamineeID)
		conn.Exec(ctx, "DELETE FROM t_exam_session WHERE id = $1", testSessionID)
		conn.Exec(ctx, "DELETE FROM t_exam_info WHERE id = $1", testExamID)
		conn.Exec(ctx, "DELETE FROM t_user WHERE id IN ($1, $2)", testUserID, testStudentID)
	}()

	currentTime := time.Now().UnixMilli()

	// 插入测试用户
	_, err = tx.Exec(ctx, `
		INSERT INTO t_user (id, role, account, category, official_name, create_time, update_time, status, id_card_no) 
		VALUES 
			($1, 2, 'test_teacher_exam', '00', '测试教师', $3, $3, '00', '110101199001011234'),
			($2, 1, 'test_student_exam', '00', '测试学生', $3, $3, '00', '110101199001011235')
		ON CONFLICT (id) DO NOTHING`,
		testUserID, testStudentID, currentTime)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试用户失败: %v", err)
	}

	// 插入考试信息
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_info (id, name, type, mode, status, creator, create_time, updated_by, update_time)
		VALUES ($1, '测试考试-考生', '00', '00', '02', $2, $3, $2, $3)
		ON CONFLICT (id) DO NOTHING
	`, testExamID, testUserID, currentTime)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试考试数据失败: %v", err)
	}

	// 插入考试场次
	_, err = tx.Exec(ctx, `
		INSERT INTO t_exam_session (id, exam_id, paper_id, mark_method, mark_mode, session_num, start_time, end_time, duration, status, creator, create_time, updated_by, update_time)
		VALUES ($1, $2, 1, '00', '00', 1, $3, $4, 120, '02', $5, $3, $5, $3)
		ON CONFLICT (id) DO NOTHING
	`, testSessionID, testExamID, currentTime, currentTime+7200000, testUserID)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试考试场次失败: %v", err)
	}

	// 插入考生数据
	_, err = tx.Exec(ctx, `
		INSERT INTO t_examinee (id, student_id, exam_session_id, serial_number, examinee_number, status, creator, create_time, updated_by, update_time)
		VALUES ($1, $2, $3, 1, '24001000001', '00', $4, $5, $4, $5)
		ON CONFLICT (id) DO NOTHING
	`, testExamineeID, testStudentID, testSessionID, testUserID, currentTime)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("插入测试考生数据失败: %v", err)
	}

	// 提交事务
	err = tx.Commit(ctx)
	if err != nil {
		t.Fatalf("提交事务失败: %v", err)
	}

	tests := []struct {
		name          string
		method        string
		queryParams   map[string]string
		userID        int64
		userRole      int64
		forceError    string
		expectedError bool
		errorContains string
		checkResult   func(t *testing.T, serviceCtx *cmn.ServiceCtx)
		description   string
		mockValues    map[string]string
	}{
		{
			name:   "GET方法-成功获取考生列表",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testExamID),
			},
			userID:        testUserID,
			userRole:      2, // 教师角色
			expectedError: false,
			checkResult: func(t *testing.T, serviceCtx *cmn.ServiceCtx) {
				if serviceCtx.Msg.Data == nil {
					t.Error("期望返回考生数据，但Data为空")
					return
				}

				var examinees []Examinee
				err := json.Unmarshal(serviceCtx.Msg.Data, &examinees)
				if err != nil {
					t.Errorf("解析返回数据失败: %v", err)
					return
				}

				// 验证数据结构
				if len(examinees) == 0 {
					t.Log("返回空的考生列表（这可能是正常的，如果没有考生数据）")
				} else {
					for _, examinee := range examinees {
						if examinee.StudentID <= 0 {
							t.Errorf("无效的学生ID: %d", examinee.StudentID)
						}
						if examinee.OfficialName == "" {
							t.Error("学生姓名不能为空")
						}
						if examinee.Account == "" {
							t.Error("学生账号不能为空")
						}
					}
				}
			},
			description: "教师角色成功获取考生列表",
		},
		{
			name:   "GET方法-获取权限失败",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testExamID),
			},
			userID:        testUserID,
			userRole:      2, // 教师角色
			expectedError: true,
			mockValues:    map[string]string{"test": "validateUserExamPermission-error"},
			description:   "获取权限失败",
		},
		{
			name:        "GET方法-缺少exam_id参数",
			method:      "GET",
			queryParams: map[string]string{
				// 不提供exam_id
			},
			userID:        testUserID,
			userRole:      2,
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "缺少exam_id参数应返回错误",
		},
		{
			name:   "GET方法-无效的exam_id参数",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": "invalid",
			},
			userID:        testUserID,
			userRole:      2,
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "无效的exam_id参数应返回错误",
		},
		{
			name:   "GET方法-exam_id为0",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": "0",
			},
			userID:        testUserID,
			userRole:      2,
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "exam_id为0应返回错误",
		},
		{
			name:   "GET方法-exam_id为负数",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": "-1",
			},
			userID:        testUserID,
			userRole:      2,
			expectedError: true,
			errorContains: "无效的考试ID",
			description:   "exam_id为负数应返回错误",
		},
		{
			name:   "GET方法-无效的用户ID",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testExamID),
			},
			userID:        0, // 无效的用户ID
			userRole:      2,
			expectedError: true,
			errorContains: "无效的用户ID",
			description:   "无效的用户ID应返回错误",
		},
		{
			name:   "GET方法-无效的用户角色",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testExamID),
			},
			userID:        testUserID,
			userRole:      0,
			expectedError: true,
			errorContains: "无效的用户角色",
			description:   "无效的用户角色应返回错误",
		},
		{
			name:   "GET方法-不存在的考试ID",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": "999999", // 不存在的考试ID
			},
			userID:        testUserID,
			userRole:      2,
			expectedError: true,
			errorContains: "无权限访问该考试",
			description:   "不存在的考试ID应返回权限错误",
		},
		{
			name:   "POST方法-不支持的方法",
			method: "POST",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testExamID),
			},
			userID:        testUserID,
			userRole:      2,
			expectedError: true,
			errorContains: "unsupported method",
			description:   "不支持的HTTP方法应返回错误",
		},
		{
			name:   "PUT方法-不支持的方法",
			method: "PUT",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testExamID),
			},
			userID:        testUserID,
			userRole:      2,
			expectedError: true,
			errorContains: "unsupported method",
			description:   "不支持的HTTP方法应返回错误",
		},
		{
			name:   "DELETE方法-不支持的方法",
			method: "DELETE",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testExamID),
			},
			userID:        testUserID,
			userRole:      2,
			expectedError: true,
			errorContains: "unsupported method",
			description:   "不支持的HTTP方法应返回错误",
		},
		{
			name:   "GET方法-模拟数据库查询错误",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testExamID),
			},
			userID:        testUserID,
			userRole:      2,
			forceError:    "conn.Query",
			expectedError: true,
			errorContains: "force error",
			description:   "模拟数据库查询错误",
		},
		{
			name:   "GET方法-模拟数据库扫描错误",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testExamID),
			},
			userID:        testUserID,
			userRole:      2,
			forceError:    "conn.Scan",
			expectedError: true,
			description:   "模拟数据库扫描错误",
		},
		{
			name:   "GET方法-模拟数据库Rows错误",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testExamID),
			},
			userID:        testUserID,
			userRole:      2,
			forceError:    "rows.Err",
			expectedError: true,
			description:   "模拟数据库扫描错误",
		},
		{
			name:   "GET方法-模拟JSON序列化错误",
			method: "GET",
			queryParams: map[string]string{
				"exam_id": fmt.Sprintf("%d", testExamID),
			},
			userID:        testUserID,
			userRole:      2,
			forceError:    "json.Marshal",
			expectedError: true,
			errorContains: "强制JSON序列化错误",
			description:   "模拟JSON序列化错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("测试用例: %s", tt.description)

			// 构建查询参数
			queryParams := url.Values{}
			for key, value := range tt.queryParams {
				queryParams.Set(key, value)
			}

			// 创建模拟上下文
			ctx := createMockContextWithRole(tt.method, "/api/examinee", queryParams, tt.forceError, tt.userID, tt.userRole)

			// 如果有模拟的错误，设置mockValues
			for key, value := range tt.mockValues {
				ctx = context.WithValue(ctx, key, value)
			}

			// 调用被测试的函数
			examinee(ctx)

			// 获取ServiceCtx以检查结果
			serviceCtx := ctx.Value(cmn.QNearKey).(*cmn.ServiceCtx)

			if tt.expectedError {
				// 期望有错误
				if serviceCtx.Err == nil {
					t.Errorf("examinee() 期望返回错误，但实际成功")
					return
				}

				if tt.errorContains != "" && !containsString(serviceCtx.Err.Error(), tt.errorContains) {
					t.Errorf("examinee() 错误信息 = %v, 期望包含 %v", serviceCtx.Err.Error(), tt.errorContains)
				}
			} else {
				// 期望成功
				if serviceCtx.Err != nil {
					t.Errorf("examinee() 期望成功，但返回错误: %v", serviceCtx.Err)
					return
				}

				// 使用自定义检查函数
				if tt.checkResult != nil {
					tt.checkResult(t, serviceCtx)
				}
			}
		})
	}
}
